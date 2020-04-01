package message

import (
	"sort"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"github.com/sasha-s/go-deadlock"
)

const scrollMinDelta = 500

var MaxMessageWidth = 750

type Messages struct {
	gtkutils.ExtendedWidget
	channelID moreatomic.Snowflake
	guildID   moreatomic.Snowflake

	c     *ningen.State
	fetch int // max messages

	Main *gtk.Box

	Scroll   *gtk.ScrolledWindow
	Viewport *gtk.Viewport
	atBottom bool

	Messages *gtk.ListBox
	messages []*Message
	guard    deadlock.RWMutex

	resetting    bool
	fetchingMore moreatomic.Bool
	lastFetched  moreatomic.Time

	// Additional components
	Input *Input
}

func NewMessages(s *ningen.State) (*Messages, error) {
	// guildID == 1 is a hack to fix DM.
	m := &Messages{c: s, fetch: s.Store.MaxMessages(), guildID: 1}

	semaphore.IdleMust(func() {
		main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		m.ExtendedWidget = main
		m.Main = main

		// Make the input and typing state:
		m.Input = NewInput(m)

		b, _ := gtk.ListBoxNew()
		m.Messages = b

		v, _ := gtk.ViewportNew(nil, nil)
		m.Viewport = v
		// p, _ := v.GetVAdjustment()
		// m.viewAdj = p

		s, _ := gtk.ScrolledWindowNew(nil, nil)
		m.Scroll = s

		// Main actually contains the scrolling window.
		gtkutils.InjectCSSUnsafe(main, "messagecontainer", "")
		main.Show()
		main.SetHExpand(true)
		main.SetVExpand(true)

		b.SetSelectionMode(gtk.SELECTION_NONE)
		b.SetVExpand(true)
		b.SetHExpand(true)
		b.Show()

		gtkutils.InjectCSSUnsafe(b, "messages", `
			.messages {
				padding-bottom: 4px;
			}
		`)

		s.Connect("edge-reached", m.onEdgeReached)
		s.Connect("edge-overshot", m.onEdgeOvershot)
		s.SetCanFocus(true)
		s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)
		s.SetProperty("propagate-natural-height", true)
		s.SetProperty("min-content-width", 300)
		s.SetProperty("min-content-height", 300)
		s.SetProperty("window-placement", gtk.CORNER_BOTTOM_LEFT)
		s.Show()

		// List should fill:
		// b.SetSizeRequest(MaxMessageWidth, -1)

		// Column contains the list:
		col := handy.ColumnNew()
		col.SetMaximumWidth(MaxMessageWidth)
		col.SetLinearGrowthWidth(10000) // force as wide as possible
		col.Add(b)
		col.Show()

		v.SetCanFocus(false)
		v.SetVAlign(gtk.ALIGN_END)
		v.SetProperty("vscroll-policy", gtkutils.SCROLL_NATURAL)
		v.SetShadowType(gtk.SHADOW_NONE)
		v.Add(col) // add col instead of list
		v.Show()

		// Fractal does this, but Go is superior.
		adj := s.GetVAdjustment()
		adj.Connect("value-changed", m.onScroll)
		v.SetFocusVAdjustment(adj)

		s.Add(v)
		s.Show()

		// Add the message window:
		main.Add(s)

		// Add what's needed afterwards:
		main.PackEnd(m.Input, false, false, 0)

		// On mouse-press, focus:
		s.Connect("button-release-event", func(_ *gtk.ScrolledWindow, ev *gdk.Event) bool {
			if gtkutils.EventIsLeftClick(ev) {
				m.Focus()
			}
			return false
		})

		// On any key-press, focus onto the input box:
		col.Connect("key-press-event", func(_ *glib.Object, ev *gdk.Event) bool {
			m.Focus()
			// Pass the event in
			m.Input.Input.Event(ev)

			// Drain down the event;
			return false
		})
	})

	m.injectHandlers()
	m.injectPopup()
	return m, nil
}

// Focus on the input box
func (m *Messages) Focus() {
	m.Input.Input.GrabFocus()
}

func (m *Messages) GetChannelID() discord.Snowflake {
	return m.channelID.Get()
}

func (m *Messages) GetGuildID() discord.Snowflake {
	return m.guildID.Get()
}

func (m *Messages) GetRecentAuthors(limit int) []discord.Snowflake {
	ids := make([]discord.Snowflake, 0, limit)
	added := make(map[discord.Snowflake]struct{}, limit)

	m.guard.RLock()
	defer m.guard.RUnlock()

	for i := len(m.messages) - 1; i >= 0; i-- {
		message := m.messages[i]

		if _, ok := added[message.AuthorID]; ok {
			continue
		}

		ids = append(ids, message.AuthorID)
		added[message.AuthorID] = struct{}{}
	}

	return ids
}

func (m *Messages) LastFromMe() *Message {
	m.guard.RLock()
	defer m.guard.RUnlock()

	for n := len(m.messages) - 1; n >= 0; n-- {
		if msg := m.messages[n]; msg.AuthorID == m.c.Ready.User.ID {
			return msg
		}
	}
	return nil
}

func (m *Messages) Last() *Message {
	if len(m.messages) == 0 {
		return nil
	}
	return m.messages[len(m.messages)-1]
}

func (m *Messages) LastID() discord.Snowflake {
	if msg := m.Last(); msg != nil {
		return msg.ID
	}
	return 0
}

func (m *Messages) Load(channel discord.Snowflake) error {
	m.guard.Lock()
	defer m.guard.Unlock()

	m.channelID.Set(channel)

	// Mark that we're loading messages.
	m.resetting = true

	// Order: latest is first.
	messages, err := m.c.Messages(channel)
	if err != nil {
		return errors.Wrap(err, "Failed to get messages")
	}

	// Set GuildID and subscribe if it's valid:
	if len(messages) > 0 {
		guild := messages[0].GuildID
		m.guildID.Set(guild)

		if guild.Valid() {
			go m.c.Subscribe(guild, channel, 0)
		}

	} else {
		// If there are no messages, don't bother.
		return nil
	}

	// Sort so that latest is last:
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].ID < messages[j].ID
	})

	// Allocate a new empty slice. This is a trade-off to re-using the old
	// slice to re-use messages.
	m.messages = make([]*Message, 0, m.fetch)

	// WaitGroup for the background goroutines that were spawned:
	// var loads = make([])

	// Iterate from earliest to latest, in a thread-safe function.
	semaphore.IdleMust(func() {
		for i := 0; i < len(messages); i++ {
			message := &messages[i]

			w := newMessageUnsafe(m.c, message)
			w.updateAuthor(m.c, message.GuildID, message.Author)
			m._insert(w)
		}

		// Iterate backwards, from latest to earliest:
		for i := len(m.messages) - 1; i >= 0; i-- {
			m.messages[i].updateExtras(m.c, &messages[i])
		}
	})

	m.resetting = false

	return nil
}

func (m *Messages) lastMessageFrom(author discord.Snowflake) *Message {
	return lastMessageFrom(m.messages, author)
}

func (m *Messages) Cleanup() {
	m.Input.Typing.Stop()

	m.guard.Lock()
	defer m.guard.Unlock()

	semaphore.IdleMust(func() {
		for _, msg := range m.messages {
			// DESTROY!!!!
			// https://stackoverflow.com/questions/2862509/free-object-widget-in-gtk
			m.Messages.Remove(msg)
		}
	})

	// Destroy the slice in Go as well, but the GC will pick it up:
	m.messages = nil
}

func (m *Messages) ScrollToBottom() {
	// Set scroll:
	vAdj := m.Scroll.GetVAdjustment()
	to := vAdj.GetUpper() - vAdj.GetPageSize() - 1
	vAdj.SetValue(to)
}

func (m *Messages) onScroll(adj *gtk.Adjustment) {
	if adj.GetUpper()-adj.GetPageSize() == adj.GetValue() {
		m.atBottom = true
	} else {
		m.atBottom = false
	}
}

// mainly used to mark something as read when scrolled to the bottom
func (m *Messages) onEdgeReached(_ *gtk.ScrolledWindow, pos gtk.PositionType) {
	// only count scroll to bottom
	if pos != gtk.POS_BOTTOM {
		return
	}

	chID := m.GetChannelID()

	r := m.c.FindLastRead(chID)
	if r == nil {
		return
	}

	// Run this in a goroutine to avoid the mutex acquire from locking the UI
	// thread. Since goroutines are cheap, this isn't a huge issue.
	go func() {
		lastID := m.LastID()
		if !lastID.Valid() {
			return
		}

		if r.LastMessageID == lastID {
			return
		}

		// Find the latest message and ack it:
		m.c.MarkRead(chID, lastID)
	}()
}

// mainly used for fetching extra message when scrolled to the top
func (m *Messages) onEdgeOvershot(_ *gtk.ScrolledWindow, pos gtk.PositionType) {
	// only count scroll to top
	if pos != gtk.POS_TOP {
		return
	}

	// Prevent fetching more if we're already fetching.
	if m.fetchingMore.Get() {
		return
	}

	// Prevent fetching if we've just fetched 5 (or less) seconds ago. HasBeen
	// also implicitly updates.
	if !m.lastFetched.HasBeen(2 * time.Second) {
		return
	}

	// Buggy, apparently steals lock.

	go func() {
		m.fetchingMore.Set(true)
		m.guard.Lock()

		defer m.fetchingMore.Set(false)
		defer m.guard.Unlock()

		semaphore.IdleMust(m.Scroll.SetSensitive, false)
		defer semaphore.IdleMust(m.Scroll.SetSensitive, true)

		m.fetchMore()
	}()
}

func (m *Messages) fetchMore() {
	if len(m.messages) < m.fetch {
		return
	}

	// Grab the first ID
	first := m.messages[0].ID

	// Grab the channel and guild dID:
	channelID := m.channelID.Get()
	guildID := m.guildID.Get()

	// Bypass the state cache
	messages, err := m.c.MessagesBefore(channelID, first, uint(m.fetch))
	if err != nil {
		// TODO: error popup
		log.Errorln("Failed to fetch past messages:", err)
		return
	}

	// Sort so that latest is last:
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].ID < messages[j].ID
	})

	// Allocate a new empty slice for past messages. The earliest message
	// appears first.
	oldMsgs := make([]*Message, 0, len(messages))

	// Iterate from earliest to latest, in a thread-safe function.
	semaphore.IdleMust(func() {
		for i := 0; i < len(messages); i++ {
			message := &messages[i]

			// Create a new message without insert.
			w := newMessageUnsafe(m.c, message)
			injectMessage(m, w)
			tryCondense(oldMsgs, w)

			oldMsgs = append(oldMsgs, w)
		}

		// Now we're prepending the message, latest first.
		for i := len(oldMsgs) - 1; i >= 0; i-- {
			w := oldMsgs[i]
			message := &messages[i]

			// Prepend into the box and show the message:
			m.Messages.Prepend(w)
			w.ShowAll()

			// Update the message too, only we use the channel's GuildID instead
			// of the message's, as GuildID isn't populated for API-fetched messages.
			w.updateAuthor(m.c, guildID, message.Author)
			w.updateExtras(m.c, message)
		}
	})

	// Prepend into the slice as well:
	m.messages = append(oldMsgs, m.messages...)
}

func (m *Messages) cleanOldMessages() {
	// Check the scrolling

	if !m.atBottom {
		return
	}

	m.guard.Lock()
	defer m.guard.Unlock()

	// Check the number of messages
	if len(m.messages) <= m.fetch {
		return
	}

	// Get the messages needed to be cleaned
	cleanLen := len(m.messages) - m.fetch
	cleaned := m.messages[:cleanLen]

	// Clean the slice
	m.messages = m.messages[cleanLen:]

	// Destroy the messages:
	semaphore.IdleMust(func() {
		for i, r := range cleaned {
			m.Messages.Remove(r.ListBoxRow)
			cleaned[i] = nil
		}
	})
}

func (m *Messages) Upsert(message *discord.Message) {
	// Clean up old messages (thread-safe):
	defer m.cleanOldMessages()

	// Are we sure this is not our message?
	if m.Update(message) {
		return
	}

	var w *Message
	semaphore.IdleMust(func() {
		w = newMessageUnsafe(m.c, message)
	})

	// Avoid the mutex from locking the UI thread when things are busy.
	m.guard.Lock()
	semaphore.IdleMust(m._insert, w)
	m.guard.Unlock()

	semaphore.IdleMust(func() {
		w.updateAuthor(m.c, message.GuildID, message.Author)
		w.updateExtras(m.c, message)
	})
}

// not thread-safe
func (m *Messages) _insert(w *Message) {
	// Bind Message's fields to Messages'
	injectMessage(m, w)

	// Try and see if the message should be condensed
	tryCondense(m.messages, w)

	// This adds the message into the list, not call the above Insert().
	m.Messages.Insert(w, -1)
	m.messages = append(m.messages, w)

	w.ShowAll()
}

func (m *Messages) Update(update *discord.Message) bool {
	var target *Message

	m.guard.RLock()

	for _, message := range m.messages {
		if false ||
			(message.ID.Valid() && message.ID == update.ID) ||
			(message.Nonce != "" && message.Nonce == update.Nonce) {

			target = message
			break
		}
	}

	m.guard.RUnlock()

	if target == nil {
		return false
	}

	target.ID = update.ID

	// Clear the nonce, if any:
	semaphore.IdleMust(func() {
		if !target.getAvailableUnsafe() {
			target.setAvailableUnsafe(true)
		}
		if update.Content != "" {
			target.UpdateContentUnsafe(m.c, update)
		}
		target.updateExtras(m.c, update)
	})

	return true
}

func (m *Messages) updateMessageAuthor(ns ...discord.Member) {
	guildID := m.guildID.Get()

	semaphore.IdleMust(func() {
		for _, n := range ns {
			for _, message := range m.messages {
				if message.AuthorID != n.User.ID {
					continue
				}
				message.updateMember(m.c, guildID, n)
			}
		}
	})
}

func (m *Messages) Delete(ids ...discord.Snowflake) {
	m.guard.Lock()
	defer m.guard.Unlock()

	m.delete(ids...)
}

func (m *Messages) delete(ids ...discord.Snowflake) {
	for _, id := range ids {
		for i, message := range m.messages {
			if message.ID != id {
				continue
			}

			oldMessage := m.messages[i]

			m.messages = append(m.messages[:i], m.messages[i+1:]...)
			semaphore.IdleMust(m.Messages.Remove, oldMessage)

			// Exit if len is 0
			if len(m.messages) == 0 {
				return
			}

			// Check if the last message (relative to i) is the author's:
			if i > 0 && m.messages[i-1].AuthorID == oldMessage.AuthorID {
				// Then we continue, since we don't need to uncollapse.
				break
			}

			// Check if next message is author's:
			if i < len(m.messages) && m.messages[i].AuthorID == oldMessage.AuthorID {
				// Then uncollapse next message:
				semaphore.IdleMust(m.messages[i].SetCondensedUnsafe, false)
			}

			break
		}
	}
}

func (m *Messages) deleteNonce(nonce string) bool {
	m.guard.Lock()
	defer m.guard.Unlock()

	for i, message := range m.messages {
		if message.Nonce != nonce {
			continue
		}

		m.messages = append(m.messages[:i], m.messages[i+1:]...)
		semaphore.IdleMust(m.Messages.Remove, message)
		return true
	}

	return false
}
