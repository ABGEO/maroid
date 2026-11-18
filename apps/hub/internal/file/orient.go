package file

import (
	"bytes"
	"encoding/binary"
	"image"
)

// The EXIF orientations, as TIFF 6.0 numbers them.
const (
	orientationNormal     = 1
	orientationMirror     = 2
	orientationRotate180  = 3
	orientationFlip       = 4
	orientationTranspose  = 5
	orientationRotate90   = 6
	orientationTransverse = 7
	orientationRotate270  = 8
)

const (
	markerStart       = 0xD8
	markerExif        = 0xE1
	markerScan        = 0xDA
	markerSize        = 2
	segmentHeaderSize = 4
	exifPrefix        = "Exif\x00\x00"
	orientationTag    = 0x0112
	shortType         = 3
	tiffHeaderSize    = 8
	ifdCountSize      = 2
	ifdEntrySize      = 12
	ifdValueOffset    = 8
)

// The chunk layouts of PNG and of RIFF, which holds a WebP.
const (
	pngSignature    = "\x89PNG\r\n\x1a\n"
	pngLengthSize   = 4
	pngChunkHeader  = 8
	pngCRCSize      = 4
	pngExifChunk    = "eXIf"
	pngEnd          = "IEND"
	riffHeaderSize  = 12
	riffFourCCSize  = 4
	riffChunkHeader = 8
	riffWordSize    = 2
	webpExifChunk   = "EXIF"
)

// orient turns the pixels so they stand as the EXIF orientation says. A value outside
// 2 to 8 leaves them as they are.
func orient(src *image.NRGBA, orientation int) *image.NRGBA {
	if orientation <= orientationNormal || orientation > orientationRotate270 {
		return src
	}

	width, height := src.Rect.Dx(), src.Rect.Dy()

	bounds := image.Rect(0, 0, width, height)
	if orientation >= orientationTranspose {
		bounds = image.Rect(0, 0, height, width)
	}

	dst := image.NewNRGBA(bounds)

	for row := range height {
		for column := range width {
			toColumn, toRow := placeOf(orientation, column, row, width, height)
			from := src.PixOffset(src.Rect.Min.X+column, src.Rect.Min.Y+row)
			copy(dst.Pix[dst.PixOffset(toColumn, toRow):][:4], src.Pix[from:][:4])
		}
	}

	return dst
}

// placeOf answers where the pixel at the column and the row of a stored image of
// width by height lands.
func placeOf(orientation, column, row, width, height int) (int, int) {
	switch orientation {
	case orientationMirror:
		return width - 1 - column, row
	case orientationRotate180:
		return width - 1 - column, height - 1 - row
	case orientationFlip:
		return column, height - 1 - row
	case orientationTranspose:
		return row, column
	case orientationRotate90:
		return height - 1 - row, column
	case orientationTransverse:
		return height - 1 - row, width - 1 - column
	default:
		return row, width - 1 - column
	}
}

// orientationOf reads the EXIF orientation of a JPEG, a PNG, or a WebP. An image with
// no EXIF, or with an EXIF that does not parse, stands as stored. A HEIF decode applies
// its own orientation.
func orientationOf(data []byte, mediaType string) int {
	var tiff []byte

	switch mediaType {
	case jpegType:
		tiff = jpegExif(data)
	case pngType:
		tiff = pngExif(data)
	case webpType:
		tiff = webpExif(data)
	}

	if tiff == nil {
		return orientationNormal
	}

	return tiffOrientation(bytes.TrimPrefix(tiff, []byte(exifPrefix)))
}

// pngExif answers the TIFF payload of the eXIf chunk of a PNG, or nil.
func pngExif(data []byte) []byte {
	if !bytes.HasPrefix(data, []byte(pngSignature)) {
		return nil
	}

	for at := len(pngSignature); at+pngChunkHeader <= len(data); {
		length := int(binary.BigEndian.Uint32(data[at:]))
		kind := string(data[at+pngLengthSize : at+pngChunkHeader])
		start := at + pngChunkHeader

		if start+length+pngCRCSize > len(data) || kind == pngEnd {
			return nil
		}

		if kind == pngExifChunk {
			return data[start : start+length]
		}

		at = start + length + pngCRCSize
	}

	return nil
}

// webpExif answers the TIFF payload of the EXIF chunk of a WebP, or nil.
func webpExif(data []byte) []byte {
	if len(data) < riffHeaderSize || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return nil
	}

	for at := riffHeaderSize; at+riffChunkHeader <= len(data); {
		kind := string(data[at : at+riffFourCCSize])
		length := int(binary.LittleEndian.Uint32(data[at+riffFourCCSize:]))
		start := at + riffChunkHeader

		if start+length > len(data) {
			return nil
		}

		if kind == webpExifChunk {
			return data[start : start+length]
		}

		at = start + length + length%riffWordSize
	}

	return nil
}

// jpegExif answers the TIFF payload of the EXIF segment of a JPEG, or nil.
func jpegExif(data []byte) []byte {
	if !bytes.HasPrefix(data, []byte{0xFF, markerStart}) {
		return nil
	}

	for at := markerSize; ; {
		marker, payload, next, found := segmentAt(data, at)
		if !found || marker == markerScan {
			return nil
		}

		if marker == markerExif && bytes.HasPrefix(payload, []byte(exifPrefix)) {
			return payload[len(exifPrefix):]
		}

		at = next
	}
}

// segmentAt answers the marker and the payload of the JPEG segment at the offset,
// and the offset of the next one.
func segmentAt(data []byte, at int) (byte, []byte, int, bool) {
	if at+segmentHeaderSize > len(data) || data[at] != 0xFF {
		return 0, nil, 0, false
	}

	length := int(binary.BigEndian.Uint16(data[at+markerSize:]))

	end := at + markerSize + length
	if length < markerSize || end > len(data) {
		return 0, nil, 0, false
	}

	return data[at+1], data[at+segmentHeaderSize : end], end, true
}

// tiffOrientation reads the orientation from IFD0 of a TIFF payload.
func tiffOrientation(tiff []byte) int {
	order := byteOrderOf(tiff)
	if order == nil {
		return orientationNormal
	}

	ifd := int(order.Uint32(tiff[4:]))
	if ifd < tiffHeaderSize || ifd+ifdCountSize > len(tiff) {
		return orientationNormal
	}

	count := int(order.Uint16(tiff[ifd:]))

	for index := range count {
		entry := ifd + ifdCountSize + index*ifdEntrySize
		if entry+ifdEntrySize > len(tiff) {
			break
		}

		if order.Uint16(tiff[entry:]) == orientationTag &&
			order.Uint16(tiff[entry+markerSize:]) == shortType {
			return int(order.Uint16(tiff[entry+ifdValueOffset:]))
		}
	}

	return orientationNormal
}

func byteOrderOf(tiff []byte) binary.ByteOrder {
	if len(tiff) < tiffHeaderSize {
		return nil
	}

	switch string(tiff[:2]) {
	case "II":
		return binary.LittleEndian
	case "MM":
		return binary.BigEndian
	default:
		return nil
	}
}
