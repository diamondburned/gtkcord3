package message

import (
	"sort"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/loadstatus"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

const scrollMinDelta = 500

type Messages struct {
	*loadstatus.Page
	Opts

	channelID discord.ChannelID
	guildID   discord.GuildID

	c     *ningen.State
	fetch int // max messages

	Main   *gtk.Box
	Column *handy.Clamp

	ScrolledBox *gtk.Box
	LoadMore    *gtk.Button

	Messages *gtk.ListBox
	messages []*Message

	// Additional components
	Input *Input

	Scroll   *gtk.ScrolledWindow
	Viewport *gtk.Viewport
	bottomed bool
}

type Opts struct {
	MessageWidth int
	// InputZeroWidth sets whether or not the sent messages should be
	// "obfuscated" with zero-width space characters, which avoids telemetry
	// somewhat.
	InputZeroWidth bool // true
	// InputOnTyping sets whether or not gtkcord3 should send typing events to
	// the Discord server and announce it.
	InputOnTyping bool // true
}

var messagesCSS = gtkutils.CSSAdder(`
	.messages {
		padding-bottom: 4px;
	}
`)

func NewMessages(s *ningen.State, opts Opts) *Messages {
	m := Messages{
		Opts:  opts,
		c:     s,
		fetch: s.Cabinet.MaxMessages(),
	}

	m.Main = gtk.NewBox(gtk.OrientationVertical, 0)

	m.Page = loadstatus.NewPage()
	m.Page.SetChild(m.Main)

	m.Column = handy.NewClamp()
	m.Input = NewInput(&m)
	m.Messages = gtk.NewListBox()
	m.Viewport = gtk.NewViewport(nil, nil)

	// Main actually contains the scrolling window.
	gtkutils.InjectCSS(m.Main, "messagecontainer", "")

	m.Main.SetHExpand(true)
	m.Main.SetVExpand(true)

	gtkutils.InjectCSS(m.Messages, "messages", "")
	messagesCSS(m.Messages.StyleContext())

	m.Scroll = gtk.NewScrolledWindow(nil, nil)
	m.Scroll.Connect("edge-reached", m.onEdgeReached)
	m.Scroll.SetCanFocus(true)
	m.Scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAlways)
	m.Scroll.SetPropagateNaturalHeight(true)
	m.Scroll.SetMinContentWidth(300)
	m.Scroll.SetMinContentHeight(300)
	m.Scroll.SetPlacement(gtk.CornerBottomLeft)

	m.LoadMore = gtk.NewButtonWithLabel("Load More")
	m.LoadMore.Connect("clicked", func() {
		m.LoadMore.SetSensitive(false)
		m.fetchMore(func() {
			m.LoadMore.SetSensitive(true)
		})
	})

	m.Messages.SetSelectionMode(gtk.SelectionNone)
	m.Messages.SetVExpand(true)
	m.Messages.SetHExpand(true)

	m.ScrolledBox = gtk.NewBox(gtk.OrientationVertical, 0)
	m.ScrolledBox.PackStart(m.LoadMore, false, false, 5)
	m.ScrolledBox.PackStart(m.Messages, true, true, 0)

	// List should fill:
	// b.SetSizeRequest(variables.MaxMessageWidth, -1)

	// Column contains the list:
	m.Column.SetMaximumSize(1000)
	m.Column.SetTighteningThreshold(800)
	m.Column.Add(m.ScrolledBox)

	m.Viewport.SetCanFocus(false)
	m.Viewport.SetVAlign(gtk.AlignEnd)
	m.Viewport.SetVScrollPolicy(gtk.ScrollNatural)
	m.Viewport.SetShadowType(gtk.ShadowNone)
	m.Viewport.Add(m.Column) // add col instead of list

	// Scroll to the bottom if we have more things.
	m.Viewport.Connect("size-allocate", func() {
		if m.bottomed {
			m.ScrollToBottom()
		}
	})

	// Fractal does this, but Go is superior.
	adj := m.Scroll.VAdjustment()
	adj.Connect("value-changed", m.onScroll)

	m.Scroll.Add(m.Viewport)

	// Add the message window:
	m.Main.Add(m.Scroll)

	// Add what's needed afterwards:
	m.Main.PackEnd(m.Input, false, false, 0)

	// Set the proper scrolls
	m.Messages.SetFocusHAdjustment(m.Scroll.HAdjustment())
	m.Messages.SetFocusVAdjustment(m.Scroll.VAdjustment())

	// On mouse-press, focus:
	m.Scroll.Connect("button-release-event", func(ev *gdk.Event) bool {
		if gtkutils.EventIsLeftClick(ev) {
			m.Focus()
		}
		return false
	})

	// // On any key-press, focus onto the input box:
	// m.Column.Connect("key-press-event", func(ev *gdk.Event) bool {
	// 	m.Focus()
	// 	// Pass the event in
	// 	m.Input.Input.Event(ev)

	// 	// Drain down the event;
	// 	return false
	// })

	// Set maximum widths
	m.SetWidth(opts.MessageWidth)

	m.injectHandlers()
	m.injectPopup()
	m.ShowAll()
	return &m
}

func (m *Messages) setMainScreen() {
	m.Page.SetChild(m.Main)
}

func (m *Messages) SetWidth(width int) {
	variables.MaxMessageWidth = width
	m.Opts.MessageWidth = width
	m.Column.SetMaximumSize(width)
	m.Input.Clamp.SetMaximumSize(width)
}

// Focus on the input box
func (m *Messages) Focus() {
	m.Input.Input.GrabFocus()
}

func (m *Messages) ChannelID() discord.ChannelID {
	return m.channelID
}

func (m *Messages) GuildID() discord.GuildID {
	return m.guildID
}

func (m *Messages) RecentAuthors(limit int) []discord.UserID {
	ids := make([]discord.UserID, 0, limit)
	added := make(map[discord.UserID]struct{}, limit)

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
	me, _ := m.c.Me()

	for n := len(m.messages) - 1; n >= 0; n-- {
		if msg := m.messages[n]; msg.AuthorID == me.ID {
			return msg
		}
	}

	return nil
}

func (m *Messages) last() *Message {
	if len(m.messages) == 0 {
		return nil
	}
	return m.messages[len(m.messages)-1]
}

func (m *Messages) lastID() discord.MessageID {
	if msg := m.last(); msg != nil {
		return msg.ID
	}
	return 0
}

func (m *Messages) Load(channelID discord.ChannelID) {
	m.channelID = channelID

	// Mark that we're loading messages.
	m.SetLoading()

	// Order: latest is first.
	go func() {
		onErr := func(err error) {
			glib.IdleAdd(func() { m.Page.SetError("Message Error", err) })
		}

		messages, err := m.c.Messages(channelID)
		if err != nil {
			onErr(err)
			return
		}

		isInGuild := len(messages) > 0 && messages[0].GuildID.IsValid()

		// Sort so that latest is last:
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].ID < messages[j].ID
		})

		glib.IdleAdd(func() {
			// Allocate a new empty slice. This is a trade-off to re-using the old
			// slice to re-use messages.
			m.messages = make([]*Message, 0, m.fetch)

			// Iterate from earliest to latest, in a thread-safe function.
			for i := 0; i < len(messages); i++ {
				message := &messages[i]

				w := NewMessage(m.c, message)
				w.UpdateAuthor(m.c, message.GuildID, message.Author)
				m.Insert(w)
			}

			// Iterate backwards, from latest to earliest:
			for i := len(m.messages) - 1; i >= 0; i-- {
				m.messages[i].UpdateExtras(m.c, &messages[i])
			}

			if isInGuild {
				m.guildID = messages[0].GuildID
			}

			m.bottomed = true
			m.ScrollToBottom()
			m.setMainScreen()
		})

		if isInGuild {
			m.c.MemberState.Subscribe(messages[0].GuildID)
		}
	}()
}

func (m *Messages) lastMessageFrom(author discord.UserID) *Message {
	return lastMessageFrom(m.messages, author)
}

func (m *Messages) Cleanup() {
	m.Input.Typing.Stop()

	for _, msg := range m.messages {
		msg.Destroy()
	}

	// Destroy the slice in Go as well, but the GC will pick it up:
	m.messages = nil

	m.channelID = 0
	m.guildID = 0
}

func (m *Messages) ScrollToBottom() {
	// Always set scroll asynchronously, so Gtk can properly calculate the
	// height of children after rendering.
	glib.IdleAdd(func() {
		// Set scroll:
		vAdj := m.Scroll.VAdjustment()
		vAdj.SetValue(vAdj.Upper() - vAdj.PageSize() - 1)
	})
}

func (m *Messages) onScroll(adj *gtk.Adjustment) {
	m.bottomed = adj.Upper()-adj.PageSize() == adj.Value()
}

// mainly used to mark something as read when scrolled to the bottom
func (m *Messages) onEdgeReached(_ *gtk.ScrolledWindow, pos gtk.PositionType) {
	// only count scroll to bottom
	if pos != gtk.PosBottom {
		return
	}
	// If there are no messages.
	if len(m.messages) == 0 {
		return
	}

	r := m.c.ReadState.FindLast(m.channelID)
	if r == nil {
		return
	}

	lastID := m.messages[len(m.messages)-1].ID
	if r.LastMessageID == lastID {
		return
	}

	chID := m.channelID

	// Run this in a goroutine to avoid the mutex acquire from locking the UI
	// thread. Since goroutines are cheap, this isn't a huge issue.
	go func() {
		// Find the latest message and ack it:
		m.c.ReadState.MarkRead(chID, lastID)
	}()
}

func (m *Messages) fetchMore(fetched func()) {
	if len(m.messages) < m.fetch {
		return
	}

	// Grab the first ID
	first := m.messages[0].ID

	channelID := m.channelID
	guildID := m.guildID

	go func() {
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

		glib.IdleAdd(func() {
			// Verify that the new messages still belong to the same channel.
			if m.channelID != channelID {
				// Drop all if not.
				return
			}

			// Allocate a new empty slice for past messages. The earliest message
			// appears first.
			oldMsgs := make([]*Message, 0, len(messages)+len(m.messages))

			for i := 0; i < len(messages); i++ {
				message := &messages[i]

				// Create a new message without insert.
				w := NewMessage(m.c, message)
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
				w.UpdateAuthor(m.c, guildID, message.Author)
				w.UpdateExtras(m.c, message)
				w.ShowAll()
			}

			// Prepend into the slice as well:
			m.messages = append(oldMsgs, m.messages...)
		})
	}()
}

func (m *Messages) cleanOldMessages() {
	// Check the scrolling
	if !m.bottomed {
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
		r.Destroy()
	}

	// Finally, clean the slice
	m.messages = append(m.messages[:0], m.messages[cleanLen:]...)

	// Apparently, Go's obscure slicing behavior allows slicing to capacity, not
	// just length, so we can go back to what we sliced away and nil them.
	var excess = m.messages[len(m.messages) : len(m.messages)+cleanLen]
	// Start setting them to nil for the GC to collect:
	for i := range excess {
		excess[i] = nil
	}
}

func (m *Messages) Upsert(message *discord.Message) *Message {
	// Clean up old messages (thread-safe):
	defer m.cleanOldMessages()

	// Are we sure this is not our message?
	w, updated := m.Update(message)
	if updated {
		return w
	}

	w = NewMessage(m.c, message)
	w.UpdateAuthor(m.c, message.GuildID, message.Author)
	w.UpdateExtras(m.c, message)
	m.Insert(w)

	return w
}

func (m *Messages) Insert(w *Message) {
	// Bind Message's fields to Messages'
	injectMessage(m, w)

	// Try and see if the message should be condensed
	tryCondense(m.messages, w)

	// This adds the message into the list, not call the above Insert().
	m.Messages.Insert(w, -1)
	m.messages = append(m.messages, w)

	w.ShowAll()

	// Gtk is hella buggy.
	w.CheckResize()
}

func (m *Messages) Update(update *discord.Message) (*Message, bool) {
	var target *Message

	for _, message := range m.messages {
		if false ||
			(message.ID.IsValid() && message.ID == update.ID) ||
			(message.Nonce != "" && message.Nonce == update.Nonce) {

			target = message
			break
		}
	}

	if target == nil {
		return nil, false
	}

	target.ID = update.ID

	// Clear the nonce, if any:
	if !target.getAvailable() {
		target.setAvailable(true)
	}

	if update.Content != "" {
		target.UpdateContent(m.c, update)
	}

	target.UpdateExtras(m.c, update)

	return target, true
}

func (m *Messages) Delete(ids ...discord.MessageID) {
	for i, message := range m.messages {
		for _, id := range ids {
			if id == message.ID {
				goto found
			}
		}

		continue

	found:
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
			continue
		}

		// Check if next message is author's:
		if i < len(m.messages) && m.messages[i].AuthorID == oldMessage.AuthorID {
			// Then uncollapse next message:
			m.messages[i].SetCondensed(false)
		}
	}
}

func (m *Messages) findWithNonce(nonce string) *Message {
	for _, message := range m.messages {
		if message.Nonce != nonce {
			continue
		}
		return message
	}
	return nil
}

func (m *Messages) deleteNonce(nonce string) bool {
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
