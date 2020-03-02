package gtkcord

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
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

	Search *gtk.Entry
	search string

	// Channels map[discord.Snowflake]*PrivateChannel
	Channels map[string]*PrivateChannel
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
	LastMsg  discord.Snowflake

	lastStatusClass string // avatar
	stateClass      string // row styl
}

// thread-safe
func newPrivateChannels(chs []discord.Channel) (pcs *PrivateChannels) {
	log.Infoln("Ch count:", len(chs))
	must(func() {
		l, _ := gtk.ListBoxNew()
		gtkutils.InjectCSSUnsafe(l, "dmchannels", "")

		cs, _ := gtk.ScrolledWindowNew(nil, nil)
		cs.SetSizeRequest(ChannelsWidth, -1)
		cs.SetVExpand(true)
		cs.Add(l)

		e, _ := gtk.EntryNew()
		e.SetPlaceholderText("Find conversation...")

		b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		b.Add(e)
		b.Add(cs)

		pcs = &PrivateChannels{
			ExtendedWidget: b,

			List:   l,
			Scroll: cs,
			Search: e,
		}

		e.Connect("changed", func() {
			t, err := e.GetText()
			if err != nil {
				log.Errorln("Failed to get text from dmchannels entry:", err)
				return
			}

			pcs.search = strings.ToLower(t)
			pcs.List.InvalidateFilter()
		})

		l.SetFilterFunc(pcs.filter, 0)
		l.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
			if len(pcs.Channels) == 0 {
				return
			}

			rw, ok := pcs.Channels[_ChIDFromRow(r)]
			if !ok {
				log.Errorln("Failed to find channel")
				return
			}

			go App.loadPrivate(rw)
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

// thread-safe
func (pcs *PrivateChannels) load(channels []discord.Channel) {
	// fuck diffing.
	if len(pcs.Channels) > 0 {
		must(func() {
			for _, ch := range pcs.Channels {
				pcs.List.Remove(ch)
			}

			// Stop sorting for now:
			pcs.List.SetSortFunc(nil, 0)
		})
	}

	// TODO: mutex
	pcs.Channels = make(map[string]*PrivateChannel, len(channels))

	for _, channel := range channels {
		w := newPrivateChannel(channel)
		w.Parent = pcs

		if w.stateClass == "pinged" {
			pcs.setUnread(true)
		}

		pcs.Channels[channel.ID.String()] = w
	}

	async(func() {
		for _, chw := range pcs.Channels {
			pcs.List.Insert(chw, -1)
		}
		pcs.List.SetSortFunc(pcs.sort, 0)
		pcs.List.InvalidateSort()
		pcs.ShowAll()
	})
}

func (pcs *PrivateChannels) Selected() *PrivateChannel {
	if len(pcs.Channels) == 0 {
		return nil
	}

	r := pcs.List.GetSelectedRow()
	if r == nil {
		r = pcs.List.GetRowAtIndex(0)
		pcs.List.SelectRow(r)
	}

	rw, ok := pcs.Channels[_ChIDFromRow(r)]
	if !ok || rw == nil {
		log.Errorln("Failed to find channel row")
	}
	return rw
}

func (pcs *PrivateChannels) filter(r *gtk.ListBoxRow, _ uintptr) bool {
	if pcs.search == "" {
		return true
	}

	pc, ok := pcs.Channels[_ChIDFromRow(r)]
	if !ok {
		log.Errorln("Failed to get channel for filter")
		return false
	}

	return strings.Contains(strings.ToLower(pc.Name), pcs.search)
}

func (pcs *PrivateChannels) sort(r1, r2 *gtk.ListBoxRow, _ uintptr) int {
	p1, ok := pcs.Channels[_ChIDFromRow(r1)]
	if !ok {
		log.Errorln("Failed to get channel 1")
		return 0
	}
	p2, ok := pcs.Channels[_ChIDFromRow(r2)]
	if !ok {
		log.Errorln("Failed to get channel 2")
		return 0
	}

	switch {
	case p1.LastMsg < p2.LastMsg:
		return 1
	case p1.LastMsg > p2.LastMsg:
		return -1
	default:
		return 0
	}
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

func (pc *PrivateChannel) ackLatest(m *Message) {
	App.State.MarkRead(pc.ID, m.ID, m.AuthorID != App.Me.ID)
}

func (pc *PrivateChannel) loadMessages() error {
	if pc.Messages == nil {
		m, err := newMessages(pc.ID)
		if err != nil {
			return err
		}
		m.OnInsert = pc.ackLatest
		pc.Messages = m
	}

	if err := pc.Messages.reset(); err != nil {
		return errors.Wrap(err, "Failed to reset private messages")
	}

	return nil
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
