package header

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Hamburger struct {
	gtkutils.ExtendedWidget
	Button *gtk.MenuButton

	State *ningen.State
}

func NewHeaderMenu() (*Hamburger, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make hamburger box")
	}
	b.Show()
	b.SetSizeRequest(guild.IconSize+guild.IconPadding*2, -1)

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

	hm := &Hamburger{ExtendedWidget: b, Button: mb}

	p := popup.NewDynamicPopover(mb, func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		if hm.State == nil {
			return nil
		}

		return popup.NewHamburger(hm.State, p)
	})

	mb.SetPopover(p.Popover)
	mb.SetUsePopover(true)

	return hm, nil
}

func (h *Hamburger) UseState(s *ningen.State) {
	h.State = s
}
