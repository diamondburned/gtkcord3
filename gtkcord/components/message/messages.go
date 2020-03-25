package message

import (
	"sort"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/typing"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/diamondburned/gtkcord3/internal/mutexlog"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const scrollMinDelta = 500

var MaxMessageWidth = 800

type Messages struct {
	gtkutils.ExtendedWidget
	ChannelID discord.Snowflake
	GuildID   discord.Snowflake

	c     *ningen.State
	fetch int // max messages

	Main *gtk.Box

	Scroll   *gtk.ScrolledWindow
	Viewport *gtk.Viewport
	atBottom bool

	Messages *gtk.ListBox
	messages []*Message
	guard    mutexlog.Mutex

	resetting    bool
	fetchingMore moreatomic.Bool
	lastFetched  moreatomic.Time

	// Additional components
	Input  *Input
	Typing *typing.State

	acked bool
}

func NewMessages(s *ningen.State) (*Messages, error) {
	m := &Messages{c: s, fetch: s.Store.MaxMessages()}

	semaphore.IdleMust(func() {
		main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		m.Main = main
		m.ExtendedWidget = main

		// Make the input and typing state:
		m.Input = NewInput(m)
		m.Typing = typing.NewState(s.State)

		// For wrappping around listbox
		c := handy.ColumnNew()

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
		s.SetProperty("min-content-width", 300)
		s.SetProperty("min-content-height", 300)
		s.SetProperty("window-placement", gtk.CORNER_BOTTOM_LEFT)
		s.Show()

		// Column actually contains the list:
		c.SetMaximumWidth(MaxMessageWidth)
		c.SetLinearGrowthWidth(MaxMessageWidth * 100 / 75) // 800 -> 600
		c.SetHExpand(true)
		c.SetVExpand(true)
		c.Add(b)
		c.Show()

		// List should fill:
		// b.SetSizeRequest(MaxMessageWidth, -1)

		// Causes resize bugs:
		v.SetCanFocus(false)
		v.SetVAlign(gtk.ALIGN_END)
		v.SetProperty("vscroll-policy", gtkutils.SCROLL_NATURAL)
		v.SetShadowType(gtk.SHADOW_NONE)
		v.Add(c) // add col instead of list
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

		// Hijack Input's box and add the typing indicator:
		m.Input.Main.Add(m.Typing)
		m.Typing.ShowAll()

		// On any key-press, focus onto the input box:
		m.Main.Connect("key-press-event", func(_ *gtk.Box, ev *gdk.Event) bool {
			m.Focus()
			// Pass the event in
			m.Input.Input.Event(ev)

			// Stop the event from reaching the List:
			return true
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
	m.guard.Lock()
	defer m.guard.Unlock()

	return m.ChannelID
}

func (m *Messages) GetGuildID() discord.Snowflake {
	m.guard.Lock()
	defer m.guard.Unlock()

	return m.GuildID
}

func (m *Messages) GetRecentAuthors(limit int) []discord.Snowflake {
	ids := make([]discord.Snowflake, 0, limit)
	added := make(map[discord.Snowflake]struct{}, limit)

	m.guard.Lock()
	defer m.guard.Unlock()

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
	m.guard.Lock()
	defer m.guard.Unlock()

	for n := len(m.messages) - 1; n >= 0; n-- {
		if msg := m.messages[n]; msg.AuthorID == m.c.Ready.User.ID {
			return msg
		}
	}
	return nil
}

func (m *Messages) Last() *Message {
	// yolo, this causes a freeze bug.

	m.guard.Lock()
	defer m.guard.Unlock()

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

	m.ChannelID = channel

	// Mark that we're loading messages.
	m.resetting = true

	// Order: latest is first.
	messages, err := m.c.Messages(m.ChannelID)
	if err != nil {
		return errors.Wrap(err, "Failed to get messages")
	}

	// Set GuildID and subscribe if it's valid:
	if len(messages) > 0 {
		m.GuildID = messages[0].GuildID
		if m.GuildID.Valid() {
			go m.c.Subscribe(m.GuildID, m.ChannelID, 0)
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
	})

	// Mark for ack, check onEdgeReached
	m.acked = false

	// Iterate backwards, from latest to earliest.
	for i := len(m.messages) - 1; i >= 0; i-- {
		go m.messages[i].UpdateExtras(m.c, &messages[i])
	}

	m.resetting = false

	return nil
}

func (m *Messages) ShouldCondense(msg *Message) bool {
	return shouldCondense(m.messages, msg)
}

func (m *Messages) lastMessageFrom(author discord.Snowflake) *Message {
	return lastMessageFrom(m.messages, author)
}

func (m *Messages) Cleanup() {
	m.guard.Lock()
	defer m.guard.Unlock()

	log.Infoln("Destroying messages from old channel.")
	m.Typing.Stop()

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
	log.Println("scrolling to:", to)
	vAdj.SetValue(to)
}

// func (m *Messages) onSizeAlloc() {
// 	log.Println("Child notify")

// 	adj, _ := m.Viewport.GetVAdjustment()

// 	max := adj.GetUpper()

// 	// if max := int64(max); max == m.lastHeight {
// 	// 	return
// 	// } else {
// 	// 	m.lastHeight = max
// 	// }

// 	cur := adj.GetValue() + adj.GetPageSize()

// 	delta := int32(max - cur)
// 	atomic.StoreInt32(&m.scrollDelta, delta)

// 	// If the scroll is not close to the bottom and we're not loading messages:
// 	if delta > scrollMinDelta {
// 		// Then we don't scroll.
// 		// log.Println("Not scrolling. Loading:", loading)
// 		return
// 	}

// 	adj.SetValue(max)
// }

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

	if m.acked {
		return
	}
	m.acked = true

	go func() {
		m.guard.Lock()
		defer m.guard.Unlock()

		if len(m.messages) == 0 {
			return
		}

		msID := m.messages[len(m.messages)-1].ID
		chID := m.ChannelID

		// Find the latest message and ack it:
		go m.c.MarkRead(chID, msID)
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
	if !m.lastFetched.HasBeen(5 * time.Second) {
		return
	}

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

	// Bypass the state cache
	messages, err := m.c.MessagesBefore(m.ChannelID, first, uint(m.fetch))
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
			w.updateAuthor(m.c, m.GuildID, message.Author)
			go w.UpdateExtras(m.c, message)
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

func (m *Messages) Insert(message *discord.Message) {
	m.guard.Lock()
	defer m.guard.Unlock()

	m.insert(message)
}

func (m *Messages) insert(message *discord.Message) {
	// Clean up old messages (thread-safe):
	defer m.cleanOldMessages()

	// Are we sure this is not our message?
	if m.update(message) {
		return
	}

	// Mark for ack, check onEdgeReached
	m.acked = false

	var w *Message
	semaphore.IdleMust(func() {
		w = newMessageUnsafe(m.c, message)
		m._insert(w)
		w.updateAuthor(m.c, message.GuildID, message.Author)
	})

	go w.UpdateExtras(m.c, message)
}

// not thread safe
func (m *Messages) _insert(w *Message) {
	// Bind Message's fields to Messages'
	injectMessage(m, w)

	// Try and see if the message should be condensed
	tryCondense(m.messages, w)

	m.Messages.Insert(w, -1)
	m.messages = append(m.messages, w)

	w.ShowAll()
}

func (m *Messages) Update(update *discord.Message) bool {
	m.guard.Lock()
	defer m.guard.Unlock()

	return m.update(update)
}

func (m *Messages) update(update *discord.Message) bool {
	var target *Message

	for _, message := range m.messages {
		if false ||
			(message.ID.Valid() && message.ID == update.ID) ||
			(message.Nonce != "" && message.Nonce == update.Nonce) {

			target = message
			break
		}
	}

	if target == nil {
		return false
	}

	// Clear the nonce, if any:
	semaphore.IdleMust(func() {
		if !target.getAvailableUnsafe() {
			target.setAvailableUnsafe(true)
		}
	})

	target.ID = update.ID
	// target.Nonce = ""

	if update.Content != "" {
		target.UpdateContent(m.c, update)
	}
	go func() {
		target.UpdateExtras(m.c, update)
	}()

	return true
}

func (m *Messages) updateMessageAuthor(ns ...discord.Member) {
	for _, n := range ns {
		for _, message := range m.messages {
			if message.AuthorID != n.User.ID {
				continue
			}
			message.UpdateMember(m.c, m.GuildID, n)
		}
	}
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

			m.messages = append(m.messages[:i], m.messages[i+1:]...)
			semaphore.IdleMust(m.Messages.Remove, message)

			// Exit if len is 0
			if len(m.messages) == 0 {
				return
			}

			// Check if the last message (relative to i) is the author's:
			if i > 0 && m.messages[i-1].AuthorID == message.AuthorID {
				// Then we continue, since we don't need to uncollapse.
				break
			}

			// Check if next message is author's:
			if i < len(m.messages) && m.messages[i].AuthorID == message.AuthorID {
				// Then uncollapse next message:
				semaphore.IdleMust(m.messages[i].SetCondensedUnsafe, false)
			}

			break
		}
	}

	return
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
