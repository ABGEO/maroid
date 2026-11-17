package file

import (
	"mime"
	"slices"

	"github.com/gabriel-vasile/mimetype"
)

const (
	// sniffLength is the start of an upload that the detection of its type reads.
	sniffLength = 3072
	jpegType    = "image/jpeg"
	pngType     = "image/png"
	webpType    = "image/webp"
	heicType    = "image/heic"
	heifType    = "image/heif"
)

// sniff detects the media type of the start of an upload, and reports whether the
// file kind accepts it. HEIC is a kind of HEIF, and a browser names either one.
func sniff(head []byte, accepted []string) (string, bool) {
	detected, _, err := mime.ParseMediaType(mimetype.Detect(head).String())
	if err != nil {
		return "", false
	}

	return detected, slices.ContainsFunc(accepted, func(mediaType string) bool {
		return mediaType == detected || isHEIF(mediaType) && isHEIF(detected)
	})
}

func isHEIF(mediaType string) bool {
	return mediaType == heicType || mediaType == heifType
}
