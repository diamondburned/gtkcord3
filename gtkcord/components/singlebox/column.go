package singlebox

import (
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type Column struct {
	*handy.Column
	Children gtk.IWidget
}

func ColumnNew() *Column {
	return WrapColumn(handy.ColumnNew())
}

func WrapColumn(col *handy.Column) *Column {
	return &Column{
		Column: col,
	}
}

func (c *Column) Clear() {
	c.Add(nil)
}

func (c *Column) Add(w gtk.IWidget) {
	if c.Children != nil {
		c.Column.Remove(c.Children)
	}

	c.Children = w

	if w == nil {
		return
	}

	c.Column.Add(w)
}
