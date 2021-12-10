package channel

import (
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/loadstatus"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
	"github.com/diamondburned/ningen/v2/states/read"
)

type PrivateChannels struct {
	*loadstatus.Page

	Main   *gtk.Box
	List   *gtk.ListBox
	Scroll *gtk.ScrolledWindow

	Search *gtk.Entry
	search string

	Channels map[discord.ChannelID]*PrivateChannel

	state *ningen.State

	OnSelect func(pm *PrivateChannel)

	lastSelected discord.ChannelID
	mustRefresh  bool
}

func NewPrivateChannels(s *ningen.State, onSelect func(pm *PrivateChannel)) (pcs *PrivateChannels) {
	l := gtk.NewListBox()
	gtkutils.InjectCSS(l, "dmchannels", "")

	cs := gtk.NewScrolledWindow(nil, nil)
	cs.SetVExpand(true)
	cs.Add(l)

	e := gtk.NewEntry()
	e.SetPlaceholderText("Find conversation...")

	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.Add(e)
	b.Add(cs)
	b.ShowAll()

	page := loadstatus.NewPage()
	page.SetChild(b)

	pcs = &PrivateChannels{
		Page:   page,
		Main:   b,
		List:   l,
		Scroll: cs,
		Search: e,

		state:    s,
		OnSelect: onSelect,
	}

	e.Connect("changed", func() {
		pcs.search = strings.ToLower(e.Text())
		pcs.List.InvalidateFilter()
	})

	l.SetFilterFunc(pcs.filter)
	l.SetSortFunc(pcs.sort)

	l.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		if len(pcs.Channels) == 0 || pcs.OnSelect == nil || r == nil {
			return
		}

		rw, ok := pcs.Channels[chIDFromRow(r)]
		if !ok {
			log.Errorln("Failed to find channel")
			return
		}

		pcs.lastSelected = rw.ID
		pcs.OnSelect(rw)
	})

	s.ReadState.OnUpdate(func(rs *read.UpdateEvent) {
		glib.IdleAdd(func() { pcs.TraverseReadState(rs) })
	})

	return
}

func (pcs *PrivateChannels) Cleanup() {
	for _, ch := range pcs.Channels {
		pcs.List.Remove(ch)
	}
	pcs.Channels = nil
}

func (pcs *PrivateChannels) Load() {
	pcs.Cleanup()
	pcs.SetLoading()

	go func() {
		channels, err := pcs.state.State.PrivateChannels()
		if err != nil {
			glib.IdleAdd(func() { pcs.SetError("Error", err) })
			return
		}

		glib.IdleAdd(func() {
			pcs.SetDone()
			pcs.Channels = make(map[discord.ChannelID]*PrivateChannel, len(channels))

			for _, channel := range channels {
				w := newPrivateChannel(channel)

				if channel.Type == discord.DirectMessage && len(channel.DMRecipients) == 1 {
					user := channel.DMRecipients[0]
					w.updateAvatar(user.AvatarURL())

					if p, _ := pcs.state.Presence(0, user.ID); p != nil {
						w.updateStatus(p.Status)
						if len(p.Activities) > 0 {
							w.updateActivity(&p.Activities[0])
						} else {
							w.updateActivity(nil)
						}
					}
				} else if channel.Icon != "" {
					w.updateAvatar(channel.IconURL())
				}

				pcs.Channels[channel.ID] = w
				pcs.List.Insert(w, -1)
			}

			if pcs.lastSelected.IsValid() {
				if ch, ok := pcs.Channels[pcs.lastSelected]; ok {
					pcs.List.SelectRow(ch.ListBoxRow)
					ch.Activate()
				}
			}

			pcs.List.InvalidateSort()
		})

		ScanUnreadDMs(pcs.state, channels, func(ch *discord.Channel) {
			glib.IdleAdd(func() {
				ch := pcs.Channels[ch.ID]
				ch.setUnread(true)
			})
		})

		for _, channel := range channels {
			if pcs.state.MutedState.Channel(channel.ID) {
				continue
			}

			rs := pcs.state.ReadState.FindLast(channel.ID)
			if rs == nil {
				continue
			}

			// Snowflakes have timestamps, which allow us to do this:
			if channel.LastMessageID.Time().After(rs.LastMessageID.Time()) {
				chID := channel.ID
				glib.IdleAdd(func() {
					ch := pcs.Channels[chID]
					ch.setUnread(true)
				})
			}
		}
	}()
}

func ScanUnreadDMs(n *ningen.State, channels []discord.Channel, fn func(ch *discord.Channel)) {
	for i, channel := range channels {
		if n.MutedState.Channel(channel.ID) {
			continue
		}

		rs := n.ReadState.FindLast(channel.ID)
		if rs == nil {
			continue
		}

		// Snowflakes have timestamps, which allow us to do this:
		if channel.LastMessageID.Time().After(rs.LastMessageID.Time()) {
			fn(&channels[i])
		}
	}
}

func (pcs *PrivateChannels) Selected() *PrivateChannel {
	if len(pcs.Channels) == 0 {
		return nil
	}

	r := pcs.List.SelectedRow()
	if r == nil {
		r = pcs.List.RowAtIndex(0)
		pcs.List.SelectRow(r)
	}

	rw, ok := pcs.Channels[chIDFromRow(r)]
	if !ok || rw == nil {
		log.Errorln("failed to find channel row")
	}
	return rw
}

func (pcs *PrivateChannels) sort(r1, r2 *gtk.ListBoxRow) int { // -1 == less == r1 first
	int1, _ := strconv.ParseInt(r1.Name(), 10, 64)
	int2, _ := strconv.ParseInt(r2.Name(), 10, 64)

	chRow1 := pcs.Channels[discord.ChannelID(int1)]
	chRow2 := pcs.Channels[discord.ChannelID(int2)]
	if v := putLast(chRow1 == nil, chRow2 == nil); v != 0 {
		return v
	}

	ch1, _ := pcs.state.Cabinet.Channel(chRow1.ID)
	ch2, _ := pcs.state.Cabinet.Channel(chRow2.ID)
	if v := putLast(ch1 == nil, ch2 == nil); v != 0 {
		return v
	}

	if v := putLast(!ch1.LastMessageID.IsValid(), !ch2.LastMessageID.IsValid()); v != 0 {
		return v
	}

	if ch1.LastMessageID > ch2.LastMessageID {
		// ch1 is older, put first.
		return -1
	}
	if ch1.LastMessageID == ch2.LastMessageID {
		// equal
		return 0
	}
	return 1 // newer
}

func putLast(b1, b2 bool) int {
	if b1 {
		return 1
	}
	if b2 {
		return -1
	}
	return 0
}

func (pcs *PrivateChannels) filter(r *gtk.ListBoxRow) bool {
	if pcs.search == "" {
		return true
	}

	pc, ok := pcs.Channels[chIDFromRow(r)]
	if !ok {
		log.Errorln("Failed to get channel for filter")
		return false
	}

	return strings.Contains(strings.ToLower(pc.Name), pcs.search)
}

func (pcs *PrivateChannels) TraverseReadState(rs *read.UpdateEvent) {
	pc, ok := pcs.Channels[rs.ChannelID]
	if !ok {
		return
	}

	pcs.List.InvalidateSort()

	pc.setUnread(rs.Unread)
}

func (pcs *PrivateChannels) FindByID(id discord.ChannelID) *PrivateChannel {
	return pcs.Channels[id]
}
