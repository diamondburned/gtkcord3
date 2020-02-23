package icons

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"io"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
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

func SetImage(img image.Image, gtkimg *gtk.Image) error {
	var buf bytes.Buffer

	if err := pngEncoder.Encode(&buf, img); err != nil {
		return errors.Wrap(err, "Failed to encode PNG")
	}

	l, err := gdk.PixbufLoaderNewWithType("png")
	if err != nil {
		return errors.Wrap(err, "Failed to create an icon pixbuf loader")
	}

	gtkutils.Connect(l, "area-updated", func() {
		p, err := l.GetPixbuf()
		if err != nil || p == nil {
			log.Errorln("Failed to get animation during area-prepared:", err)
			return
		}
		semaphore.IdleMust(gtkimg.SetFromPixbuf, p)
	})

	if _, err := io.Copy(l, &buf); err != nil {
		return errors.Wrap(err, "Failed to stream to pixbuf_loader")
	}

	if err := l.Close(); err != nil {
		return errors.Wrap(err, "Failed to close pixbuf_loader")
	}

	return nil
}
