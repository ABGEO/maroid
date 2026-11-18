package file

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/gen2brain/h265/heic"
	"golang.org/x/image/draw"
	xwebp "golang.org/x/image/webp"
)

const (
	thumbnailEdge    = 480
	displayEdge      = 2048
	thumbnailQuality = 75
	displayQuality   = 80
	// maxPixels refuses a header that would make the decode hold more memory than a
	// photo of a phone needs.
	maxPixels = 100_000_000
)

var (
	// errDamaged reports content that does not decode as its media type.
	errDamaged = errors.New("file: the content does not decode")
	// errNotImage reports a media type that the pipeline does not decode.
	errNotImage = errors.New("file: the media type is not an image")
)

// imageVariants holds the JPEG variants of one image, and its size after the
// orientation.
type imageVariants struct {
	thumbnail []byte
	display   []byte
	width     int
	height    int
}

// variantsOf decodes an image of the media type and derives its variants.
func variantsOf(data []byte, mediaType string) (imageVariants, error) {
	decoded, err := decodeImage(data, mediaType)
	if err != nil {
		return imageVariants{}, err
	}

	return deriveVariants(decoded, orientationOf(data, mediaType))
}

// decodeImage checks the size that the header declares, then decodes the pixels. A
// HEIF decode applies its own rotation and mirror.
func decodeImage(data []byte, mediaType string) (image.Image, error) {
	decodeConfig, decode, err := decoderOf(mediaType)
	if err != nil {
		return nil, err
	}

	config, err := decodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDamaged, err)
	}

	if int64(config.Width)*int64(config.Height) > maxPixels {
		return nil, fmt.Errorf("%w: %d by %d pixels", errDamaged, config.Width, config.Height)
	}

	decoded, err := decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDamaged, err)
	}

	return decoded, nil
}

func decoderOf(mediaType string) (
	func(io.Reader) (image.Config, error),
	func(io.Reader) (image.Image, error),
	error,
) {
	switch mediaType {
	case jpegType:
		return jpeg.DecodeConfig, jpeg.Decode, nil
	case pngType:
		return png.DecodeConfig, png.Decode, nil
	case webpType:
		return xwebp.DecodeConfig, xwebp.Decode, nil
	case heicType, heifType:
		return heic.DecodeConfig, func(r io.Reader) (image.Image, error) {
			return heic.Decode(r, heic.Options{AutoRotate: true, FrameSizeLimit: maxPixels})
		}, nil
	default:
		return nil, nil, fmt.Errorf("%w: %s", errNotImage, mediaType)
	}
}

// deriveVariants writes the display image at a long edge of at most 2048 pixels and
// the thumbnail at 480 pixels, and never enlarges. It turns the display image after
// the resize, because the long edge does not change with a quarter turn. Each JPEG
// comes from the pixels alone, so it carries no metadata of the upload.
func deriveVariants(decoded image.Image, orientation int) (imageVariants, error) {
	display := onWhite(orient(fit(decoded, displayEdge), orientation))
	thumbnail := fit(display, thumbnailEdge)

	width, height := decoded.Bounds().Dx(), decoded.Bounds().Dy()
	if orientation >= orientationTranspose && orientation <= orientationRotate270 {
		width, height = height, width
	}

	displayBytes, err := encodeJPEG(display, displayQuality)
	if err != nil {
		return imageVariants{}, err
	}

	thumbnailBytes, err := encodeJPEG(thumbnail, thumbnailQuality)
	if err != nil {
		return imageVariants{}, err
	}

	return imageVariants{
		thumbnail: thumbnailBytes,
		display:   displayBytes,
		width:     width,
		height:    height,
	}, nil
}

// fit answers the pixels at a long edge of at most edge, with the ratio kept.
func fit(src image.Image, edge int) *image.NRGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	switch {
	case width >= height && width > edge:
		width, height = edge, max(1, height*edge/width)
	case height > width && height > edge:
		width, height = max(1, width*edge/height), edge
	}

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))

	if width == bounds.Dx() && height == bounds.Dy() {
		draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)
	} else {
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Src, nil)
	}

	return dst
}

// onWhite lays the pixels over white, since a JPEG holds no transparency.
func onWhite(src *image.NRGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Over)

	return dst
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buffer bytes.Buffer

	if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("encoding a JPEG variant: %w", err)
	}

	return buffer.Bytes(), nil
}
