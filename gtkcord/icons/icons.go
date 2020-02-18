package icons

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"

	"github.com/gotk3/gotk3/gdk"
	"github.com/pkg/errors"
)

func FromPNG(b64 string) image.Image {
	b, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		panic("Failed to decode image: " + err.Error())
	}

	i, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		panic("Failed to decode image: " + err.Error())
	}

	return i
}

var pngEncoder = png.Encoder{
	CompressionLevel: png.BestSpeed,
}

func Pixbuf(img image.Image) (*gdk.Pixbuf, error) {
	var buf bytes.Buffer

	if err := pngEncoder.Encode(&buf, img); err != nil {
		return nil, errors.Wrap(err, "Failed to encode PNG")
	}

	l, err := gdk.PixbufLoaderNewWithType("png")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create an icon pixbuf loader")
	}

	p, err := l.WriteAndReturnPixbuf(buf.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "Failed to set icon to pixbuf")
	}

	return p, nil
}

func PixbufIcon(img image.Image, size int) (*gdk.Pixbuf, error) {
	var buf bytes.Buffer

	if err := pngEncoder.Encode(&buf, img); err != nil {
		return nil, errors.Wrap(err, "Failed to encode PNG")
	}

	l, err := gdk.PixbufLoaderNewWithType("png")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create an icon pixbuf loader")
	}

	l.SetSize(size, size)

	p, err := l.WriteAndReturnPixbuf(buf.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "Failed to set icon to pixbuf")
	}

	return p, nil
}

func PixbufSolid(w, h int, r, g, b, a uint8) (*gdk.Pixbuf, error) {
	i := image.NewNRGBA(image.Rect(0, 0, w, h))
	for j := 0; j < len(i.Pix); j += 4 {
		i.Pix[j+0] = r
		i.Pix[j+1] = g
		i.Pix[j+2] = b
		i.Pix[j+3] = a
	}

	return Pixbuf(i)
}
