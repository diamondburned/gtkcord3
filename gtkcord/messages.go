package gtkcord

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const DefaultFetch = 25

type Messages struct {
	gtkutils.ExtendedWidget
	ChannelID discord.Snowflake
	GuildID   discord.Snowflake

	Main *gtk.Box

	Scroll   *gtk.ScrolledWindow
	Viewport *gtk.Viewport
	Messages *gtk.ListBox

	messages []*Message
	guard    sync.RWMutex

	Resetting atomic.Value

	// Additional components
	Input  *MessageInput
	Typing *TypingState

	// self indicates if the message is from self or not
	OnInsert func(m *Message)
}

func newMessages(chID discord.Snowflake) (*Messages, error) {
	m := &Messages{
		ChannelID: chID,
	}

	main := must(gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box)
	m.Main = main
	m.ExtendedWidget = main

	b := must(gtk.ListBoxNew).(*gtk.ListBox)
	m.Messages = b

	v := must(gtk.ViewportNew,
		nilAdjustment(), nilAdjustment()).(*gtk.Viewport)
	m.Viewport = v

	s := must(gtk.ScrolledWindowNew,
		nilAdjustment(), nilAdjustment()).(*gtk.ScrolledWindow)
	m.Scroll = s

	m.loadTypingState()
	m.loadMessageInput()

	must(func() {
		// Main actually contains the scrolling window.
		gtkutils.InjectCSSUnsafe(main, "messagecontainer", `
			.messagecontainer {
				background-color: @theme_base_color;
			}
		`)
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

		s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)
		s.Show()

		v.Connect("size-allocate", m.onSizeAlloc)
		v.Add(b)
		v.Show()

		s.Add(v)
		s.Show()

		// Add the message window:
		main.Add(s)

		// Add what's needed afterwards:
		main.PackEnd(m.Input, false, false, 0)

		// Hijack Input's box and add the typing indicator:
		m.Input.Main.Add(m.Typing)
		m.Typing.ShowAll()
	})

	return m, nil
}

func (ch *Channel) loadMessages() error {
	if ch.Messages == nil {
		m, err := newMessages(ch.ID)
		if err != nil {
			return err
		}

		m.GuildID = ch.Guild
		m.OnInsert = ch.ackLatest

		ch.Messages = m
	}

	if err := ch.Messages.reset(); err != nil {
		return errors.Wrap(err, "Failed to reset messages")
	}

	return nil
}

func (ch *Channel) ackLatest(m *Message) {
	ch.LastMsg = m.ID
	App.State.MarkRead(ch.ID, ch.LastMsg, m.AuthorID != App.Me.ID)
}

func (ch *Channel) GetMessages() *Messages {
	return ch.Messages
}

func (m *Messages) triggerInsert() {
	if m.OnInsert == nil {
		return
	}
	last := m.Last()
	if last == nil {
		return
	}

	m.OnInsert(last)
}

func (m *Messages) Last() *Message {
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

func (m *Messages) reset() error {
	m.guard.Lock()
	defer m.guard.Unlock()

	// Mark that we're loading messages.
	m.Resetting.Store(true)

	// Order: latest is first.
	messages, err := App.State.Messages(m.ChannelID)
	if err != nil {
		return errors.Wrap(err, "Failed to get messages")
	}

	// Sort so that latest is last:
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].ID < messages[j].ID
	})

	must(func() {
		for _, w := range m.messages {
			if w != nil {
				m.Messages.Remove(w.ListBoxRow)
			}
		}
	})

	// Allocate a new empty slice. This is a trade-off to re-using the old
	// slice to re-use messages.
	var newMessages = make([]*Message, 0, DefaultFetch)

	// WaitGroup for the background goroutines that were spawned:
	var loads = make([]func(), 0, DefaultFetch)

	// Iterate from earliest to latest.
	for i := 0; i < len(messages); i++ {
		message := messages[i]

		msg, err := newMessage(message)
		if err != nil {
			return errors.Wrap(err, "Failed to render message")
		}
		msg.Messages = m

		var condensed = false
		if shouldCondense(newMessages, message) {
			msg.setOffset(lastMessageFrom(newMessages, message.Author.ID))
			condensed = true
		}

		// Messages are added, earliest first.
		newMessages = append(newMessages, msg)
		must(msg.SetCondensed, condensed)
		must(m.Messages.Insert, msg, -1)

		loads = append(loads, func() {
			msg.UpdateAuthor(message.Author)
			msg.UpdateExtras(message)
		})
	}

	// Set the new slice.
	m.messages = newMessages

	// If there are no messages, don't bother.
	if len(newMessages) == 0 {
		m.Resetting.Store(false)
		return nil
	}

	// Show all messages.
	must(m.Messages.ShowAll)

	go func() {
		m.triggerInsert()

		// Iterate backwards, from latest to earliest.
		for i := len(loads) - 1; i >= 0; i-- {
			loads[i]()
		}
		m.Resetting.Store(false)
	}()

	return nil
}

func (m *Messages) ShouldCondense(msg discord.Message) bool {
	return shouldCondense(m.messages, msg)
}

func shouldCondense(msgs []*Message, msg discord.Message) bool {
	if len(msgs) == 0 {
		return false
	}

	var last = msgs[len(msgs)-1]

	return last.AuthorID == msg.Author.ID &&
		msg.Timestamp.Time().Sub(last.Timestamp) < 5*time.Minute
}

func lastMessageFrom(msgs []*Message, author discord.Snowflake) *Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msg := msgs[i]; msg.AuthorID == author && !msg.Condensed {
			return msg
		}
	}
	return nil
}

func (m *Messages) Destroy() {
	m.guard.Lock()
	defer m.guard.Unlock()

	log.Infoln("Destroying messages from old channel.")
	go m.Typing.Stop()

	for _, msg := range m.messages {
		// DESTROY!!!!
		// https://stackoverflow.com/questions/2862509/free-object-widget-in-gtk
		m.Messages.Remove(msg)
	}

	// Destroy the slice in Go as well, but the GC will pick it up:
	m.messages = nil
}

func (m *Messages) onSizeAlloc() {
	adj, err := m.Viewport.GetVAdjustment()
	if err != nil {
		log.Errorln("Failed to get viewport:", err)
		return
	}

	max := adj.GetUpper()
	cur := adj.GetValue() + adj.GetPageSize()

	v, ok := m.Resetting.Load().(bool)
	var loading = ok && v

	// If the scroll is not close to the bottom and we're not loading messages:
	if max-cur > 500 && !loading {
		// Then we don't scroll.
		return
	}

	adj.SetValue(max)
	m.Viewport.SetVAdjustment(adj)

	m.Resetting.Store(false)
}

func (m *Messages) Insert(message discord.Message) error {
	defer func() {
		if message.ID.Valid() {
			m.triggerInsert()
		}
	}()

	// Are we sure this is not our message?
	if m.Update(message) {
		return nil
	}

	w, err := newMessage(message)
	if err != nil {
		return errors.Wrap(err, "Failed to render message")
	}

	return m.insert(w, message)
}

func (m *Messages) insert(w *Message, message discord.Message) error {
	w.Messages = m

	semaphore.Go(func() {
		w.UpdateAuthor(message.Author)
		w.UpdateExtras(message)
	})

	m.guard.Lock()
	defer m.guard.Unlock()

	var condense = m.ShouldCondense(message)
	if condense {
		w.setOffset(lastMessageFrom(m.messages, message.Author.ID))
		must(w.SetCondensed, true)
	}

	must(func() {
		m.Messages.Insert(w, -1)
		w.ShowAll()
	})

	m.messages = append(m.messages, w)
	return nil
}

func (m *Messages) Update(update discord.Message) bool {
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

	// Clear the nonce, if any:
	if !target.getAvailable() {
		target.setAvailable(true)
	}

	target.ID = update.ID
	target.Nonce = ""

	if update.Content != "" {
		target.UpdateContent(update)
	}
	semaphore.Go(func() {
		target.UpdateExtras(update)
	})

	return true
}

func (m *Messages) UpdateMessageAuthor(n discord.Member) {
	m.guard.RLock()
	for _, message := range m.messages {
		if message.AuthorID != n.User.ID {
			continue
		}
		message.updateAuthorName(n)
	}
	m.guard.RUnlock()
}

func (m *Messages) Delete(ids ...discord.Snowflake) (deleted bool) {
	m.guard.Lock()
	defer m.guard.Unlock()

IDLoop:
	for _, id := range ids {
		for i, message := range m.messages {
			if message.ID != id {
				continue
			}

			m.messages = append(m.messages[:i], m.messages[i+1:]...)
			must(m.Messages.Remove, message)

			deleted = true
			continue IDLoop
		}
	}

	return false
}

func (m *Messages) deleteNonce(nonce string) bool {
	m.guard.Lock()
	defer m.guard.Unlock()

	for i, message := range m.messages {
		if message.Nonce != nonce {
			continue
		}

		m.messages = append(m.messages[:i], m.messages[i+1:]...)
		must(m.Messages.Remove, message)
		return true
	}

	return false
}
