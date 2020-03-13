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
	Popover *popup.Popover

	// About
}

const HeaderWidth = 240

func NewHeaderMenu(s *ningen.State) (*Hamburger, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make hamburger box")
	}
	b.SetSizeRequest(guild.IconSize+guild.IconPadding*2, -1)

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
	p := popup.NewDynamicPopover(mb, func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		body := popup.NewStatefulPopupBody(s, s.Ready.User.ID, 0)
		body.ParentStyle, _ = p.GetStyleContext()
		hamburgerAddExtras(s, body.Box)
		return body
	})

	mb.SetPopover(p.Popover)
	mb.SetUsePopover(true)

	hm := &Hamburger{
		ExtendedWidget: b,
		Popover:        p,
	}
	hm.ShowAll()

	return hm, nil
}

func hamburgerAddExtras(s *ningen.State, box *gtk.Box) {

}
