package popup

import (
	"fmt"
	"html"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

const (
	PopupAvatarSize = 96
	PopupWidth      = 240

	OfflineColor = 0x747F8D
	BusyColor    = 0xF04747
	IdleColor    = 0xFAA61A
	OnlineColor  = 0x43B581
)

type Popover struct {
	*gtk.Popover
	Style    *gtk.StyleContext
	oldClass string

	Children gtkutils.WidgetDestroyer
}

func NewPopover(relative gtk.IWidget) *Popover {
	p, _ := gtk.PopoverNew(relative)
	p.SetSizeRequest(PopupWidth, -1)
	style, _ := p.GetStyleContext()

	gtkutils.InjectCSSUnsafe(p, "user-info", "")

	popover := &Popover{
		Popover: p,
		Style:   style,
	}
	popover.Connect("closed", func() {
		popover.Style.RemoveClass(popover.oldClass)
		popover.oldClass = ""

		if popover.Children == nil {
			return
		}

		popover.Children.Destroy()
		popover.Children = nil
	})

	return popover
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
	relative.Connect("clicked", func() {
		if w := create(p.Popover); w != nil {
			p.SetChildren(w)
			p.ShowAll()
		}
	})

	// LMAO
	if mb, ok := relative.(*gtk.MenuButton); ok {
		mb.SetPopover(p.Popover)
		mb.SetUsePopover(true)
	}

	return p
}

func NewModelButton(markup string) *gtk.ModelButton {
	// Create the button
	btn, _ := gtk.ModelButtonNew()
	btn.SetLabel(markup)

	// Set the label
	c, err := btn.GetChild()
	if err != nil {
		log.Errorln("Failed to get child of ModelButton")
		return btn
	}

	l := &gtk.Label{Widget: *c}
	l.SetUseMarkup(true)
	l.SetHAlign(gtk.ALIGN_START)

	btn.ShowAll()
	return btn
}

func NewButton(markup string, callback func()) *gtk.ModelButton {
	btn := NewModelButton(markup)
	btn.Connect("button-release-event", func() bool {
		callback()
		return true
	})

	return btn
}

type UserPopup struct {
	*Popover
	*UserPopupBody
}

type UserPopupBody struct {
	*gtk.Grid
	GridMax int

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
	main, _ := gtk.GridNew()
	main.Show()
	gtkutils.InjectCSSUnsafe(main, "popup-grid", "")

	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.SetSizeRequest(PopupWidth, -1)
	b.SetMarginTop(10)
	b.SetMarginBottom(10)

	gtkutils.InjectCSSUnsafe(b, "popup-user", "")
	main.Attach(b, 0, 0, 1, 1)

	iAvatar, _ := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_LARGE_TOOLBAR)
	iAvatar.SetHAlign(gtk.ALIGN_CENTER)
	iAvatar.SetSizeRequest(PopupAvatarSize, PopupAvatarSize)
	iAvatar.SetMarginTop(10)
	iAvatar.SetMarginBottom(7)
	b.Add(iAvatar)

	sAvatar, _ := iAvatar.GetStyleContext()
	sAvatar.AddClass("avatar")
	sAvatar.AddClass("status")

	l, _ := gtk.LabelNew("?")
	l.SetMarginEnd(7)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.SetJustify(gtk.JUSTIFY_CENTER)
	b.Add(l)

	return &UserPopupBody{
		Grid:        main,
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

// row > 0
func (b *UserPopupBody) Attach(w gtk.IWidget, row int) {
	if row > b.GridMax {
		b.GridMax = row
	}
	b.Grid.Attach(w, 0, row, 1, 1)
}

func (b *UserPopupBody) setAvatarClass(class string) {
	gtkutils.DiffClassUnsafe(&b.lastAvatarClass, class, b.AvatarStyle)
}

func (b *UserPopupBody) Update(u discord.User) {
	if b.Username != nil {
		b.Username.SetMarkup(formatUser(u))
	}

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
		url+"?size=128", b.Avatar, PopupAvatarSize, PopupAvatarSize, cache.Round)
	if err != nil {
		log.Errorln("Failed to get the pixbuf avatar icon:", err)
		return
	}
}

func (b *UserPopupBody) UpdateActivity(a *discord.Activity) {
	if a == nil {
		if b.Activity != nil {
			b.Grid.Remove(b.Activity)
			b.Activity = nil
			// b.setClass("")
		}
		return
	}

	if b.Activity == nil {
		b.Activity = NewUserPopupActivity()
		b.Attach(b.Activity, 1)
	}

	b.Activity.UpdateUnsafe(*a)

	switch a.Type {
	case discord.GameActivity, discord.ListeningActivity, discord.StreamingActivity:
		b.setAvatarClass("unknown")
	}

	b.Grid.ShowAll()
}

func (b *UserPopupBody) UpdateStatus(status discord.Status) {
	switch status {
	case discord.OnlineStatus:
		b.Avatar.SetTooltipText("Online")
		b.setAvatarClass("online")
	case discord.DoNotDisturbStatus:
		b.Avatar.SetTooltipText("Busy")
		b.setAvatarClass("busy")
	case discord.IdleStatus:
		b.Avatar.SetTooltipText("Idle")
		b.setAvatarClass("idle")
	case discord.InvisibleStatus, discord.OfflineStatus:
		b.Avatar.SetTooltipText("Offline")
		b.setAvatarClass("offline")
	case discord.UnknownStatus:
		b.setAvatarClass("unknown")
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

// func SpawnUserPopup(s *ningen.State, guildID, userID discord.Snowflake) *gtk.Popover {
// 	popup := NewUserPopup(nil)

// 	go func() {
// 		u, err := s.User(userID)
// 		if err != nil {
// 			log.Errorln("Failed to get user:", err)
// 			return
// 		}

// 		p, err := s.Presence(guildID, u.ID)
// 		if err == nil {
// 			popup.UpdateStatus(p.Status)
// 			popup.UpdateActivity(p.Game)
// 		}

// 		if !guildID.Valid() {
// 			popup.Update(*u)
// 			return
// 		}

// 		// fetch above presence if error not nil
// 		if err != nil {
// 			s.RequestMember(guildID, userID)
// 		}

// 		m, err := s.Member(guildID, u.ID)
// 		if err != nil {
// 			popup.Update(*u)
// 			return
// 		}

// 		popup.UpdateMember(*m)

// 		r, err := NewUserPopupRoles(s, guildID, m.RoleIDs)
// 		if err != nil {
// 			log.Errorln("Failed to get roles:", err)
// 			return
// 		}

// 		semaphore.IdleMust(popup.Attach, r, 2)
// 		semaphore.IdleMust(popup.Grid.ShowAll)
// 	}()

// 	return popup.Popover.Popover
// }
