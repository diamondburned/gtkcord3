package user

import (
	"fmt"
	"html"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const AvatarSize = 32

type Container struct {
	*gtk.Box

	Avatar *gtk.Image
	AStyle *gtk.StyleContext

	// Right side Box is not here
	LabelBox *gtk.Box
	Name     *gtk.Label
	Activity *gtk.Label

	NameValue string

	lastStatusClass string
}

// NOT thread-safe
func New() *Container {
	labelBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	labelBox.SetMarginStart(8)
	labelBox.SetVAlign(gtk.ALIGN_CENTER)
	labelBox.Show()

	l, _ := gtk.LabelNew("")
	l.SetLineWrap(false)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.Show()
	l.SetXAlign(0.0)
	l.SetHAlign(gtk.ALIGN_START)
	labelBox.Add(l)

	a, _ := gtk.ImageNew()
	a.Show()
	a.SetVAlign(gtk.ALIGN_CENTER)
	a.SetHAlign(gtk.ALIGN_CENTER)
	gtkutils.ImageSetIcon(a, "avatar-default-symbolic", AvatarSize)
	gtkutils.Margin4(a, 2, 2, 8, 0)

	s, _ := a.GetStyleContext()
	s.AddClass("status")

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
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
	c.setStatusClass("offline")
	return c
}

func (c *Container) setStatusClass(class string) {
	gtkutils.DiffClassUnsafe(&c.lastStatusClass, class, c.AStyle)
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
		lAct, _ := gtk.LabelNew("")
		lAct.Show()
		lAct.SetUseMarkup(true)
		lAct.SetHAlign(gtk.ALIGN_START)
		lAct.SetEllipsize(pango.ELLIPSIZE_END)

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
		var emoji = ac.Emoji.Name
		if ac.Emoji.ID.Valid() { // if the emoji is custom:
			emoji = ":" + emoji + ":"
		}

		game = emoji + " " + ac.State
	}

	c.Activity.SetMarkup(`<span size="smaller">` + html.EscapeString(game) + "</span>")
	c.Activity.SetTooltipText(game)
}

func (c *Container) UpdateStatus(status discord.Status) {
	switch status {
	case discord.OnlineStatus:
		c.setStatusClass("online")
	case discord.DoNotDisturbStatus:
		c.setStatusClass("busy")
	case discord.IdleStatus:
		c.setStatusClass("idle")
	case discord.InvisibleStatus, discord.OfflineStatus, discord.UnknownStatus:
		c.setStatusClass("offline")
	}
}

func (c *Container) UpdateUser(u discord.User) {
	c.NameValue = u.Username
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
	var colored = name

	// Check role color
	if color := discord.MemberColor(guild, m); color > 0 {
		colored = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, name)
	}

	// Set name
	c.Name.SetMarkup(colored)
	c.Name.SetTooltipText(name)
}

func (c *Container) UpdatePresence(m discord.Presence, guild discord.Guild) {
	var name = m.User.Username
	if m.Nick != "" {
		name = m.Nick
	}

	c.NameValue = name

	// Escape name
	name = html.EscapeString(name)
	var colored = name

	// Check role color
	if color := PresenceColor(guild, m); color > 0 {
		colored = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, name)
	}

	// Set name
	c.Name.SetMarkup(colored)
	c.Name.SetTooltipText(name)
}

func (c *Container) UpdateAvatar(url string) {
	if url == "" {
		return
	}

	go func() {
		err := cache.SetImageScaled(url+"?size=64", c.Avatar, AvatarSize, AvatarSize, cache.Round)
		if err != nil {
			log.Errorln("Failed to get DM avatar", url+":", err)
		}
	}()
}

func PresenceColor(guild discord.Guild, member discord.Presence) discord.Color {
	var c = discord.DefaultMemberColor
	var pos int

	for _, r := range guild.Roles {
		for _, mr := range member.RoleIDs {
			if mr != r.ID {
				continue
			}

			if r.Color > 0 && r.Position > pos {
				c = r.Color
				pos = r.Position
			}
		}
	}

	return c
}
