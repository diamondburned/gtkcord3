package gtkcord

import (
	"fmt"
	"io/ioutil"

	"github.com/diamondburned/arikawa/state"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const AvatarSize = 38

type HeaderMenu struct {
	*gtk.MenuButton
	Avatar *gtk.Image
	Name   *gtk.Label

	// Avatar
	Pixbuf    *gdk.Pixbuf
	Animation *gdk.PixbufAnimation

	// About
}

func newHeaderMenu() (*HeaderMenu, error) {
	m, err := gtk.MenuButtonNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create menu button")
	}

	hm := &HeaderMenu{
		MenuButton: m,
	}

	// header box

	b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create hamburger header box")
	}

	i, err := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar placeholder")
	}
	i.SetSizeRequest(IconSize-IconPadding, -1)
	b.Add(i)
	hm.Avatar = i

	l, err := gtk.LabelNew("?")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create label")
	}
	l.SetXAlign(0.0)
	l.SetMarginStart(10)
	b.Add(i)
	hm.Name = l

	m.Add(b)

	return hm, nil
}

func (m *HeaderMenu) Refresh(s *state.State) error {
	u, err := s.Me()
	if err != nil {
		return errors.Wrap(err, "Failed to get myself")
	}

	m.Name.SetMarkup(escape(u.Username + "#" + u.Discriminator))

	if u.Avatar != "" {
		go m.UpdateAvatar(u.AvatarURL())
	}

	return nil
}

func (m *HeaderMenu) UpdateAvatar(url string) {
	var animated = url[:len(url)-4] == ".gif"

	r, err := HTTPClient.Get(url + "?size=64")
	if err != nil {
		logWrap(err, "Failed to GET URL "+url)
		return
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		logError(fmt.Errorf("Bad status code %d for %s", r.StatusCode, url))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logWrap(err, "Failed to download image")
		return
	}

	if !animated {
		p, err := NewPixbuf(b, PbSize(AvatarSize, AvatarSize))
		if err != nil {
			logWrap(err, "Failed to get the pixbuf guild icon")
			return
		}

		m.Pixbuf = p
	} else {
		p, err := NewAnimator(b, PbSize(AvatarSize, AvatarSize))
		if err != nil {
			logWrap(err, "Failed to get the pixbuf guild animation")
		}

		m.Animation = p
	}

	m.updateAvatar()
}

func (m *HeaderMenu) updateAvatar() {
	switch {
	case m.Pixbuf != nil:
		must(func(m *HeaderMenu) {
			m.Avatar.SetFromPixbuf(m.Pixbuf)
		}, m)
	case m.Animation != nil:
		must(func(m *HeaderMenu) {
			m.Avatar.SetFromAnimation(m.Animation)
		}, m)
	}
}
