package user

import (
	"fmt"
	"html"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/components/avatar"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

const AvatarSize = 32
const AvatarSizeGtk = gtk.IconSizeDND

type Container struct {
	*gtk.Box

	Avatar *avatar.WithStatus
	AStyle *gtk.StyleContext

	// Right side Box is not here
	LabelBox *gtk.Box
	Name     *gtk.Label
	Activity *gtk.Label

	NameValue string
}

func New() *Container {
	labelBox := gtk.NewBox(gtk.OrientationVertical, 0)
	labelBox.SetMarginStart(8)
	labelBox.SetVAlign(gtk.AlignCenter)
	labelBox.Show()

	l := gtk.NewLabel("")
	l.SetLineWrap(false)
	l.SetEllipsize(pango.EllipsizeEnd)
	l.Show()
	l.SetXAlign(0.0)
	l.SetHAlign(gtk.AlignStart)
	labelBox.Add(l)

	a := avatar.NewWithStatus(AvatarSize)
	a.SetFromIconName("avatar-default-symbolic", 0)
	a.SetVAlign(gtk.AlignCenter)
	a.SetHAlign(gtk.AlignCenter)
	gtkutils.Margin4(a, 2, 2, 8, 0)
	a.Show()

	s := a.StyleContext()
	s.AddClass("status")

	b := gtk.NewBox(gtk.OrientationHorizontal, 0)
	b.Show()
	b.Add(a)
	b.Add(labelBox)

	c := &Container{
		Box:      b,
		Avatar:   a,
		AStyle:   s,
		LabelBox: labelBox,
		Name:     l,
	}
	return c
}

func (c *Container) UpdateActivity(ac *discord.Activity) {
	// if a == nil, then we should reset the label to not show any game.
	if ac == nil {
		if c.Activity != nil {
			c.LabelBox.Remove(c.Activity)
			c.Activity = nil
		}

		return
	}

	if c.Activity == nil {
		lAct := gtk.NewLabel("")
		lAct.Show()
		lAct.SetUseMarkup(true)
		lAct.SetHAlign(gtk.AlignStart)
		lAct.SetEllipsize(pango.EllipsizeEnd)

		c.Activity = lAct
		c.LabelBox.Add(lAct)
	}

	// else, update game
	var game string

	switch ac.Type {
	case discord.GameActivity:
		game = "Playing " + ac.Name
	case discord.ListeningActivity:
		game = "Listening to " + ac.Name
	case discord.StreamingActivity:
		game = "Streaming " + ac.Details
	case discord.CustomActivity:
		switch {
		case ac.Emoji == nil:
			game = ac.State
		case ac.Emoji.ID.IsValid():
			game = ":" + ac.Emoji.Name + ": " + ac.State
		default:
			game = ac.Emoji.Name + " " + ac.State
		}
	}

	c.Activity.SetMarkup(`<span size="smaller">` + html.EscapeString(game) + "</span>")
	c.Activity.SetTooltipText(game)
}

func (c *Container) UpdateStatus(status gateway.Status) {
	c.Avatar.SetStatus(status)
}

func (c *Container) UpdateUser(u discord.User) {
	c.NameValue = u.Username
	c.Avatar.SetInitials(c.NameValue)
	c.Name.SetText(c.NameValue)
	c.Name.SetTooltipText(u.Username + "#" + u.Discriminator)
	c.Name.SetUseMarkup(false)
}

func (c *Container) UpdateMember(m discord.Member, guild discord.Guild) {
	var name = m.User.Username
	if m.Nick != "" {
		name = m.Nick
	}

	c.NameValue = name

	// Escape name
	name = html.EscapeString(name)
	colored := name

	// Check role color
	if color := discord.MemberColor(guild, m); color > 0 {
		colored = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, name)
	}

	// Set name
	c.Avatar.SetInitials(c.NameValue)
	c.Name.SetMarkup(colored)
	c.Name.SetTooltipText(name)
}

func (c *Container) UpdateAvatar(url discord.URL) {
	c.Avatar.SetURL(url + "?size=64")
}
