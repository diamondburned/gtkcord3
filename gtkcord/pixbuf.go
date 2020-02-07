package gtkcord

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/pkg/errors"
)

func loadPixbuf(
	b []byte, cfg func(pl *gdk.PixbufLoader)) (*gdk.PixbufLoader, error) {

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new pixbuf loader")
	}

	cfg(l)

	if _, err := l.Write(b); err != nil {
		return nil, errors.Wrap(err, "Failed to set image to pixbuf")
	}

	return l, nil
}

func NewPixbuf(
	b []byte, cfg func(pl *gdk.PixbufLoader)) (*gdk.Pixbuf, error) {

	l, err := loadPixbuf(b, cfg)
	if err != nil {
		return nil, err
	}

	p, err := l.GetPixbuf()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the pixbuf icon")
	}

	return p, nil
}

func NewAnimator(
	b []byte, cfg func(pl *gdk.PixbufLoader)) (*gdk.PixbufAnimation, error) {

	l, err := loadPixbuf(b, cfg)
	if err != nil {
		return nil, err
	}

	p, err := l.GetAnimation()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the pixbuf icon")
	}

	return p, nil
}

func PbSize(width, height int) func(pl *gdk.PixbufLoader) {
	return func(pl *gdk.PixbufLoader) {
		pl.SetSize(width, height)
	}
}

func PbNoop(*gdk.PixbufLoader) {}
