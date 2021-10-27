package singlebox

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

type Box struct {
	*gtk.Box
	Children gtk.Widgetter
}

func NewBox(o gtk.Orientation, spacing int) *Box {
	return WrapBox(gtk.NewBox(o, spacing))
}

func WrapBox(box *gtk.Box) *Box {
	return &Box{
		Box: box,
	}
}

func (b *Box) Clear() {
	b.SetChild(nil)
}

func (b *Box) Add(w gtk.Widgetter) {
	b.SetChild(w)
}

func (b *Box) SetChild(w gtk.Widgetter) {
	if b.Children != nil {
		b.Box.Remove(b.Children)
	}

	b.Children = w

	if w == nil {
		return
	}

	b.Box.Add(w)
}
