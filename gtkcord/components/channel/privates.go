package channels

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

const (
	DMAvatarSize = 38
	IconPadding  = 8
)

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

	OnSelect func(pm *PrivateChannel)
}

// thread-safe
func NewPrivateChannels() (pcs *PrivateChannels) {
	semaphore.IdleMust(func() {
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
			if len(pcs.Channels) == 0 || pcs.OnSelect == nil {
				return
			}

			rw, ok := pcs.Channels[_ChIDFromRow(r)]
			if !ok {
				log.Errorln("Failed to find channel")
				return
			}

			go pcs.OnSelect(rw)
		})
	})

	icon := icons.GetIcon("system-users-symbolic", DMAvatarSize/3*2)
	semaphore.IdleMust(func() {
		r, _ := gtk.ListBoxRowNew()
		r.SetSizeRequest(DMAvatarSize+IconPadding*2, DMAvatarSize+IconPadding*2)
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

	return pcs
}

// thread-safe
func (pcs *PrivateChannels) LoadChannels(channels []discord.Channel) {
	// fuck diffing.
	if len(pcs.Channels) > 0 {
		semaphore.IdleMust(func() {
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

	semaphore.Async(func() {
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

func (pcs *PrivateChannels) setUnread(unread bool) {
	if !unread {
		for _, ch := range pcs.Channels {
			if ch.stateClass == "pinged" {
				unread = true
				break
			}
		}
	}

	if unread {
		pcs.setButtonClass("pinged")
	} else {
		pcs.setButtonClass("")
	}
}
