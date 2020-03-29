package singlebox

import (
	"github.com/gotk3/gotk3/gtk"
)

type Box struct {
	*gtk.Box
	Children gtk.IWidget
}

func BoxNew(o gtk.Orientation, spacing int) (*Box, error) {
	b, err := gtk.BoxNew(o, spacing)
	if err != nil {
		return nil, err
	}

	return WrapBox(b), nil
}

func WrapBox(box *gtk.Box) *Box {
	return &Box{
		Box: box,
	}
}

func (b *Box) Clear() {
	b.Add(nil)
}

func (b *Box) Add(w gtk.IWidget) {
	if b.Children != nil {
		b.Box.Remove(b.Children)
	}

	b.Children = w

	if w == nil {
		return
	}

	b.Box.Add(w)
}
