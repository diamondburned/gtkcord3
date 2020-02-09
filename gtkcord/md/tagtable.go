package md

import "github.com/gotk3/gotk3/gtk"

var _tt *gtk.TextTagTable

// TODO
func NewTagTable() (*gtk.TextTagTable, error) {
	if _tt == nil {
		tt, err := gtk.TextTagTableNew()
		if err != nil {
			return nil, err
		}

		_tt = tt
	}

	return _tt, nil
}
