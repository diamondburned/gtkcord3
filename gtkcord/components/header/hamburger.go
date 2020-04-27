package header

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type MainHamburger struct {
	gtkutils.ExtendedWidget
	Button *gtk.MenuButton
}

func newMainHamburger() (*MainHamburger, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make hamburger box")
	}
	b.Show()
	b.SetSizeRequest(guild.TotalWidth, -1)

	mb, err := gtk.MenuButtonNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create menu button")
	}
	mb.SetSensitive(true)
	mb.SetHAlign(gtk.ALIGN_CENTER)
	mb.Show()
	b.Add(mb)

	i, err := gtk.ImageNewFromIconName("open-menu", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar placeholder")
	}
	i.Show()
	mb.Add(i)

	return &MainHamburger{ExtendedWidget: b, Button: mb}, nil
}
