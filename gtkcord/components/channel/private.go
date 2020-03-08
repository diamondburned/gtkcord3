package channel

import (
	"fmt"
	"html"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const (
	DMAvatarSize = 38
	IconPadding  = 8
)

type PrivateChannel struct {
	*gtk.ListBoxRow
	Main  *gtk.Box
	Style *gtk.StyleContext

	Avatar *gtk.Image
	AStyle *gtk.StyleContext

	Label *gtk.Label

	ID   discord.Snowflake
	Name string
	Game string

	Group bool

	lastStatusClass string // avatar
	stateClass      string // row style
}

// thread-safe
func newPrivateChannel(ch discord.Channel) (pc *PrivateChannel) {
	var name = ch.Name
	if name == "" {
		var names = make([]string, len(ch.DMRecipients))
		for i, p := range ch.DMRecipients {
			names[i] = p.Username
		}

		name = humanize.Strings(names)
	}

	l, _ := gtk.LabelNew(html.EscapeString(name))
	l.SetUseMarkup(true)
	l.SetMarginStart(8)
	l.SetEllipsize(pango.ELLIPSIZE_END)

	a, _ := gtk.ImageNew()
	gtkutils.ImageSetIcon(a, "network-workgroup-symbolic", DMAvatarSize)
	gtkutils.Margin4(a, 4, 4, 8, 0)

	s, _ := a.GetStyleContext()
	s.AddClass("status")

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.Add(a)
	b.Add(l)

	r, _ := gtk.ListBoxRowNew()
	r.Add(b)
	// set the channel ID to searches
	r.SetProperty("name", ch.ID.String())

	rs, _ := r.GetStyleContext()
	rs.AddClass("dmchannel")

	pc = &PrivateChannel{
		ListBoxRow: r,
		Main:       b,
		Style:      rs,

		Avatar: a,
		AStyle: s,

		Label: l,

		ID:   ch.ID,
		Name: name,
		// Group: ch.Type == discord.GroupDM,
	}

	pc.setStatusClass("offline")

	// if ch.Type != discord.DirectMessage {
	// 	return pc
	// }

	return pc
}

func _ChIDFromRow(row *gtk.ListBoxRow) string {
	v, err := row.GetProperty("name")
	if err != nil {
		log.Errorln("Failed to get channel ID:", err)
		return ""
	}
	return v.(string)
}

func (pc *PrivateChannel) setStatusClass(class string) {
	gtkutils.DiffClassUnsafe(&pc.lastStatusClass, class, pc.AStyle)
}
func (pc *PrivateChannel) setClass(class string) {
	gtkutils.DiffClassUnsafe(&pc.stateClass, class, pc.Style)
}

func (pc *PrivateChannel) updateActivity(ac *discord.Activity) {
	// if a == nil, then we should reset the label to not show any game.
	if ac == nil {
		// only if there was a game before
		if pc.Game == "" {
			return
		}

		pc.Label.SetMarkup(pc.Name)
		pc.Label.SetTooltipMarkup(pc.Name)

		return
	}

	// else, update game
	switch ac.Type {
	case discord.GameActivity:
		pc.Game = "Playing " + ac.Name
	case discord.ListeningActivity:
		pc.Game = "Listening to " + ac.Name
	case discord.StreamingActivity:
		pc.Game = "Streaming " + ac.Details
	case discord.CustomActivity:
		pc.Game = ac.Emoji.String() + ac.State
	}

	pc.Game = html.EscapeString(pc.Game)

	label := fmt.Sprintf(
		"%s\n"+`<span size="smaller">%s</span>`,
		pc.Name, pc.Game,
	)

	pc.Label.SetMarkup(label)
	pc.Label.SetTooltipMarkup(label)
}

func (pc *PrivateChannel) updateStatus(status discord.Status) {
	switch status {
	case discord.OnlineStatus:
		pc.setStatusClass("online")
	case discord.DoNotDisturbStatus:
		pc.setStatusClass("busy")
	case discord.IdleStatus:
		pc.setStatusClass("idle")
	case discord.InvisibleStatus, discord.OfflineStatus, discord.UnknownStatus:
		// Unknown is offline too.
		pc.setStatusClass("offline")
	}
}

func (pc *PrivateChannel) updateAvatar(url string) {
	err := cache.SetImageScaled(url+"?size=64", pc.Avatar, DMAvatarSize, DMAvatarSize, cache.Round)
	if err != nil {
		log.Errorln("Failed to get DM avatar", url+":", err)
	}
}

func (pc *PrivateChannel) setUnread(unread bool) {
	if unread {
		pc.setClass("pinged")
	} else {
		pc.setClass("")
	}
}
