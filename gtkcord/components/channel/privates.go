package channel

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sasha-s/go-deadlock"
)

type PrivateChannels struct {
	*gtk.Box

	List   *gtk.ListBox
	Scroll *gtk.ScrolledWindow

	Search *gtk.Entry
	search string

	// Channels map[discord.Snowflake]*PrivateChannel
	Channels map[string]*PrivateChannel

	busy  deadlock.RWMutex
	state *ningen.State

	OnSelect func(pm *PrivateChannel)
}

// thread-safe
func NewPrivateChannels(s *ningen.State, onSelect func(pm *PrivateChannel)) (pcs *PrivateChannels) {
	semaphore.IdleMust(func() {
		l, _ := gtk.ListBoxNew()
		l.Show()
		gtkutils.InjectCSSUnsafe(l, "dmchannels", "")

		cs, _ := gtk.ScrolledWindowNew(nil, nil)
		cs.Show()
		cs.SetVExpand(true)
		cs.Add(l)

		e, _ := gtk.EntryNew()
		e.Show()
		e.SetPlaceholderText("Find conversation...")

		b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		b.Show()
		b.Add(e)
		b.Add(cs)

		pcs = &PrivateChannels{
			Box:    b,
			List:   l,
			Scroll: cs,
			Search: e,

			state:    s,
			OnSelect: onSelect,
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
			if len(pcs.Channels) == 0 || pcs.OnSelect == nil || r == nil {
				return
			}

			rw, ok := pcs.Channels[_ChIDFromRow(r)]
			if !ok {
				log.Errorln("Failed to find channel")
				return
			}

			pcs.OnSelect(rw)
		})
	})

	s.Read.OnChange(pcs.TraverseReadState)
	return
}

func (pcs *PrivateChannels) Cleanup() {
	pcs.busy.Lock()
	defer pcs.busy.Unlock()

	if pcs.Channels != nil {
		for _, ch := range pcs.Channels {
			pcs.List.Remove(ch)
		}

		pcs.Channels = nil
	}
}

// thread-safe
func (pcs *PrivateChannels) LoadChannels() error {
	pcs.busy.Lock()
	defer pcs.busy.Unlock()
	// defer at the end of the bottom goroutine.

	channels, err := pcs.state.PrivateChannels()
	if err != nil {
		return err
	}

	pcs.Channels = make(map[string]*PrivateChannel, len(channels))

	for _, channel := range channels {
		w := newPrivateChannel(channel)

		if channel.Type == discord.DirectMessage && len(channel.DMRecipients) == 1 {
			user := channel.DMRecipients[0]
			w.updateAvatar(user.AvatarURL())

			if p, _ := pcs.state.Presence(0, user.ID); p != nil {
				var game = p.Game
				if game == nil && len(p.Activities) > 0 {
					game = &p.Activities[0]
				}

				w.updateStatus(p.Status)
				w.updateActivity(game)
			}

		} else if channel.Icon != "" {
			w.updateAvatar(channel.IconURL())
		}

		pcs.Channels[channel.ID.String()] = w
		pcs.List.Insert(w, -1)
	}

	go func() {
		pcs.busy.Lock()
		defer pcs.busy.Unlock()

		for _, channel := range channels {
			rs := pcs.state.Read.FindLast(channel.ID)
			if rs == nil {
				continue
			}

			// Snowflakes have timestamps, which allow us to do this:
			if channel.LastMessageID.Time().After(rs.LastMessageID.Time()) {
				ch := pcs.Channels[channel.ID.String()]
				semaphore.Async(ch.setUnread, true)
			}
		}
	}()

	return nil
}

func (pcs *PrivateChannels) Selected() *PrivateChannel {
	pcs.busy.RLock()
	defer pcs.busy.RUnlock()

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

func (pcs *PrivateChannels) filter(r *gtk.ListBoxRow, _ ...interface{}) bool {
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

func (pcs *PrivateChannels) TraverseReadState(rs gateway.ReadState, unread bool) {
	// Read lock is used, as the size of the slice isn't directly modified.
	pcs.busy.RLock()
	defer pcs.busy.RUnlock()

	if len(pcs.Channels) == 0 {
		return
	}

	pc, ok := pcs.Channels[rs.ChannelID.String()]
	if !ok {
		return
	}

	// Prepend/move to top.
	semaphore.IdleMust(func() {
		pcs.List.Remove(pc)
		pcs.List.Prepend(pc)
	})

	pc.setUnread(unread)
}

func (pcs *PrivateChannels) FindByID(id discord.Snowflake) *PrivateChannel {
	ch, _ := pcs.Channels[id.String()]
	return ch
}

// func (pcs *PrivateChannels) updatePresence(p discord.Presence) {
// 	for _, ch := range pcs.Channels {
// 		if ch.Recp == p.User.ID && !ch.Group {
// 			ch.UpdateStatus(p.Status)
// 			ch.UpdateActivity(p.Game)
// 			break
// 		}
// 	}
// }

// func (pcs *PrivateChannels) setUnread(unread bool) {
// 	if !unread {
// 		for _, ch := range pcs.Channels {
// 			if ch.stateClass == "pinged" {
// 				unread = true
// 				break
// 			}
// 		}
// 	}

// 	if unread {
// 		pcs.setButtonClass("pinged")
// 	} else {
// 		pcs.setButtonClass("")
// 	}
// }
