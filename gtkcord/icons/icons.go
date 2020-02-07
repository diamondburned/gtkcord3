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
		return nil, errors.Wrap(err, "Failed to create an SVG pixbuf loader")
	}

	if _, err := l.Write(buf.Bytes()); err != nil {
		return nil, errors.Wrap(err, "Failed to set SVG to pixbuf")
	}

	p, err := l.GetPixbuf()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the pixbuf SVG")
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
		return nil, errors.Wrap(err, "Failed to create an SVG pixbuf loader")
	}

	l.SetSize(size, size)

	if _, err := l.Write(buf.Bytes()); err != nil {
		return nil, errors.Wrap(err, "Failed to set SVG to pixbuf")
	}

	p, err := l.GetPixbuf()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the pixbuf SVG")
	}

	return p, nil
}
