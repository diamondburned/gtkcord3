package singlebox

import (
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type Box struct {
	*gtk.Box
	Container
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
		Container: Container{
			parent: box,
		},
	}
}

type Column struct {
	*handy.Column
	Container
}

func ColumnNew() *Column {
	return WrapColumn(handy.ColumnNew())
}

func WrapColumn(col *handy.Column) *Column {
	return &Column{
		Column: col,
		Container: Container{
			parent: col,
		},
	}
}

type Container struct {
	Children gtk.IWidget

	parent interface {
		Remove(gtk.IWidget)
		Add(gtk.IWidget)
	}
}

func (b *Container) Clear() {
	b.Add(nil)
}

func (b *Container) Add(w gtk.IWidget) {
	if b.Children != nil {
		b.parent.Remove(b.Children)
	}

	b.Children = w

	if w == nil {
		return
	}

	b.parent.Add(w)
}
