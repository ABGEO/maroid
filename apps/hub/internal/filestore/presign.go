package filestore

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Upload is a signed address of one PUT. The request carries each header. The file
// store refuses a body whose length or type differs from the signed one.
type Upload struct {
	URL       string
	Method    string
	Headers   map[string]string
	ExpiresAt time.Time
}

// Address is a signed address of one GET, and the time it stops working.
type Address struct {
	URL       string
	ExpiresAt time.Time
}

// clockedPresigner signs with the clock of the Client in place of the clock of the SDK.
type clockedPresigner struct {
	now func() time.Time
}

func (p clockedPresigner) PresignHTTP(
	ctx context.Context,
	credentials aws.Credentials,
	r *http.Request,
	payloadHash string,
	service string,
	region string,
	_ time.Time,
	optFns ...func(*v4.SignerOptions),
) (string, http.Header, error) {
	url, headers, err := v4.NewSigner().PresignHTTP(
		ctx, credentials, r, payloadHash, service, region, p.now(), optFns...,
	)
	if err != nil {
		return "", nil, fmt.Errorf("signing an address: %w", err)
	}

	return url, headers, nil
}

// PresignPut signs a PUT of exactly size bytes of the content type under the key.
func (c *Client) PresignPut(
	ctx context.Context,
	key string,
	size int64,
	contentType string,
) (Upload, error) {
	signed, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        &c.bucket,
		Key:           &key,
		ContentLength: aws.Int64(size),
		ContentType:   &contentType,
	}, s3.WithPresignExpires(c.uploadTTL))
	if err != nil {
		return Upload{}, fmt.Errorf("signing an upload: %w", err)
	}

	headers := make(map[string]string, len(signed.SignedHeader))

	for name, values := range signed.SignedHeader {
		name = http.CanonicalHeaderKey(name)
		if len(values) == 0 || browserSets(name) {
			continue
		}

		headers[name] = values[0]
	}

	return Upload{
		URL:       signed.URL,
		Method:    signed.Method,
		Headers:   headers,
		ExpiresAt: c.now().Add(c.uploadTTL),
	}, nil
}

// PresignGet signs a GET of the object under the key. A non-empty name makes the
// browser save the object under that name in place of showing it.
func (c *Client) PresignGet(
	ctx context.Context,
	key string,
	attachmentName string,
) (Address, error) {
	input := &s3.GetObjectInput{Bucket: &c.bucket, Key: &key}

	if attachmentName != "" {
		input.ResponseContentDisposition = aws.String(
			mime.FormatMediaType("attachment", map[string]string{"filename": attachmentName}),
		)
	}

	signed, err := c.presigner.PresignGetObject(ctx, input, s3.WithPresignExpires(c.readTTL))
	if err != nil {
		return Address{}, fmt.Errorf("signing a read: %w", err)
	}

	return Address{URL: signed.URL, ExpiresAt: c.now().Add(c.readTTL)}, nil
}

// browserSets reports a signed header that a browser sends itself and refuses to take
// from a script. The browser sends the real length, so the signature still holds it.
func browserSets(name string) bool {
	return name == "Host" || name == "Content-Length"
}
