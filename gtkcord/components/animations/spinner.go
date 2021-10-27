package animations

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

func NewSpinner(sz int) gtk.Widgetter {
	s := gtk.NewSpinner()
	s.SetSizeRequest(sz, sz)
	s.SetVAlign(gtk.AlignCenter)
	s.SetHAlign(gtk.AlignCenter)
	s.Start()
	s.ShowAll()

	return s
}

func NewSizedSpinner(sz int) gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.SetHExpand(true)
	box.SetVAlign(gtk.AlignCenter)
	box.SetVAlign(gtk.AlignCenter)

	s := gtk.NewSpinner()
	s.SetHExpand(true)
	s.SetVAlign(gtk.AlignCenter)
	s.SetHAlign(gtk.AlignCenter)
	s.SetSizeRequest(sz, sz)

	box.Add(s)
	box.ConnectMap(func() { s.Start() })
	box.ConnectUnmap(func() { s.Stop() })
	box.ShowAll()

	return box
}
