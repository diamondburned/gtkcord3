package gtkcord

import (
	"fmt"

	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const HeaderAvatarSize = 38

type HeaderMenu struct {
	gtkutils.ExtendedWidget

	Menu   *gtk.Popover
	Avatar *gtk.Image
	Name   *gtk.Label

	// Avatar
	Pixbuf    *gdk.Pixbuf
	Animation *gdk.PixbufAnimation

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

	m, err := gtk.PopoverNew(mb)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create menu")
	}
	mb.SetPopover(m)
	mb.SetUsePopover(true)

	hm := &HeaderMenu{
		ExtendedWidget: b,
		Menu:           m,
	}

	{ // header box

		b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create hamburger header box")
		}
		b.SetMarginTop(7)
		b.SetMarginBottom(7)
		b.SetSizeRequest(ChannelsWidth, -1)

		i, err := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_LARGE_TOOLBAR)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create avatar placeholder")
		}
		i.SetSizeRequest(IconSize, -1)
		b.Add(i)
		hm.Avatar = i

		l, err := gtk.LabelNew("?")
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create label")
		}
		l.SetXAlign(0.0)
		l.SetMarginStart(10)
		l.SetMarginEnd(10)
		b.Add(l)
		hm.Name = l

		b.ShowAll()
		m.Add(b)
	}

	hm.ShowAll()

	return hm, nil
}

func (m *HeaderMenu) Refresh() {
	me := App.Me

	must(m.Name.SetMarkup, fmt.Sprintf(
		"<span weight=\"bold\">%s</span>\n<span size=\"smaller\">#%s</span>",
		escape(me.Username), me.Discriminator,
	))

	if me.Avatar != "" {
		go m.UpdateAvatar(me.AvatarURL())
	}

	return
}

func (m *HeaderMenu) UpdateAvatar(url string) {
	err := cache.SetImage(
		url+"?size=64", m.Avatar,
		cache.Resize(HeaderAvatarSize, HeaderAvatarSize), cache.Round)
	if err != nil {
		logWrap(err, "Failed to get the pixbuf guild icon")
		return
	}
}
