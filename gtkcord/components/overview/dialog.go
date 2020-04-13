package overview

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func SpawnDialog(c *Container) {
	d := handy.DialogNew(window.Window)
	d.SetDefaultSize(600, 600)

	// Hack for close button
	d.Connect("response", func(_ *glib.Object, resp gtk.ResponseType) {
		if resp == gtk.RESPONSE_DELETE_EVENT {
			d.Destroy()
		}
	})

	header, _ := gtk.HeaderBarNew()
	header.Show()
	header.SetTitle("Details")
	header.SetShowCloseButton(true)
	d.SetTitlebar(header)

	a, _ := d.GetContentArea()
	d.Remove(a)
	d.Add(c)

	d.Run()
	d.GrabFocus()
}
