package header

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
)

type MainHamburger struct {
	gtk.Widgetter
	Button *gtk.MenuButton
}

func newMainHamburger() *MainHamburger {
	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.Show()
	b.SetSizeRequest(guild.TotalWidth, -1)

	mb := gtk.NewMenuButton()
	mb.SetSensitive(true)
	mb.SetHAlign(gtk.AlignCenter)
	mb.Show()
	b.Add(mb)

	i := gtk.NewImageFromIconName("open-menu", int(gtk.IconSizeLargeToolbar))
	i.Show()
	mb.Add(i)

	return &MainHamburger{Widgetter: b, Button: mb}
}
