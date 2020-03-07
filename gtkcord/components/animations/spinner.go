package animations

import "github.com/gotk3/gotk3/gtk"

func NewSpinner(sz int) (gtk.IWidget, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	b.SetHExpand(true)
	b.SetVExpand(true)

	s, err := gtk.SpinnerNew()
	if err != nil {
		return nil, err
	}
	s.SetSizeRequest(sz, sz)
	s.SetVAlign(gtk.ALIGN_CENTER)
	s.SetHAlign(gtk.ALIGN_CENTER)

	b.Add(s)

	return b, nil
}
