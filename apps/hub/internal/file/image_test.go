package file_test

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"slices"
	"testing"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gen2brain/h265/heic"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/file"
)

const (
	pngType        = "image/png"
	heicType       = "image/heic"
	heifType       = "image/heif"
	cameraMaker    = "MaroidCam"
	orientationTag = 0x0112
	shortType      = 3
	markerExif     = 0xE1
	exifPrefix     = "Exif\x00\x00"
)

func red() color.NRGBA   { return color.NRGBA{R: 255, A: 255} }
func green() color.NRGBA { return color.NRGBA{G: 255, A: 255} }
func blue() color.NRGBA  { return color.NRGBA{B: 255, A: 255} }

func filled(width, height int, paint func(x, y int) color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for y := range height {
		for x := range width {
			img.SetNRGBA(x, y, paint(x, y))
		}
	}

	return img
}

func plain(width, height int) *image.NRGBA {
	return filled(width, height, func(int, int) color.NRGBA { return blue() })
}

// tiffWith answers a big endian TIFF payload whose IFD0 holds the orientation and
// the make of the camera.
func tiffWith(orientation uint16) []byte {
	var tiff bytes.Buffer

	tiff.WriteString("MM\x00\x2a")
	_ = binary.Write(&tiff, binary.BigEndian, uint32(8))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(2))
	_ = binary.Write(&tiff, binary.BigEndian, []uint16{0x010F, 2})
	_ = binary.Write(&tiff, binary.BigEndian, uint32(len(cameraMaker)+1))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(8+2+2*12+4))
	_ = binary.Write(&tiff, binary.BigEndian, []uint16{orientationTag, shortType})
	_ = binary.Write(&tiff, binary.BigEndian, uint32(1))
	_ = binary.Write(&tiff, binary.BigEndian, []uint16{orientation, 0})
	_ = binary.Write(&tiff, binary.BigEndian, uint32(0))
	tiff.WriteString(cameraMaker + "\x00")

	return tiff.Bytes()
}

// jpegWith encodes the image as a JPEG with an EXIF segment that holds the
// orientation and the make of the camera.
func jpegWith(t *testing.T, img image.Image, orientation uint16) []byte {
	t.Helper()

	var encoded bytes.Buffer

	require.NoError(t, jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 90}))

	payload := append([]byte(exifPrefix), tiffWith(orientation)...)
	length := uint16(len(payload) + 2) //nolint:gosec // the payload holds a few dozen bytes
	segment := binary.BigEndian.AppendUint16([]byte{0xFF, markerExif}, length)
	segment = append(segment, payload...)

	data := encoded.Bytes()

	return append(append(append([]byte{}, data[:2]...), segment...), data[2:]...)
}

// pngWith encodes the image as a PNG with an eXIf chunk after the header, which holds
// the orientation.
func pngWith(t *testing.T, img image.Image, orientation uint16) []byte {
	t.Helper()

	var encoded bytes.Buffer

	require.NoError(t, png.Encode(&encoded, img))

	const afterHeader = 8 + 8 + 13 + 4

	body := append([]byte("eXIf"), tiffWith(orientation)...)
	//nolint:gosec // the payload holds a few dozen bytes
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(body)-4))
	chunk = append(chunk, body...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(body))

	data := encoded.Bytes()

	return slices.Concat(data[:afterHeader], chunk, data[afterHeader:])
}

// webpWith wraps the WebP fixture of 400 by 300 pixels as an extended WebP whose EXIF
// chunk holds the orientation. The hub decodes a WebP, and encodes none.
func webpWith(t *testing.T, orientation uint16) []byte {
	t.Helper()

	encoded, err := os.ReadFile("testdata/textured.webp")
	require.NoError(t, err)

	const riffHeader, extendedSize, exifFlag = 12, 10, 0x08

	littleEndian24 := func(value int) []byte {
		//nolint:gosec // each byte keeps 8 bits of the edge on purpose
		return []byte{byte(value), byte(value >> 8), byte(value >> 16)}
	}

	bounds := image.Rect(0, 0, 400, 300)
	extended := binary.LittleEndian.AppendUint32([]byte("VP8X"), extendedSize)
	extended = append(extended, exifFlag, 0, 0, 0)
	extended = append(extended, littleEndian24(bounds.Dx()-1)...)
	extended = append(extended, littleEndian24(bounds.Dy()-1)...)

	tiff := tiffWith(orientation)
	//nolint:gosec // the payload holds a few dozen bytes
	exif := binary.LittleEndian.AppendUint32([]byte("EXIF"), uint32(len(tiff)))
	exif = append(exif, tiff...)

	if len(tiff)%2 == 1 {
		exif = append(exif, 0)
	}

	body := slices.Concat([]byte("WEBP"), extended, encoded[riffHeader:], exif)
	//nolint:gosec // a test image holds far less than 4 GB
	riff := binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(len(body)))

	return append(riff, body...)
}

func sizeOf(t *testing.T, encoded []byte) (int, int) {
	t.Helper()

	config, err := jpeg.DecodeConfig(bytes.NewReader(encoded))
	require.NoError(t, err)

	return config.Width, config.Height
}

func requireNear(t *testing.T, want color.NRGBA, got color.Color, where string) {
	t.Helper()

	r, g, b, _ := got.RGBA()
	wr, wg, wb, _ := want.RGBA()

	for _, pair := range [][2]uint32{{r, wr}, {g, wg}, {b, wb}} {
		diff := int(pair[0]>>8) - int(pair[1]>>8)
		require.LessOrEqual(t, max(diff, -diff), 40, "%s: got %v, want %v", where, got, want)
	}
}

// FILES-SC-014: A large image gives a thumbnail of 480 and a display image of 2048
// pixels on the long edge, with the ratio kept.
func TestALargeImageShrinksToBothEdges(t *testing.T) {
	t.Parallel()

	variants, err := file.DeriveVariants(plain(8000, 6000), file.OrientationNormal)
	require.NoError(t, err)

	width, height := sizeOf(t, variants.Thumbnail)
	require.Equal(t, [2]int{480, 360}, [2]int{width, height})

	width, height = sizeOf(t, variants.Display)
	require.Equal(t, [2]int{2048, 1536}, [2]int{width, height})
	require.Equal(t, [2]int{8000, 6000}, [2]int{variants.Width, variants.Height})
}

// FILES-SC-015: A small image keeps its size in both variants.
func TestASmallImageIsNeverEnlarged(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer

	require.NoError(t, png.Encode(&encoded, plain(300, 200)))

	variants, err := file.VariantsOf(encoded.Bytes(), pngType)
	require.NoError(t, err)

	for _, variant := range [][]byte{variants.Thumbnail, variants.Display} {
		width, height := sizeOf(t, variant)
		require.Equal(t, [2]int{300, 200}, [2]int{width, height})
	}
}

// FILES-SC-016: A JPEG with the EXIF orientation 6 stands upright in its variants,
// and the top row of the display image is the left column of the stored pixels.
func TestAJPEGStandsAsItsOrientationSays(t *testing.T) {
	t.Parallel()

	stored := filled(4000, 3000, func(x, y int) color.NRGBA {
		switch {
		case x >= 2000:
			return blue()
		case y < 1500:
			return red()
		default:
			return green()
		}
	})

	variants, err := file.VariantsOf(jpegWith(t, stored, file.OrientationRotate90), jpegType)
	require.NoError(t, err)
	require.Equal(t, [2]int{3000, 4000}, [2]int{variants.Width, variants.Height})

	display, err := jpeg.Decode(bytes.NewReader(variants.Display))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 1536, 2048), display.Bounds())

	requireNear(t, green(), display.At(20, 20), "the left of the top row")
	requireNear(t, red(), display.At(1515, 20), "the right of the top row")
	requireNear(t, blue(), display.At(768, 2028), "the bottom")
}

// FILES-DD-007: The EXIF orientation of a PNG and of a WebP turns their variants too.
func TestAPNGAndAWebPStandAsTheirOrientationSays(t *testing.T) {
	t.Parallel()

	stored := textured(400, 300)

	for mediaType, data := range map[string][]byte{
		pngType:  pngWith(t, stored, file.OrientationRotate90),
		webpType: webpWith(t, file.OrientationRotate90),
	} {
		variants, err := file.VariantsOf(data, mediaType)
		require.NoError(t, err, mediaType)
		require.Equal(t, [2]int{300, 400}, [2]int{variants.Width, variants.Height}, mediaType)

		width, height := sizeOf(t, variants.Display)
		require.Equal(t, [2]int{300, 400}, [2]int{width, height}, mediaType)
	}
}

// FILES-DD-007: A transparent pixel lands on white, since a JPEG holds no
// transparency.
func TestATransparentImageLandsOnWhite(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer

	require.NoError(t, png.Encode(&encoded, filled(64, 48, func(int, int) color.NRGBA {
		return color.NRGBA{}
	})))

	variants, err := file.VariantsOf(encoded.Bytes(), pngType)
	require.NoError(t, err)

	display, err := jpeg.Decode(bytes.NewReader(variants.Display))
	require.NoError(t, err)
	requireNear(t, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, display.At(32, 24), "the center")
}

// FILES-SC-036: The thumbnail of a phone photo, of a portrait, and of a small PNG
// holds at most 100 KB.
func TestEachThumbnailHoldsAtMost100KB(t *testing.T) {
	t.Parallel()

	const thumbnailLimit = 100 << 10

	var phone, small bytes.Buffer

	require.NoError(t, heic.Encode(&phone, textured(4032, 3024)))
	require.NoError(t, png.Encode(&small, textured(300, 200)))

	uploads := map[string]struct {
		data      []byte
		mediaType string
	}{
		"phone.heic":   {phone.Bytes(), heicType},
		"portrait.jpg": {jpegWith(t, textured(3000, 4000), file.OrientationNormal), jpegType},
		"small.png":    {small.Bytes(), pngType},
	}

	for name, upload := range uploads {
		variants, err := file.VariantsOf(upload.data, upload.mediaType)
		require.NoError(t, err, name)
		require.LessOrEqual(t, len(variants.Thumbnail), thumbnailLimit, name)
	}
}

// FILES-DD-007: Each EXIF orientation from 1 to 8 places a pixel where TIFF 6.0 says.
func TestEachOrientationPlacesTheCorner(t *testing.T) {
	t.Parallel()

	stored := filled(3, 2, func(x, y int) color.NRGBA {
		if x == 0 && y == 0 {
			return red()
		}

		return blue()
	})

	for orientation, corner := range map[int]image.Point{
		1: {0, 0},
		2: {2, 0},
		3: {2, 1},
		4: {0, 1},
		5: {0, 0},
		6: {1, 0},
		7: {1, 2},
		8: {0, 2},
	} {
		turned := file.Orient(stored, orientation)
		require.Equal(t, red(), turned.NRGBAAt(corner.X, corner.Y), "orientation %d", orientation)
	}
}

// FILES-SC-018: The variants of a HEIC photo are JPEG.
func TestTheVariantsOfAHEICAreJPEG(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer

	require.NoError(t, heic.Encode(&encoded, plain(640, 480)))

	variants, err := file.VariantsOf(encoded.Bytes(), heicType)
	require.NoError(t, err)
	require.Equal(t, [2]int{640, 480}, [2]int{variants.Width, variants.Height})

	for _, variant := range [][]byte{variants.Thumbnail, variants.Display} {
		require.Equal(t, jpegType, mimetype.Detect(variant).String())
	}
}

// FILES-DD-007: A variant comes from the pixels alone, so the EXIF of a JPEG or of a
// HEIC never reaches it.
func TestNoVariantCarriesTheMetadataOfTheUpload(t *testing.T) {
	t.Parallel()

	var withExif bytes.Buffer

	require.NoError(t, heic.Encode(&withExif, plain(64, 48), heic.EncodeOptions{Exif: tiffWith(1)}))

	uploads := map[string][]byte{
		jpegType: jpegWith(t, plain(64, 48), file.OrientationNormal),
		heicType: withExif.Bytes(),
	}

	for mediaType, data := range uploads {
		require.Contains(t, string(data), cameraMaker, mediaType)

		variants, err := file.VariantsOf(data, mediaType)
		require.NoError(t, err)

		for _, variant := range [][]byte{variants.Thumbnail, variants.Display} {
			require.NotContains(t, string(variant), cameraMaker, mediaType)
			require.NotContains(t, string(variant), "EXIF", mediaType)
		}
	}
}

// FILES-DD-007: A header that declares more than 100 megapixels is damaged, and the
// decode stops before it holds the pixels.
func TestAnImageAboveTheLimitIsDamaged(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer

	require.NoError(t, png.Encode(&encoded, plain(1, 1)))

	data := encoded.Bytes()
	header := data[8 : 8+8+13]
	binary.BigEndian.PutUint32(header[8:], 20000)
	binary.BigEndian.PutUint32(header[12:], 20000)
	binary.BigEndian.PutUint32(data[8+8+13:], crc32.ChecksumIEEE(header[4:]))

	_, err := file.VariantsOf(data, pngType)
	require.ErrorIs(t, err, file.ErrDamaged())

	_, err = file.VariantsOf(data[:len(data)/2], jpegType)
	require.ErrorIs(t, err, file.ErrDamaged())
}

// FILES-DD-008: The content decides the type. HEIC and HEIF match each other, and a
// PDF under the name of a photo is refused.
func TestSniffReadsTheTypeFromTheContent(t *testing.T) {
	t.Parallel()

	var photo bytes.Buffer

	require.NoError(t, heic.Encode(&photo, plain(64, 48)))

	detected, accepted := file.Sniff(
		photo.Bytes()[:min(photo.Len(), file.SniffLength)],
		[]string{heifType},
	)
	require.True(t, accepted)
	require.Equal(t, heicType, detected)

	detected, accepted = file.Sniff(
		[]byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n"),
		[]string{jpegType, heicType},
	)
	require.False(t, accepted)
	require.Equal(t, pdfType, detected)

	detected, accepted = file.Sniff(
		jpegWith(t, plain(8, 8), file.OrientationNormal),
		[]string{jpegType},
	)
	require.True(t, accepted)
	require.Equal(t, jpegType, detected)
}
