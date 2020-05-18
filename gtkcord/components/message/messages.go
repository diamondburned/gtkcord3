package message

import (
	"sort"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const scrollMinDelta = 500

type Messages struct {
	Opts
	gtkutils.ExtendedWidget

	channelID moreatomic.Snowflake
	guildID   moreatomic.Snowflake

	c     *ningen.State
	fetch int // max messages

	Main   *gtk.Box
	Column *handy.Column

	Scroll   *gtk.ScrolledWindow
	Viewport *gtk.Viewport
	atBottom moreatomic.Bool

	Messages *gtk.ListBox
	messages []*Message
	// guard    deadlock.RWMutex

	fetching    moreatomic.Bool
	lastFetched moreatomic.Time

	// Additional components
	Input *Input
}

type Opts struct {
	// Whether or not the sent messages should be "obfuscated" with zero-width
	// space characters, which avoids telemetry somewhat.
	InputZeroWidth bool // true

	// Whether or not gtkcord should send typing events to the Discord server
	// and announce it.
	InputOnTyping bool // true

	MessageWidth int
}

func NewMessages(s *ningen.State, opts Opts) (*Messages, error) {
	// guildID == 1 is a hack to fix DM.
	m := &Messages{Opts: opts, c: s, fetch: s.Store.MaxMessages(), guildID: 1}

	semaphore.IdleMust(func() {
		main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		m.ExtendedWidget = main
		m.Main = main

		col := handy.ColumnNew()
		m.Column = col

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

		b.SetSelectionMode(gtk.SELECTION_NONE)
		b.SetVExpand(true)
		b.SetHExpand(true)
		b.Show()

		// List should fill:
		// b.SetSizeRequest(variables.MaxMessageWidth, -1)

		// Column contains the list:
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

		s.Add(v)
		s.Show()

		// Add the message window:
		main.Add(s)

		// Add what's needed afterwards:
		main.PackEnd(m.Input, false, false, 0)

		// Set the proper scrolls
		b.SetFocusHAdjustment(s.GetHAdjustment())
		b.SetFocusVAdjustment(s.GetVAdjustment())

		// Scroll to the bottom if we have more things.
		b.Connect("size-allocate", func() {
			if m.atBottom.Get() {
				m.ScrollToBottom()
			}
		})

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

		// Set maximum widths
		m.SetWidth(opts.MessageWidth)
	})

	m.injectHandlers()
	m.injectPopup()
	return m, nil
}

func (m *Messages) SetWidth(width int) {
	variables.MaxMessageWidth = width
	m.Opts.MessageWidth = width
	m.Column.SetMaximumWidth(width)
	m.Input.Column.SetMaximumWidth(width)
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

func (m *Messages) GetRecentAuthorsUnsafe(limit int) []discord.Snowflake {
	ids := make([]discord.Snowflake, 0, limit)
	added := make(map[discord.Snowflake]struct{}, limit)

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

// unsafe
func (m *Messages) LastFromMeUnsafe() *Message {
	for n := len(m.messages) - 1; n >= 0; n-- {
		if msg := m.messages[n]; msg.AuthorID == m.c.Ready.User.ID {
			return msg
		}
	}
	return nil
}

// Load is thread-safe. Done will be called in the main thread.
func (m *Messages) Load(channel discord.Snowflake, done func(error)) {
	m.channelID.Set(channel)

	// Order: latest is first.
	messages, err := m.c.Messages(channel)
	if err != nil {
		semaphore.Async(func() {
			done(errors.Wrap(err, "Failed to get messages"))
		})
		return
	}

	// If there are no messages, don't bother.
	if len(messages) == 0 {
		semaphore.Async(func() {
			// Pretend we're done.
			done(nil)
		})
		return
	}

	// Set GuildID and subscribe if it's valid:
	var guildID = messages[0].GuildID

	if guildID.Valid() {
		// Ensure there's a member list.
		if err := m.c.Members.GetMemberList(guildID, channel, nil); err != nil {
			m.c.Members.RequestMemberList(guildID, channel, 0)
		}
	}

	// Sort so that latest is last:
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].ID < messages[j].ID
	})

	semaphore.Async(func() {
		// Set the guild ID to the state struct.
		m.guildID.Set(guildID)

		// Allocate a new empty slice. This is a trade-off to re-using the old
		// slice to re-use messages.
		m.messages = make([]*Message, 0, m.fetch)

		// WaitGroup for the background goroutines that were spawned:
		// var loads = make([])

		// Iterate from earliest to latest, in a thread-safe function.
		for i := 0; i < len(messages); i++ {
			message := &messages[i]

			w := newMessageUnsafe(m.c, message)
			w.updateAuthor(m.c, message.GuildID, message.Author)
			m._insert(w)
		}

		// Request that we should always scroll to bottom.
		m.atBottom.Set(true)
		// Call the done callback.
		done(nil)
	})

	// Give some breathing room.

	semaphore.Async(func() {
		// Iterate backwards, from latest to earliest:
		for i := len(m.messages) - 1; i >= 0; i-- {
			m.messages[i].updateExtras(m.c, &messages[i])
		}
	})
}

func (m *Messages) lastMessageFrom(author discord.Snowflake) *Message {
	return lastMessageFrom(m.messages, author)
}

func (m *Messages) Cleanup() {
	m.Input.Typing.Stop()

	for _, msg := range m.messages {
		// DESTROY!!!!
		// https://stackoverflow.com/questions/2862509/free-object-widget-in-gtk
		m.Messages.Remove(msg)
	}

	// Destroy the slice in Go as well, but the GC will pick it up:
	m.messages = nil
}

func (m *Messages) ScrollToBottom() {
	vAdj := m.Scroll.GetVAdjustment()
	vAdj.SetValue(vAdj.GetUpper())
}

func (m *Messages) onScroll(adj *gtk.Adjustment) {
	if adj.GetUpper()-adj.GetPageSize() <= adj.GetValue() {
		m.atBottom.Set(true)
	} else {
		m.atBottom.Set(false)
	}
}

// mainly used to mark something as read when scrolled to the bottom
func (m *Messages) onEdgeReached(_ *gtk.ScrolledWindow, pos gtk.PositionType) {
	// only count scroll to bottom
	if pos != gtk.POS_BOTTOM {
		return
	}
	// If there are no messages.
	if len(m.messages) == 0 {
		return
	}

	chID := m.GetChannelID()
	lastID := m.messages[len(m.messages)-1].ID

	// Find the latest message and ack it. Since MarkRead calls the onChanges
	// functions which would be using semaphore.IdleMust, we need to spawn this
	// in a goroutine.
	r := m.c.Read.FindLast(chID)
	if r == nil || r.LastMessageID == lastID {
		return
	}

	m.c.Read.MarkRead(chID, lastID)
}

// mainly used for fetching extra message when scrolled to the top
func (m *Messages) onEdgeOvershot(_ *gtk.ScrolledWindow, pos gtk.PositionType) {
	// Do we even have messages?
	if len(m.messages) < m.fetch {
		return
	}

	// only count scroll to top
	if pos != gtk.POS_TOP {
		return
	}

	// Prevent fetching if we've just fetched 2 (or less) seconds ago. HasBeen
	// also implicitly updates.
	if !m.lastFetched.HasBeen(2*time.Second) || m.fetching.Get() {
		return
	}

	m.Scroll.SetSensitive(false)

	// Grab the first ID
	go m.fetchMore(m.messages[0].ID)
}

func (m *Messages) fetchMore(first discord.Snowflake) {
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

	// I'm not sure if this would make Go put everything on the heap. Maybe it
	// already does.
	semaphore.Async(func() {
		// Iterate from earliest to latest, in a thread-safe function.
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

		// Prepend into the slice as well:
		m.messages = append(oldMsgs, m.messages...)

		// Reactivate the boxes.
		m.Scroll.SetSensitive(true)
	})
}

func (m *Messages) cleanOldMessages() {
	// Check the scrolling

	if !m.atBottom.Get() {
		return
	}

	// Check the number of messages
	if len(m.messages) <= m.fetch {
		return
	}

	// Get the messages needed to be cleaned
	cleanLen := len(m.messages) - m.fetch

	// Destroy the messages before reslicing
	// Iterate from 0 to the oldest message to be kept:
	for _, r := range m.messages[:cleanLen] {
		m.Messages.Remove(r)
	}

	// Finally, clean the slice
	m.messages = append(m.messages[:0], m.messages[cleanLen:]...)

	// Apparently, Go's obscure slicihg behavior allows slicing to capacity, not
	// length.
	var excess = m.messages[len(m.messages) : len(m.messages)+cleanLen]
	// Start setting them to nil for the GC to collect:
	for i := range excess {
		excess[i] = nil
	}
}

func (m *Messages) UpsertUnsafe(message *discord.Message) {
	// Clean up old messages (thread-safe):
	defer m.cleanOldMessages()

	// Are we sure this is not our message?
	if m.UpdateUnsafe(message) {
		return
	}

	w := newMessageUnsafe(m.c, message)
	m._insert(w)
	w.updateAuthor(m.c, message.GuildID, message.Author)
	w.updateExtras(m.c, message)
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

func (m *Messages) UpdateUnsafe(update *discord.Message) bool {
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

	target.ID = update.ID

	// Clear the nonce, if any:
	if !target.getAvailableUnsafe() {
		target.setAvailableUnsafe(true)
	}

	if update.Content != "" {
		target.UpdateContentUnsafe(m.c, update)
	}

	target.updateExtras(m.c, update)

	return true
}

func (m *Messages) deleteUnsafe(ids ...discord.Snowflake) {
	for _, id := range ids {
		for i, message := range m.messages {
			if message.ID != id {
				continue
			}

			oldMessage := m.messages[i]

			m.messages = append(m.messages[:i], m.messages[i+1:]...)
			m.Messages.Remove(oldMessage)

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
				m.messages[i].SetCondensedUnsafe(false)
			}

			break
		}
	}
}

func (m *Messages) deleteNonceUnsafe(nonce string) bool {
	for i, message := range m.messages {
		if message.Nonce != nonce {
			continue
		}

		m.messages = append(m.messages[:i], m.messages[i+1:]...)
		m.Messages.Remove(message)
		return true
	}

	return false
}
