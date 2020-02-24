package gtkcord

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
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
	Main *gtk.Box

	Style    *gtk.StyleContext
	oldClass string

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

// not thread safe
func NewUserPopup(relative gtk.IWidget) *UserPopup {
	p, _ := gtk.PopoverNew(relative)
	style, _ := p.GetStyleContext()

	gtkutils.InjectCSSUnsafe(p, "user-info", `
		popover.user-info { padding: 0; }
	`)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	p.Add(main)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.SetSizeRequest(ChannelsWidth, -1)
	b.SetMarginTop(10)
	b.SetMarginBottom(10)
	main.Add(b)

	iStatus, _ := gtk.ImageNew()
	iStatus.SetSizeRequest(HeaderStatusSize, HeaderStatusSize)

	circle := icons.SolidCircle(HeaderStatusSize, OfflineColor)

	if err := icons.SetImage(circle, iStatus); err != nil {
		log.Panicln("Failed to set status image to solid circle 0x000000:", err)
	}

	iAvatar, _ := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_LARGE_TOOLBAR)
	iAvatar.SetSizeRequest(HeaderAvatarSize, HeaderAvatarSize)

	avaOverlay, _ := gtk.OverlayNew()
	avaOverlay.SetMarginStart(7)
	avaOverlay.SetMarginEnd(7)
	avaOverlay.Add(iStatus)
	avaOverlay.AddOverlay(iAvatar)
	b.Add(avaOverlay)

	l, _ := gtk.LabelNew("?")
	l.SetXAlign(0.0)
	l.SetMarginStart(7)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	b.Add(l)

	return &UserPopup{
		Popover:  p,
		Main:     main,
		Style:    style,
		Status:   iStatus,
		Avatar:   iAvatar,
		Username: l,
	}
}

func (b *UserPopup) formatUser(u discord.User) string {
	return fmt.Sprintf(
		"<span weight=\"bold\">%s</span><span size=\"smaller\">#%s</span>",
		escape(u.Username), u.Discriminator,
	)
}

func (b *UserPopup) setClass(class string) {
	if b.oldClass != "" {
		b.Style.RemoveClass(b.oldClass)
	}

	if class == "" {
		return
	}

	b.oldClass = class
	b.Style.AddClass(class)
}

func (b *UserPopup) Update(u discord.User) {
	b.Username.SetMarkup(b.formatUser(u))

	if u.Avatar != "" {
		go b.updateAvatar(u.AvatarURL())
	}
}

func (b *UserPopup) UpdateMember(m discord.Member) {
	var body = b.formatUser(m.User)
	if m.Nick != "" {
		body = fmt.Sprintf(
			`<span weight="bold">%s</span>`+"\n%s",
			escape(m.Nick), body,
		)
	}

	b.Username.SetMarkup(body)

	if m.User.Avatar != "" {
		go b.updateAvatar(m.User.AvatarURL())
	}
}

func (b *UserPopup) updateAvatar(url string) {
	err := cache.SetImageScaled(
		url+"?size=64", b.Avatar, HeaderAvatarSize, HeaderAvatarSize, cache.Round)
	if err != nil {
		logWrap(err, "Failed to get the pixbuf avatar icon")
		return
	}
}

func (b *UserPopup) UpdateActivity(a *discord.Activity) {
	if a == nil {
		if b.Activity != nil {
			must(b.Main.Remove, b.Activity)
			b.Activity = nil
			b.setClass("")
		}
		return
	}

	if b.Activity == nil {
		b.Activity = must(NewUserPopupActivity).(*UserPopupActivity)
		must(b.Main.Add, b.Activity)
	}

	b.Activity.Update(*a)

	if strings.HasPrefix(a.Assets.LargeImage, "spotify:") {
		b.setClass("spotify")
		b.UpdateStatus(discord.UnknownStatus)
	} else {
		b.setClass("")
	}
}

func (b *UserPopup) UpdateStatus(status discord.Status) {
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
	case discord.OfflineStatus:
		color = OfflineColor
	case discord.UnknownStatus:
		color = 0xFFFFFF
	}

	circle := icons.SolidCircle(HeaderStatusSize, color)
	if err := icons.SetImage(circle, b.Status); err != nil {
		log.Errorln("Failed to set status image:", err)
	}
}

type UserPopupRoles struct {
	*gtk.Box
	Header *gtk.Label

	// can be nil
	Main  *gtk.FlowBox
	Roles []UserPopupRole
}
type UserPopupRole struct {
	*gtk.FlowBoxChild
	Main *gtk.Label
}

// thread-safe
func NewUserPopupRoles(guild discord.Snowflake, ids []discord.Snowflake) (*UserPopupRoles, error) {
	b := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	l := must(gtk.LabelNew, "Roles").(*gtk.Label)
	must(margin4, l, SectionPadding, 0, SectionPadding, SectionPadding)
	must(l.SetHAlign, gtk.ALIGN_START)
	must(b.Add, l)

	popup := &UserPopupRoles{
		Box:    b,
		Header: l,
	}

	if len(ids) == 0 {
		must(l.SetLabel, "No Roles")
		must(l.SetMarginBottom, SectionPadding)
		return popup, nil
	}

	fb := must(gtk.FlowBoxNew).(*gtk.FlowBox)
	must(margin, fb, SectionPadding)
	must(fb.SetSelectionMode, gtk.SELECTION_NONE)
	must(fb.SetHAlign, gtk.ALIGN_FILL)
	must(fb.SetVAlign, gtk.ALIGN_START)
	must(b.Add, fb)

	popup.Main = fb

	var roles = make([]UserPopupRole, 0, len(ids))
	popup.Roles = roles

	for _, id := range ids {
		r, err := App.State.Role(guild, id)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get role")
		}

		var color = r.Color
		if color == 0 {
			color = 0x555555
		}

		var hex = fmt.Sprintf("#%06X", color)

		must(func() {
			l, _ := gtk.LabelNew("")
			l.SetTooltipText(r.Name)
			l.SetLabel(" " + r.Name + " ")
			l.SetSingleLineMode(true)
			l.SetEllipsize(pango.ELLIPSIZE_END)
			l.SetMaxWidthChars(20)

			gtkutils.InjectCSSUnsafe(l, "", `
				label {
					border: 1px solid `+hex+`;
					border-left-width: 5px;
				}
			`)

			c, _ := gtk.FlowBoxChildNew()
			c.Add(l)
			fb.Insert(c, -1)

			roles = append(roles, UserPopupRole{
				FlowBoxChild: c,
				Main:         l,
			})
		})
	}

	return popup, nil
}

func NewGuildMemberPopup(guildID, userID discord.Snowflake) *gtk.Popover {
	popup := NewUserPopup(nil)

	go func() {
		u, err := App.State.User(userID)
		if err != nil {
			log.Errorln("Failed to get user:", err)
			return
		}

		p, err := App.State.Presence(guildID, u.ID)
		if err != nil {
			requestMember(guildID, userID)
		} else {
			popup.UpdateStatus(p.Status)
			popup.UpdateActivity(p.Game)
		}

		m, err := App.State.Member(guildID, u.ID)
		if err != nil {
			popup.Update(*u)
			return
		}

		popup.UpdateMember(*m)

		r, err := NewUserPopupRoles(guildID, m.RoleIDs)
		if err != nil {
			log.Errorln("Failed to get roles:", err)
			return
		}

		must(popup.Main.Add, r)
		must(popup.Main.ShowAll)
	}()

	return popup.Popover
}

func requestMember(guild discord.Snowflake, user ...discord.Snowflake) {
	err := App.State.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
		GuildID:   []discord.Snowflake{guild},
		UserIDs:   user,
		Presences: true,
	})

	if err != nil {
		log.Errorln("Failed to request guild members:", err)
	}
}
