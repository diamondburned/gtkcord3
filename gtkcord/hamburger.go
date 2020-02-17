package gtkcord

import (
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const HeaderAvatarSize = 38

type HeaderMenu struct {
	ExtendedWidget
	Menu   *gtk.MenuButton
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

	m, err := gtk.MenuButtonNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create menu button")
	}
	m.SetHAlign(gtk.ALIGN_CENTER)
	b.Add(m)

	i, err := gtk.ImageNewFromIconName("open-menu", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar placeholder")
	}

	m.Add(i)

	hm := &HeaderMenu{
		ExtendedWidget: b,
		Menu:           m,
	}

	{ // header box

		b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create hamburger header box")
		}

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
		b.Add(l)
		hm.Name = l

		c, err := gtk.PopoverMenuNew()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to make popover menu")
		}

		c.Add(b)
		m.SetPopover(&c.Popover)
	}

	hm.ShowAll()

	return hm, nil
}

func (m *HeaderMenu) Refresh() error {
	me := App.Me

	m.Name.SetMarkup(escape(me.Username + "#" + me.Discriminator))

	if me.Avatar != "" {
		go m.UpdateAvatar(me.AvatarURL())
	}

	return nil
}

func (m *HeaderMenu) UpdateAvatar(url string) {
	var animated = url[:len(url)-4] == ".gif"
	var err error

	if !animated {
		err = cache.SetImage(
			url+"?size=64", m.Avatar, cache.Resize(HeaderAvatarSize, HeaderAvatarSize))
	} else {
		err = cache.SetAnimation(
			url+"?size=64", m.Avatar, cache.Resize(HeaderAvatarSize, HeaderAvatarSize))
	}

	if err != nil {
		logWrap(err, "Failed to get the pixbuf guild icon")
		return
	}
}
