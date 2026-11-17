package filestore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// PingLimit is the longest wait for the bucket before the hub reports the file store
// as not ready.
const PingLimit = time.Second

// ErrObjectAbsent reports that the bucket holds no object under the key.
var ErrObjectAbsent = errors.New("filestore: absent")

// Object holds the facts of one object that the bucket answers.
type Object struct {
	Size        int64
	ContentType string
}

// Client reaches one bucket. The hub calls the file store at the internal endpoint,
// and a browser follows each signed address to the public endpoint.
type Client struct {
	internal  *s3.Client
	presigner *s3.PresignClient
	bucket    string
	uploadTTL time.Duration
	readTTL   time.Duration
	now       func() time.Time
}

// Option changes one setting of a Client.
type Option func(*Client)

// WithClock sets the clock that signs each address and dates its expiry.
func WithClock(now func() time.Time) Option {
	return func(c *Client) { c.now = now }
}

// New creates a Client over the bucket of the configuration.
func New(cfg config.Files, opts ...Option) *Client {
	client := &Client{
		internal:  s3.New(s3.Options{}, endpointOptions(cfg, cfg.Endpoint)),
		bucket:    cfg.Bucket,
		uploadTTL: cfg.UploadTTL,
		readTTL:   cfg.ReadTTL,
		now:       time.Now,
	}

	for _, opt := range opts {
		opt(client)
	}

	client.presigner = s3.NewPresignClient(
		s3.New(s3.Options{}, endpointOptions(cfg, cfg.PublicEndpoint)),
		func(o *s3.PresignOptions) { o.Presigner = clockedPresigner{now: client.now} },
	)

	return client
}

// endpointOptions reaches the endpoint with path style, because a shared server
// answers one host for every bucket. A checksum goes only where S3 demands one, so an
// address that the hub signs asks the browser for no checksum header.
func endpointOptions(cfg config.Files, endpoint string) func(*s3.Options) {
	return func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
		options.Region = cfg.Region
		options.Credentials = credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)
		options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		options.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	}
}

// Ping reports whether the bucket answers within PingLimit.
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, PingLimit)
	defer cancel()

	if _, err := c.internal.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &c.bucket}); err != nil {
		return fmt.Errorf("reaching the bucket: %w", err)
	}

	return nil
}

// Head answers the facts of the object under the key, or ErrObjectAbsent.
func (c *Client) Head(ctx context.Context, key string) (Object, error) {
	out, err := c.internal.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &c.bucket, Key: &key})
	if err != nil {
		return Object{}, fmt.Errorf("reading the facts of an object: %w", absentOr(err))
	}

	return Object{
		Size:        aws.ToInt64(out.ContentLength),
		ContentType: aws.ToString(out.ContentType),
	}, nil
}

// ReadRange answers at most the first limit bytes of the object under the key.
func (c *Client) ReadRange(ctx context.Context, key string, limit int64) ([]byte, error) {
	out, err := c.internal.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
		Range:  aws.String(fmt.Sprintf("bytes=0-%d", limit-1)),
	})
	if err != nil {
		return nil, fmt.Errorf("reading the start of an object: %w", absentOr(err))
	}

	defer func() { _ = out.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(out.Body, limit))
	if err != nil {
		return nil, fmt.Errorf("reading the start of an object: %w", err)
	}

	return data, nil
}

// Read opens the object under the key. The caller closes the body.
func (c *Client) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := c.internal.GetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &key})
	if err != nil {
		return nil, fmt.Errorf("reading an object: %w", absentOr(err))
	}

	return out.Body, nil
}

// Write stores the data under the key with the content type.
func (c *Client) Write(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.internal.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &c.bucket,
		Key:           &key,
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   &contentType,
	})
	if err != nil {
		return fmt.Errorf("writing an object: %w", err)
	}

	return nil
}

// Delete removes the object under the key. An absent object is no failure.
func (c *Client) Delete(ctx context.Context, key string) error {
	if _, err := c.internal.DeleteObject(
		ctx,
		&s3.DeleteObjectInput{Bucket: &c.bucket, Key: &key},
	); err != nil {
		return fmt.Errorf("deleting an object: %w", err)
	}

	return nil
}

// DeletePrefix removes every object whose key starts with the prefix. It deletes one
// object at a time, because a file holds at most three, and DeleteObjects demands a
// checksum that not every S3 server reads.
func (c *Client) DeletePrefix(ctx context.Context, prefix string) error {
	pages := s3.NewListObjectsV2Paginator(c.internal, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &prefix,
	})

	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing the objects of a prefix: %w", err)
		}

		for _, object := range page.Contents {
			if err = c.Delete(ctx, aws.ToString(object.Key)); err != nil {
				return err
			}
		}
	}

	return nil
}

func absentOr(err error) error {
	var (
		notFound *types.NotFound
		noKey    *types.NoSuchKey
		response *smithyhttp.ResponseError
	)

	switch {
	case errors.As(err, &notFound), errors.As(err, &noKey):
		return ErrObjectAbsent
	case errors.As(err, &response) && response.HTTPStatusCode() == 404:
		return ErrObjectAbsent
	default:
		return err
	}
}
