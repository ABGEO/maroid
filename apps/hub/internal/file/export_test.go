package file

import (
	"image"
	"time"
)

// The tests of package file_test reach the unexported functions through these.

// The EXIF orientations that the tests name.
const (
	OrientationNormal   = orientationNormal
	OrientationRotate90 = orientationRotate90
	SniffLength         = sniffLength
)

// Variants holds the variants of one image and its size after the orientation.
type Variants struct {
	Thumbnail []byte
	Display   []byte
	Width     int
	Height    int
}

func exported(held imageVariants, err error) (Variants, error) {
	return Variants{
		Thumbnail: held.thumbnail,
		Display:   held.display,
		Width:     held.width,
		Height:    held.height,
	}, err
}

func VariantsOf(data []byte, mediaType string) (Variants, error) {
	return exported(variantsOf(data, mediaType))
}

func DeriveVariants(decoded image.Image, orientation int) (Variants, error) {
	return exported(deriveVariants(decoded, orientation))
}

func Orient(src *image.NRGBA, orientation int) *image.NRGBA {
	return orient(src, orientation)
}

func Sniff(head []byte, accepted []string) (string, bool) {
	return sniff(head, accepted)
}

func ErrDamaged() error {
	return errDamaged
}

// SetTaskTimeout shortens the deadline of each task, so a test meets it at once.
func (w *Worker) SetTaskTimeout(timeout time.Duration) {
	w.taskTimeout = timeout
}
