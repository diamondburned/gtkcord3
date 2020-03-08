package popup

import (
	"fmt"
	"html"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

const (
	PopupAvatarSize = 48
	PopupWidth      = 240

	OfflineColor = 0x747F8D
	BusyColor    = 0xF04747
	IdleColor    = 0xFAA61A
	OnlineColor  = 0x43B581
)

func init() {
	md.ChannelPressed = ChannelMentionPressed
	md.UserPressed = UserMentionPressed
}

func UserMentionPressed(ev *gdk.EventButton, user discord.GuildUser) {
	log.Println("User mention pressed:", user.Username)
}

func ChannelMentionPressed(ev *gdk.EventButton, ch discord.Channel) {
	log.Println("Channel mention pressed:", ch.Name)
}

type Popover struct {
	*gtk.Popover
	Style    *gtk.StyleContext
	oldClass string

	Children gtkutils.WidgetDestroyer
}

func NewPopover(relative gtk.IWidget) *Popover {
	p, _ := gtk.PopoverNew(relative)
	style, _ := p.GetStyleContext()

	gtkutils.InjectCSSUnsafe(p, "user-info", `
		popover.user-info { padding: 0; }
	`)

	return &Popover{
		Popover: p,
		Style:   style,
	}
}

func (p *Popover) SetChildren(children gtkutils.WidgetDestroyer) {
	if p.Children != nil {
		p.Popover.Remove(p.Children)
	}
	p.Children = children
	p.Popover.Add(children)
}

type PopoverCreator = func(p *gtk.Popover) gtkutils.WidgetDestroyer

func NewDynamicPopover(relative gtkutils.WidgetConnector, create PopoverCreator) *Popover {
	p := NewPopover(relative)
	p.Connect("closed", func() {
		p.Children.Destroy()
		p.Children = nil
	})
	relative.Connect("clicked", func() {
		if w := create(p.Popover); w != nil {
			p.SetChildren(w)
			p.ShowAll()
		}
	})

	return p
}

type UserPopup struct {
	*Popover
	*UserPopupBody
}

type UserPopupBody struct {
	*gtk.Box

	Avatar      *gtk.Image
	AvatarStyle *gtk.StyleContext // .avatar
	// check window/css.go header for status_* colors
	lastAvatarClass string

	Username *gtk.Label

	Activity *UserPopupActivity
}

// not thread safe
func NewUserPopup(relative gtk.IWidget) *UserPopup {
	return NewUserPopupCustom(relative, NewUserPopupBody())
}

// not thread safe
func NewUserPopupCustom(relative gtk.IWidget, body *UserPopupBody) *UserPopup {
	p := NewPopover(relative)
	p.SetChildren(body)

	return &UserPopup{
		Popover:       p,
		UserPopupBody: body,
	}
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

func (b *UserPopup) UpdateActivity(a *discord.Activity) {
	b.UserPopupBody.UpdateActivity(a)

	if a == nil {
		b.setClass("")
		return
	}

	if strings.HasPrefix(a.Assets.LargeImage, "spotify:") {
		b.setClass("spotify")
	} else {
		b.setClass("")
	}
}

func NewUserPopupBody() *UserPopupBody {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.SetSizeRequest(PopupWidth, -1)
	b.SetMarginTop(10)
	b.SetMarginBottom(10)
	main.Add(b)

	iAvatar, _ := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_LARGE_TOOLBAR)
	iAvatar.SetSizeRequest(PopupAvatarSize, PopupAvatarSize)
	iAvatar.SetMarginStart(7)
	iAvatar.SetMarginEnd(7)
	b.Add(iAvatar)

	sAvatar, _ := iAvatar.GetStyleContext()
	sAvatar.AddClass("avatar")
	sAvatar.AddClass("status")

	l, _ := gtk.LabelNew("?")
	l.SetXAlign(0.0)
	l.SetMarginStart(7)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	b.Add(l)

	return &UserPopupBody{
		Box:         main,
		Avatar:      iAvatar,
		AvatarStyle: sAvatar,
		Username:    l,
	}
}

func formatUser(u discord.User) string {
	return fmt.Sprintf(
		"<span weight=\"bold\">%s</span><span size=\"smaller\">#%s</span>",
		html.EscapeString(u.Username), u.Discriminator,
	)
}

func (b *UserPopupBody) setAvatarClass(class string) {
	if b.lastAvatarClass != "" {
		b.AvatarStyle.RemoveClass(b.lastAvatarClass)
		b.lastAvatarClass = ""
	}

	if class == "" {
		return
	}

	b.lastAvatarClass = class
	b.AvatarStyle.AddClass(class)
}

func (b *UserPopupBody) Update(u discord.User) {
	b.Username.SetMarkup(formatUser(u))

	if u.Avatar != "" {
		go b.updateAvatar(u.AvatarURL())
	}
}

func (b *UserPopupBody) UpdateMember(m discord.Member) {
	b.UpdateMemberPart(m.Nick, m.User)
}

func (b *UserPopupBody) UpdateMemberPart(nick string, u discord.User) {
	var body = formatUser(u)
	if nick != "" {
		body = fmt.Sprintf(
			`<span weight="bold">%s</span>`+"\n"+`<span size="smaller">%s</span>`,
			html.EscapeString(nick), body,
		)
	}

	b.Username.SetMarkup(body)

	if u.Avatar != "" {
		go b.updateAvatar(u.AvatarURL())
	}
}

func (b *UserPopupBody) updateAvatar(url string) {
	err := cache.SetImageScaled(
		url+"?size=64", b.Avatar, PopupAvatarSize, PopupAvatarSize, cache.Round)
	if err != nil {
		log.Errorln("Failed to get the pixbuf avatar icon:", err)
		return
	}
}

func (b *UserPopupBody) UpdateActivity(a *discord.Activity) {
	if a == nil {
		if b.Activity != nil {
			semaphore.IdleMust(b.Box.Remove, b.Activity)
			b.Activity = nil
			// b.setClass("")
		}
		return
	}

	if b.Activity == nil {
		b.Activity = semaphore.IdleMust(NewUserPopupActivity).(*UserPopupActivity)
		semaphore.IdleMust(b.Box.Add, b.Activity)
	}

	b.Activity.Update(*a)

	if strings.HasPrefix(a.Assets.LargeImage, "spotify:") {
		b.UpdateStatus(discord.UnknownStatus)
	}

	semaphore.IdleMust(b.Box.ShowAll)
}

func (b *UserPopupBody) UpdateStatus(status discord.Status) {
	semaphore.IdleMust(func() {
		switch status {
		case discord.OnlineStatus:
			b.setAvatarClass("online")
		case discord.DoNotDisturbStatus:
			b.setAvatarClass("busy")
		case discord.IdleStatus:
			b.setAvatarClass("idle")
		case discord.InvisibleStatus, discord.OfflineStatus:
			b.setAvatarClass("offline")
		case discord.UnknownStatus:
			b.setAvatarClass("unknown")
		}
	})
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
func NewUserPopupRoles(s *ningen.State,
	guild discord.Snowflake, ids []discord.Snowflake) (*UserPopupRoles, error) {

	// TODO: optimize this

	b := semaphore.IdleMust(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	l := semaphore.IdleMust(gtk.LabelNew, "Roles").(*gtk.Label)
	semaphore.IdleMust(gtkutils.Margin4, l, SectionPadding, 0, SectionPadding, SectionPadding)
	semaphore.IdleMust(l.SetHAlign, gtk.ALIGN_START)
	semaphore.IdleMust(b.Add, l)

	popup := &UserPopupRoles{
		Box:    b,
		Header: l,
	}

	if len(ids) == 0 {
		semaphore.IdleMust(l.SetLabel, "No Roles")
		semaphore.IdleMust(l.SetMarginBottom, SectionPadding)
		return popup, nil
	}

	fb := semaphore.IdleMust(gtk.FlowBoxNew).(*gtk.FlowBox)
	semaphore.IdleMust(gtkutils.Margin, fb, SectionPadding)
	semaphore.IdleMust(fb.SetSelectionMode, gtk.SELECTION_NONE)
	semaphore.IdleMust(fb.SetHAlign, gtk.ALIGN_FILL)
	semaphore.IdleMust(fb.SetVAlign, gtk.ALIGN_START)
	semaphore.IdleMust(b.Add, fb)

	popup.Main = fb

	var roles = make([]UserPopupRole, 0, len(ids))
	popup.Roles = roles

	for _, id := range ids {
		r, err := s.Role(guild, id)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get role")
		}

		var color = r.Color
		if color == 0 {
			color = 0x555555
		}

		var hex = fmt.Sprintf("#%06X", color)

		semaphore.IdleMust(func() {
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

func SpawnUserPopup(s *ningen.State, guildID, userID discord.Snowflake) *gtk.Popover {
	popup := NewUserPopup(nil)

	go func() {
		u, err := s.User(userID)
		if err != nil {
			log.Errorln("Failed to get user:", err)
			return
		}

		p, err := s.Presence(guildID, u.ID)
		if err == nil {
			popup.UpdateStatus(p.Status)
			popup.UpdateActivity(p.Game)
		}

		if !guildID.Valid() {
			popup.Update(*u)
			return
		}

		// fetch above presence if error not nil
		if err != nil {
			s.RequestMember(guildID, userID)
		}

		m, err := s.Member(guildID, u.ID)
		if err != nil {
			popup.Update(*u)
			return
		}

		popup.UpdateMember(*m)

		r, err := NewUserPopupRoles(s, guildID, m.RoleIDs)
		if err != nil {
			log.Errorln("Failed to get roles:", err)
			return
		}

		semaphore.IdleMust(popup.Box.Add, r)
		semaphore.IdleMust(popup.Box.ShowAll)
	}()

	return popup.Popover.Popover
}
