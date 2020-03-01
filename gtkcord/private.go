package gtkcord

import (
	"fmt"
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const DMAvatarSize = 38

type PrivateChannels struct {
	gtkutils.ExtendedWidget

	GuildRow struct {
		gtkutils.ExtendedWidget
		Style *gtk.StyleContext
		class string
	}

	List   *gtk.ListBox
	Scroll *gtk.ScrolledWindow

	Channels []*PrivateChannel
	// channels map[discord.Snowflake]*PrivateChannel

}

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

	Messages *Messages

	lastStatusClass string // avatar
	stateClass      string // row styl
}

// thread-safe
func newPrivateChannels(chs []discord.Channel) (pcs *PrivateChannels) {
	must(func() {
		l, _ := gtk.ListBoxNew()
		gtkutils.InjectCSSUnsafe(l, "dmchannels", "")

		cs, _ := gtk.ScrolledWindowNew(nil, nil)
		cs.SetSizeRequest(ChannelsWidth, -1)
		cs.Add(l)

		pcs = &PrivateChannels{
			ExtendedWidget: cs,

			List:   l,
			Scroll: cs,
		}

		l.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
			if len(pcs.Channels) == 0 {
				return
			}

			row := pcs.Channels[r.GetIndex()]
			go App.loadPrivate(row)
		})
	})

	icon := App.parser.GetIcon("system-users-symbolic", IconSize/3*2)
	must(func() {
		r, _ := gtk.ListBoxRowNew()
		r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
		r.SetHAlign(gtk.ALIGN_FILL)
		r.SetVAlign(gtk.ALIGN_CENTER)
		r.SetTooltipMarkup("<b>Private Messages</b>")
		r.SetActivatable(true)

		gtkutils.InjectCSSUnsafe(r, "friends-button", "")

		i, _ := gtk.ImageNewFromPixbuf(icon)
		i.SetHAlign(gtk.ALIGN_CENTER)
		i.SetVAlign(gtk.ALIGN_CENTER)
		r.Add(i)

		pcs.GuildRow.ExtendedWidget = r
		pcs.GuildRow.Style, _ = r.GetStyleContext()
		pcs.GuildRow.Style.AddClass("dmbutton")
		pcs.GuildRow.Style.AddClass("guild")
	})

	pcs.load(chs)
	return pcs
}

func (pcs *PrivateChannels) ensureSelected() {
	if r := pcs.List.GetSelectedRow(); r != nil {
		return
	}

	if len(pcs.Channels) == 0 {
		return
	}

	pcs.List.SelectRow(pcs.Channels[0].ListBoxRow)
	go App.loadPrivate(pcs.Channels[0])
}

// thread-safe
func (pcs *PrivateChannels) load(channels []discord.Channel) {
	// Sort the direct message channels, so that the latest messages come first.
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].LastMessageID > channels[j].LastMessageID
	})

	// fuck diffing.
	if len(pcs.Channels) > 0 {
		must(func() {
			for _, ch := range pcs.Channels {
				pcs.List.Remove(ch)
				// delete(pcs.channels, ch.ID)
			}
		})
	}

	// TODO: mutex
	pcs.Channels = make([]*PrivateChannel, 0, len(channels))

	for _, channel := range channels {
		w := newPrivateChannel(channel)
		w.Parent = pcs

		if w.stateClass == "pinged" {
			pcs.setUnread(true)
		}

		pcs.Channels = append(pcs.Channels, w)
	}

	async(func() {
		for _, chw := range pcs.Channels {
			pcs.List.Insert(chw, -1)
		}
		pcs.ShowAll()
	})
}

func (pcs *PrivateChannels) setButtonClass(class string) {
	gtkutils.DiffClass(&pcs.GuildRow.class, class, pcs.GuildRow.Style)
}

func (pcs *PrivateChannels) updatePresence(p discord.Presence) {
	for _, ch := range pcs.Channels {
		if ch.Recp == p.User.ID && !ch.Group {
			ch.UpdateStatus(p.Status)
			ch.UpdateActivity(p.Game)
			break
		}
	}
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

	name = escape(name)
	icon := App.parser.GetIcon("network-workgroup-symbolic", DMAvatarSize)

	must(func() {
		l, _ := gtk.LabelNew(name)
		l.SetUseMarkup(true)
		l.SetMarginStart(8)
		l.SetEllipsize(pango.ELLIPSIZE_END)

		a, _ := gtk.ImageNewFromPixbuf(icon)
		margin4(a, 4, 4, 8, 0)

		s, _ := a.GetStyleContext()
		s.AddClass("status")

		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		b.Add(a)
		b.Add(l)

		r, _ := gtk.ListBoxRowNew()
		r.Add(b)

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
		}

		if len(ch.DMRecipients) > 0 {
			pc.Recp = ch.DMRecipients[0].ID
		}
	})

	if rs := App.State.FindLastRead(pc.ID); rs != nil {
		pc.setUnread(rs.LastMessageID != ch.LastMessageID)
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

func (pc *PrivateChannel) ackLatest() {
	last := pc.Messages.LastID()
	if !last.Valid() {
		return
	}
	App.State.MarkRead(pc.ID, last)
}

func (pc *PrivateChannel) loadMessages() error {
	if pc.Messages == nil {
		m, err := newMessages(pc.ID)
		if err != nil {
			return err
		}

		if pc.Parent != nil {
			m.OnInsert = func() {
				sort.Slice(pc.Parent.Channels, func(i, j int) bool {
					return pc.Parent.Channels[i] == pc
				})

				must(func() {
					// Bring this channel to the top:
					pc.Parent.List.Remove(pc.ListBoxRow)
					pc.Parent.List.Prepend(pc.ListBoxRow)
				})

				pc.ackLatest()
			}
		}

		pc.Messages = m
	}

	defer func() {
		go pc.ackLatest()
	}()

	return pc.Messages.reset()
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

		async(func() {
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

	pc.Game = escape(pc.Game)

	label := fmt.Sprintf(
		"%s\n"+`<span size="smaller">%s</span>`,
		pc.Name, pc.Game,
	)

	async(func() {
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
