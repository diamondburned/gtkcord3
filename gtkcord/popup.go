package gtkcord

import (
	"fmt"
	"log"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const HeaderAvatarSize = 48

type UserPopup struct {
	*gtk.Popover
}

type UserPopupBody struct {
	*gtk.Box

	Avatar   *gtk.Image
	Username *gtk.Label
}

func userMentionPressed(ev *gdk.EventButton, user discord.GuildUser) {
	log.Println("User mention pressed:", user.Username)
}

func channelMentionPressed(ev *gdk.EventButton, ch discord.Channel) {
	log.Println("Channel mention pressed:", ch.Name)
}

func NewUserPopupBody() (*UserPopupBody, error) {
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
	i.SetMarginStart(7)
	b.Add(i)

	l, err := gtk.LabelNew("?")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create label")
	}
	l.SetXAlign(0.0)
	l.SetMarginStart(7)
	l.SetMarginEnd(10)
	b.Add(l)

	return &UserPopupBody{
		Box:      b,
		Avatar:   i,
		Username: l,
	}, nil
}

func (b *UserPopupBody) Update(u discord.User) {
	b.Username.SetMarkup(fmt.Sprintf(
		"<span weight=\"bold\">%s</span>\n<span size=\"smaller\">#%s</span>",
		escape(u.Username), u.Discriminator,
	))

	if u.Avatar != "" {
		go b.updateAvatar(u.AvatarURL())
	}
}

func (b *UserPopupBody) UpdateMember(m discord.Member) {
	var body = fmt.Sprintf(
		"<span weight=\"bold\">%s</span>\n<span size=\"smaller\">#%s</span>",
		escape(m.User.Username), m.User.Discriminator,
	)
	if m.Nick != "" {
		body = fmt.Sprintf(
			`<span weight="bold" size="larger">%s</span>`+"\n%s",
			escape(m.Nick), body,
		)
	}

	b.Username.SetMarkup(body)

	if m.User.Avatar != "" {
		go b.updateAvatar(m.User.AvatarURL())
	}
}

func (b *UserPopupBody) updateAvatar(url string) {
	err := cache.SetImageScaled(
		url+"?size=64", b.Avatar, HeaderAvatarSize, HeaderAvatarSize, cache.Round)
	if err != nil {
		logWrap(err, "Failed to get the pixbuf avatar icon")
		return
	}
}
