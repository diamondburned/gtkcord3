package gtkcord

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

const (
	HeaderAvatarSize = 48
	HeaderStatusSize = HeaderAvatarSize + 6 // used for cover too

	OfflineColor = 0x747F8D
	BusyColor    = 0xF04747
	IdleColor    = 0xFAA61A
	OnlineColor  = 0x43B581
)

type UserPopup struct {
	*gtk.Popover
}

type UserPopupBody struct {
	*gtk.Box

	Status   *gtk.Image
	Avatar   *gtk.Image
	Username *gtk.Label

	Activity *UserPopupActivity
}

func userMentionPressed(ev *gdk.EventButton, user discord.GuildUser) {
	log.Println("User mention pressed:", user.Username)
}

func channelMentionPressed(ev *gdk.EventButton, ch discord.Channel) {
	log.Println("Channel mention pressed:", ch.Name)
}

func NewUserPopupBody() (*UserPopupBody, error) {
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.SetSizeRequest(ChannelsWidth, -1)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetMarginTop(10)
	main.SetMarginBottom(10)
	main.Add(b)

	iStatus, _ := gtk.ImageNew()
	iStatus.SetSizeRequest(HeaderStatusSize, HeaderStatusSize)

	circle := icons.SolidCircle(HeaderStatusSize, OfflineColor)

	if err := icons.SetImage(circle, iStatus); err != nil {
		return nil, errors.Wrap(err, "Failed to set status image to solid circle 0x000000")
	}

	iAvatar, _ := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_LARGE_TOOLBAR)
	iAvatar.SetSizeRequest(HeaderAvatarSize, HeaderAvatarSize)

	avaOverlay, _ := gtk.OverlayNew()
	avaOverlay.SetMarginStart(7)
	avaOverlay.Add(iStatus)
	avaOverlay.AddOverlay(iAvatar)
	b.Add(avaOverlay)

	l, _ := gtk.LabelNew("?")
	l.SetXAlign(0.0)
	l.SetMarginStart(7)
	l.SetMarginEnd(10)
	l.SetSingleLineMode(true)
	l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	b.Add(l)

	return &UserPopupBody{
		Box:      main,
		Status:   iStatus,
		Avatar:   iAvatar,
		Username: l,
	}, nil
}

func (b *UserPopupBody) Update(u discord.User) {
	b.Username.SetMarkup(fmt.Sprintf(
		"<span weight=\"bold\">%s</span><span size=\"smaller\">#%s</span>",
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

func (b *UserPopupBody) UpdateActivity(a *discord.Activity) {
	if a == nil {
		if b.Activity != nil {
			must(b.Remove, b.Activity)
			b.Activity = nil
		}
		return
	}

	if b.Activity == nil {
		b.Activity = must(NewUserPopupActivity).(*UserPopupActivity)
		must(b.Add, b.Activity)
	}

	b.Activity.Update(*a)
	must(b.ShowAll)
}

func (b *UserPopupBody) UpdateStatus(status discord.Status) {
	var color uint32

	switch status {
	case discord.OnlineStatus:
		color = OnlineColor
	case discord.DoNotDisturbStatus:
		color = BusyColor
	case discord.IdleStatus:
		color = IdleColor
	case discord.InvisibleStatus:
		color = OfflineColor
	case discord.OfflineStatus, discord.UnknownStatus:
		color = OfflineColor
	}

	circle := icons.SolidCircle(HeaderStatusSize, color)
	if err := icons.SetImage(circle, b.Status); err != nil {
		log.Errorln("Failed to set status image:", err)
	}
}

type UserPopupActivity struct {
	*gtk.Box
	Header *gtk.Label

	Details *gtk.Box
	Image   *gtk.Image
	Info    *gtk.Label
}

func NewUserPopupActivity() *UserPopupActivity {
	details, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	details.SetMarginTop(7)

	header, _ := gtk.LabelNew("")
	header.SetMarginTop(5)
	header.SetMarginStart(7)
	header.SetMarginEnd(7)
	header.SetHAlign(gtk.ALIGN_START)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetMarginTop(10)
	main.Add(header)
	main.Add(details)

	return &UserPopupActivity{
		Box:     main,
		Header:  header,
		Details: details,
	}
}

func assetURL(id discord.Snowflake, asset string) string {
	if strings.HasPrefix(asset, "spotify:") {
		return "https://i.scdn.co/image/" + strings.TrimPrefix(asset, "spotify:")
	}
	return "https://cdn.discordapp.com/app-assets/" + id.String() + "/" + asset + ".png"
}

func assetHeader(name string) string {
	return `<span size="smaller" weight="bold">` + escape(name) + "</span>"
}

func (a *UserPopupActivity) Update(ac discord.Activity) {
	var (
		imgURL string
		header string
	)

	switch ac.Type {
	case discord.GameActivity:
		imgURL = assetURL(ac.ApplicationID, ac.Assets.LargeImage)
		header = assetHeader("Playing " + ac.Name)

	case discord.ListeningActivity:
		imgURL = assetURL(ac.ApplicationID, ac.Assets.LargeImage)
		header = assetHeader("Listening to " + ac.Name)

	case discord.StreamingActivity:
		imgURL = assetURL(ac.ApplicationID, ac.Assets.LargeImage)
		header = assetHeader("Streaming " + ac.Details)

	case discord.CustomActivity:
		header = escape(ac.Name)
	}

	must(a.updateImage, imgURL, ac.Assets.LargeText)

	if a.Info == nil {
		l := must(gtk.LabelNew, "?").(*gtk.Label)
		must(func() {
			l.SetXAlign(0.0)
			l.SetMarginStart(7)
			l.SetMarginEnd(7)
			l.SetLineWrapMode(pango.WRAP_WORD_CHAR)

			a.Details.Add(l)
		})

		a.Info = l
	}

	must(a.Header.SetMarkup, `<span weight="bold">`+header+`</span>`)
	must(a.Info.SetMarkup, fmt.Sprintf(
		"<span weight=\"bold\">%s</span>\n<span size=\"smaller\">%s</span>",
		escape(ac.Details), escape(ac.State),
	))
}

// not thread safe
func (a *UserPopupActivity) updateImage(url, text string) {
	if url == "" {
		if a.Image != nil {
			a.Remove(a.Image)
			a.Image.Destroy()
			a.Image = nil
		}
		return
	}

	if a.Image == nil {
		a.Image, _ = gtk.ImageNew()
		a.Image.SetSizeRequest(HeaderStatusSize, HeaderStatusSize)
		a.Image.SetMarginStart(7)
		a.Details.PackStart(a.Image, false, false, 0)
	}

	a.Image.SetTooltipText(text)
	go asyncFetch(url, a.Image, HeaderStatusSize, HeaderStatusSize)
}
