package channel

import (
	"fmt"
	"html"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type PrivateChannel struct {
	Parent *PrivateChannels

	*gtk.ListBoxRow
	Main  *gtk.Box
	Style *gtk.StyleContext

	Avatar *gtk.Image
	AStyle *gtk.StyleContext

	Label *gtk.Label

	ID   discord.Snowflake
	Recp discord.Snowflake // first recipient
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

	name = html.EscapeString(name)
	icon := icons.GetIcon("network-workgroup-symbolic", DMAvatarSize)

	semaphore.IdleMust(func() {
		l, _ := gtk.LabelNew(name)
		l.SetUseMarkup(true)
		l.SetMarginStart(8)
		l.SetEllipsize(pango.ELLIPSIZE_END)

		a, _ := gtk.ImageNewFromPixbuf(icon)
		gtkutils.Margin4(a, 4, 4, 8, 0)

		s, _ := a.GetStyleContext()
		s.AddClass("status")

		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		b.Add(a)
		b.Add(l)

		r, _ := gtk.ListBoxRowNew()
		r.Add(b)
		// set the channel ID to name
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

			ID:    ch.ID,
			Name:  name,
			Group: true,
			// Set the property. We'll need this for sorting.
			LastMsg: ch.LastMessageID,
		}

		if len(ch.DMRecipients) > 0 {
			pc.Recp = ch.DMRecipients[0].ID
		}
	})

	if rs := App.State.FindLastRead(pc.ID); rs != nil {
		pc.updateReadState(rs)
	}

	if ch.Type != discord.DirectMessage {
		return pc
	}

	pc.setStatusClass("offline")
	pc.Group = false

	if len(ch.DMRecipients) > 0 && ch.DMRecipients[0].Avatar != "" {
		url := ch.DMRecipients[0].AvatarURL() + "?size=64"

		go func() {
			err := cache.SetImageScaled(url, pc.Avatar, DMAvatarSize, DMAvatarSize, cache.Round)
			if err != nil {
				log.Errorln("Failed to get DM avatar", url+":", err)
			}
		}()

		p, err := App.State.Presence(0, ch.DMRecipients[0].ID)
		if err == nil {
			pc.UpdateStatus(p.Status)
			pc.UpdateActivity(p.Game)
		}
	}

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

func (pc *PrivateChannel) ackLatest(m *message.Message) {
	App.State.MarkRead(pc.ID, m.ID, m.AuthorID != App.Me.ID)
}

func (pc *PrivateChannel) setStatusClass(class string) {
	gtkutils.DiffClass(&pc.lastStatusClass, class, pc.AStyle)
}
func (pc *PrivateChannel) setClass(class string) {
	gtkutils.DiffClass(&pc.stateClass, class, pc.Style)
}

func (pc *PrivateChannel) UpdateActivity(ac *discord.Activity) {
	// if a == nil, then we should reset the label to not show any game.
	if ac == nil {
		// only if there was a game before
		if pc.Game == "" {
			return
		}

		semaphore.Async(func() {
			pc.Label.SetMarkup(pc.Name)
			pc.Label.SetTooltipMarkup(pc.Name)
		})

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

	semaphore.Async(func() {
		pc.Label.SetMarkup(label)
		pc.Label.SetTooltipMarkup(label)
	})
}

func (pc *PrivateChannel) UpdateStatus(status discord.Status) {
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

func (pc *PrivateChannel) updateReadState(rs *gateway.ReadState) {
	if rs == nil {
		pc.setUnread(false)
		return
	}

	unread := pc.LastMsg != rs.LastMessageID
	pc.setUnread(unread)

	if pc.Parent != nil {
		must(func() {
			if pc.ListBoxRow.GetIndex() != 0 {
				pc.Parent.List.InvalidateSort()
			}
		})
	}
}

func (pc *PrivateChannel) setUnread(unread bool) {
	if unread {
		pc.setClass("pinged")
	} else {
		pc.setClass("")
	}

	if pc.Parent != nil {
		pc.Parent.setUnread(unread)

		must(func() {
			if pc.ListBoxRow.GetIndex() == 0 {
				return
			}
			pc.Parent.List.InvalidateSort()
		})
	}
}
