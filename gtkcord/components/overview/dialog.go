package overview

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
)

func SpawnDialog(w gtk.Widgetter) {
	header := gtk.NewHeaderBar()
	header.Show()
	header.SetTitle("Details")
	header.SetShowCloseButton(true)

	d := gtk.NewDialog()
	d.SetTransientFor(&window.Window.Window)
	d.SetDefaultSize(600, 600)
	d.SetTitlebar(header)

	a := d.ContentArea()
	d.Remove(a)
	d.Add(w)

	d.Run()
	d.GrabFocus()
}
