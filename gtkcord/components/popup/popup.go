package popup

import (
	"fmt"
	"html"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"

	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
)

const (
	PopupAvatarSize = 96
	PopupImageSize  = 48 // rich presence image
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

	Children gtk.Widgetter
}

func NewPopover(relative gtk.Widgetter) *Popover {
	p := gtk.NewPopover(relative)
	p.SetSizeRequest(PopupWidth, -1)
	style := p.StyleContext()

	gtkutils.InjectCSS(p, "user-info", "")

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

		gtk.BaseWidget(popover.Children).Destroy()
		popover.Children = nil
	})

	return popover
}

func (p *Popover) SetChildren(children gtk.Widgetter) {
	if p.Children != nil {
		p.Popover.Remove(p.Children)
	}
	p.Children = children
	p.Popover.Add(children)
}

// PopoverCreator describes a popover creator function.
type PopoverCreator = func(p *gtk.Popover) gtk.Widgetter

func NewDynamicPopover(relative gtk.Widgetter, create PopoverCreator) *Popover {
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
	btn := gtk.NewModelButton()
	btn.SetLabel(markup)

	// Set the label
	l, ok := btn.Child().(*gtk.Label)
	if ok {
		l.SetUseMarkup(true)
		l.SetHAlign(gtk.AlignStart)
	}

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

	Avatar      *roundimage.Image
	AvatarStyle *gtk.StyleContext // .avatar
	// check window/css.go header for status_* colors
	lastAvatarClass string

	Username *gtk.Label

	Activity *UserPopupActivity
}

// not thread safe
func NewUserPopup(relative gtk.Widgetter) *UserPopup {
	return NewUserPopupCustom(relative, NewUserPopupBody())
}

// not thread safe
func NewUserPopupCustom(relative gtk.Widgetter, body *UserPopupBody) *UserPopup {
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
	main := gtk.NewGrid()
	main.Show()
	gtkutils.InjectCSS(main, "popup-grid", "")

	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.SetSizeRequest(PopupWidth, -1)
	b.SetMarginTop(10)
	b.SetMarginBottom(10)

	gtkutils.InjectCSS(b, "popup-user", "")
	main.Attach(b, 0, 0, 1, 1)

	iAvatar := roundimage.NewImage(0)
	iAvatar.SetFromIconName("user-info", int(gtk.IconSizeLargeToolbar))
	iAvatar.SetHAlign(gtk.AlignCenter)
	iAvatar.SetSizeRequest(PopupAvatarSize, PopupAvatarSize)
	iAvatar.SetMarginTop(10)
	iAvatar.SetMarginBottom(7)
	b.Add(iAvatar)

	sAvatar := iAvatar.StyleContext()
	sAvatar.AddClass("avatar")
	sAvatar.AddClass("status")

	l := gtk.NewLabel("?")
	l.SetMarginEnd(7)
	l.SetEllipsize(pango.EllipsizeEnd)
	l.SetJustify(gtk.JustifyCenter)
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
func (b *UserPopupBody) Attach(w gtk.Widgetter, row int) {
	if row > b.GridMax {
		b.GridMax = row
	}
	b.Grid.Attach(w, 0, row, 1, 1)
}

func (b *UserPopupBody) setAvatarClass(class string) {
	gtkutils.DiffClass(&b.lastAvatarClass, class, b.AvatarStyle)
}

func (b *UserPopupBody) Update(u discord.User) {
	if b.Username != nil {
		b.Username.SetMarkup(formatUser(u))
	}

	if u.Avatar != "" {
		b.updateAvatar(u.AvatarURL())
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
		b.updateAvatar(u.AvatarURL())
	}
}

func (b *UserPopupBody) updateAvatar(url string) {
	cache.SetImageURLScaled(b.Avatar, url+"?size=128", PopupAvatarSize, PopupAvatarSize)
}

func (b *UserPopupBody) UpdateActivity(a *discord.Activity) {
	if a == nil {
		if b.Activity != nil {
			b.Grid.Remove(b.Activity)
			b.Activity = nil
		}
		return
	}

	if b.Activity == nil {
		b.Activity = NewUserPopupActivity()
		b.Attach(b.Activity, 1)
	}

	b.Activity.Update(*a)

	switch a.Type {
	case discord.GameActivity, discord.ListeningActivity, discord.StreamingActivity:
		b.setAvatarClass("unknown")
	}

	b.Grid.ShowAll()
}

func (b *UserPopupBody) UpdateStatus(status gateway.Status) {
	switch status {
	case gateway.OnlineStatus:
		b.Avatar.SetTooltipText("Online")
		b.setAvatarClass("online")
	case gateway.DoNotDisturbStatus:
		b.Avatar.SetTooltipText("Busy")
		b.setAvatarClass("busy")
	case gateway.IdleStatus:
		b.Avatar.SetTooltipText("Idle")
		b.setAvatarClass("idle")
	case gateway.InvisibleStatus, gateway.OfflineStatus:
		b.Avatar.SetTooltipText("Offline")
		b.setAvatarClass("offline")
	case gateway.UnknownStatus:
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

func NewUserPopupRoles(s *ningen.State, guildID discord.GuildID, uID discord.UserID) *UserPopupRoles {
	s = s.Offline()

	roleLabel := gtk.NewLabel("Roles")
	roleLabel.SetMarginTop(SectionPadding)
	roleLabel.SetMarginBottom(0)
	roleLabel.SetMarginLeft(SectionPadding)
	roleLabel.SetMarginRight(SectionPadding)
	roleLabel.SetHAlign(gtk.AlignStart)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Add(roleLabel)

	popup := &UserPopupRoles{
		Box:    box,
		Header: roleLabel,
	}

	// TODO: optimize this
	member, _ := s.Member(guildID, uID)
	if member == nil {
		roleLabel.SetLabel("Unknown member")
		roleLabel.SetMarginBottom(SectionPadding)
		return popup
	}

	if len(member.RoleIDs) == 0 {
		roleLabel.SetLabel("No Roles")
		roleLabel.SetMarginBottom(SectionPadding)
		return popup
	}

	fb := gtk.NewFlowBox()
	fb.SetSelectionMode(gtk.SelectionNone)
	fb.SetHAlign(gtk.AlignFill)
	fb.SetHAlign(gtk.AlignStart)
	gtkutils.Margin(fb, SectionPadding)

	box.Add(fb)

	popup.Main = fb
	popup.Roles = make([]UserPopupRole, 0, len(member.RoleIDs))

	for _, id := range member.RoleIDs {
		r, err := s.Role(guildID, id)
		if err != nil {
			log.Errorln("failed to get role for popup:", err)
			continue
		}

		color := r.Color
		if color == 0 {
			color = 0x555555
		}

		hex := fmt.Sprintf("#%06X", color)

		l := gtk.NewLabel("")
		l.SetTooltipText(r.Name)
		l.SetLabel(" " + r.Name + " ")
		l.SetSingleLineMode(true)
		l.SetEllipsize(pango.EllipsizeEnd)
		l.SetMaxWidthChars(20)

		gtkutils.InjectCSS(l, "", `
			label {
				border: 1px solid `+hex+`;
				border-left-width: 5px;
			}
		`)

		c := gtk.NewFlowBoxChild()
		c.Add(l)
		fb.Insert(c, -1)

		popup.Roles = append(popup.Roles, UserPopupRole{
			FlowBoxChild: c,
			Main:         l,
		})
	}

	return popup
}
