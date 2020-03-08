package animations

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
)

func NewSpinner(sz int) (gtk.IWidget, error) {
	s, err := gtk.SpinnerNew()
	if err != nil {
		return nil, err
	}
	s.SetSizeRequest(sz, sz)
	s.SetVAlign(gtk.ALIGN_CENTER)
	s.SetHAlign(gtk.ALIGN_CENTER)
	s.Start()

	s.ShowAll()

	return s, nil
}

func NewSizedSpinner(sz int) (gtkutils.WidgetSizeRequester, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	b.SetHExpand(true)
	b.SetVAlign(gtk.ALIGN_CENTER)
	b.SetVAlign(gtk.ALIGN_CENTER)

	s, err := gtk.SpinnerNew()
	if err != nil {
		return nil, err
	}
	s.SetHExpand(true)
	s.SetVAlign(gtk.ALIGN_CENTER)
	s.SetHAlign(gtk.ALIGN_CENTER)
	s.SetSizeRequest(sz, sz)
	s.Start()

	b.Add(s)
	b.ShowAll()

	return b, nil
}
