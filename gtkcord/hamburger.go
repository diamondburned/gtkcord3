package gtkcord

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type HeaderMenu struct {
	gtkutils.ExtendedWidget
	User *UserPopup

	// About
}

func newHeaderMenu() (*HeaderMenu, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make hamburger box")
	}
	b.SetSizeRequest(IconSize+IconPadding*2, -1)

	mb, err := gtk.MenuButtonNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create menu button")
	}
	mb.SetHAlign(gtk.ALIGN_CENTER)
	b.Add(mb)

	i, err := gtk.ImageNewFromIconName("open-menu", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar placeholder")
	}
	mb.Add(i)

	// Header box
	u := NewUserPopup(mb)
	u.Main.ShowAll()

	mb.SetPopover(u.Popover)
	mb.SetUsePopover(true)

	hm := &HeaderMenu{
		ExtendedWidget: b,
		User:           u,
	}
	hm.ShowAll()

	return hm, nil
}
