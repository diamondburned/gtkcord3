package animations

import "github.com/gotk3/gotk3/gtk"

func NewSpinner(sz int) (gtk.IWidget, error) {
	s, err := gtk.SpinnerNew()
	if err != nil {
		return nil, err
	}
	s.SetSizeRequest(sz, sz)
	s.SetVAlign(gtk.ALIGN_CENTER)
	s.SetHAlign(gtk.ALIGN_CENTER)

	return s, nil
}
