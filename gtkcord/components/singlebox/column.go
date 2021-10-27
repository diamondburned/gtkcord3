package singlebox

/*
type Column struct {
	*handy.Column
	Children gtk.Widgetter
}

func ColumnNew() *Column {
	return WrapColumn(handy.NewColumn())
}

func WrapColumn(col *handy.Column) *Column {
	return &Column{
		Column: col,
	}
}

func (c *Column) Clear() {
	c.Add(nil)
}

func (c *Column) Add(w gtk.Widgetter) {
	if c.Children != nil {
		c.Column.Remove(c.Children)
	}

	c.Children = w

	if w == nil {
		return
	}

	c.Column.Add(w)
}
*/
