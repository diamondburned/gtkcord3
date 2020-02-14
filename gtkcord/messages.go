package gtkcord

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const DefaultFetch = 25

type Messages struct {
	ExtendedWidget
	Channel *Channel

	Main *gtk.Box

	Scroll   *gtk.ScrolledWindow
	Viewport *gtk.Viewport
	Messages *gtk.Box

	messages []*Message
	guard    sync.Mutex

	Resetting atomic.Value

	Input *MessageInput
}

func (ch *Channel) loadMessages() error {
	if ch.Messages == nil {
		ch.Messages = &Messages{
			Channel: ch,
		}
	}

	m := ch.Messages

	m.guard.Lock()
	defer m.guard.Unlock()

	if m.Main == nil {
		main, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		if err != nil {
			return errors.Wrap(err, "Failed to make box")
		}
		m.Main = main
		m.ExtendedWidget = main

		b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		if err != nil {
			return errors.Wrap(err, "Failed to make box")
		}
		m.Messages = b

		v, err := gtk.ViewportNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to create viewport")
		}
		m.Viewport = v

		s, err := gtk.ScrolledWindowNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to create channel scroller")
		}
		m.Scroll = s

		if err := m.loadMessageInput(); err != nil {
			return errors.Wrap(err, "Failed to load message input")
		}

		must(func() {
			b.SetVExpand(true)
			b.SetHExpand(true)
			b.SetMarginBottom(15)

			s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)

			v.Connect("size-allocate", m.onSizeAlloc)
			v.Add(b)
			s.Add(v)

			// Main actually contains the scrolling window.
			main.Add(s)
		})
	}

	// Mark that we're loading messages.
	m.Resetting.Store(true)

	must(func() {
		for _, w := range m.messages {
			m.Messages.Remove(w)
		}
	})

	// Order: latest is first.
	messages, err := App.State.Messages(ch.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to get messages")
	}

	// Allocate a new empty slice. This is a trade-off to re-using the old
	// slice to re-use messages.
	var newMessages = make([]*Message, 0, DefaultFetch)

	// Iterate from earliest to latest.
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		var msg *Message

		// See if we could find the message in our old list:
		for _, w := range m.messages {
			if w.ID == message.ID {
				msg = w
				break
			}
		}

		if msg == nil {
			w, err := newMessage(App.State, App.parser, message)
			if err != nil {
				return errors.Wrap(err, "Failed to render message")
			}
			msg = w
		}

		if shouldCondense(newMessages, message) {
			msg.setOffset(lastMessageFrom(newMessages, message.Author.ID))
			msg.SetCondensed(true)
		}

		// Messages are added, earliest first.
		newMessages = append(newMessages, msg)
	}

	// Set the new slice.
	m.messages = newMessages

	must(func() {
		for _, msg := range newMessages {
			m.Messages.Add(msg)
			msg.ShowAll()
		}
	})

	// Hack around the mutex
	copiedMsg := append([]*Message{}, newMessages...)

	go func() {
		// Revert to latest is last, earliest is first.
		for L, R := 0, len(messages)-1; L < R; L, R = L+1, R-1 {
			messages[L], messages[R] = messages[R], messages[L]
		}

		// Iterate in reverse, so latest first.
		for i := len(copiedMsg) - 1; i >= 0; i-- {
			message, discordm := copiedMsg[i], messages[i]
			message.UpdateAuthor(discordm.Author)
			message.UpdateExtras(discordm)
		}

		// We're done resetting, set this to false.
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

func (m *Messages) onSizeAlloc() {
	adj, err := m.Viewport.GetVAdjustment()
	if err != nil {
		logWrap(err, "Failed to get viewport")
		return
	}

	max := adj.GetUpper()
	cur := adj.GetValue()

	v, ok := m.Resetting.Load().(bool)
	var loading = ok && v

	// If the scroll is not close to the bottom and we're not loading messages:
	if max-cur > 2500 && !loading {
		// Then we don't scroll.
		return
	}

	log.Debugln("Scrolling because", max, "-", cur, "> 2500, and loading =", loading)

	adj.SetValue(max)
	m.Viewport.SetVAdjustment(adj)
}

func (m *Messages) Insert(message discord.Message) error {
	// Are we sure this is not our message?
	if m.Update(message) {
		return nil
	}

	w, err := newMessage(App.State, App.parser, message)
	if err != nil {
		return errors.Wrap(err, "Failed to render message")
	}

	semaphore.Go(func() {
		w.UpdateAuthor(message.Author)
		w.UpdateExtras(message)
	})

	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ShouldCondense(message) {
		w.setOffset(lastMessageFrom(m.messages, message.Author.ID))
		w.SetCondensed(true)
	}

	must(func() {
		m.Messages.Add(w)
		w.ShowAll()
	})

	m.messages = append(m.messages, w)
	return nil
}

func (m *Messages) Update(update discord.Message) bool {
	var target *Message

	m.guard.Lock()
	for _, message := range m.messages {
		if false ||
			(message.ID.Valid() && message.ID == update.ID) ||
			(message.Nonce != "" && message.Nonce == update.Nonce) {

			target = message
			break
		}
	}
	m.guard.Unlock()

	if target == nil {
		return false
	}

	// Clear the nonce, if any:
	if !target.getAvailable() {
		target.setAvailable(true)
	}

	log.Println("New message update, nonce:", update.Nonce, target.getAvailable())

	if update.Content != "" {
		target.UpdateContent(update)
	}
	semaphore.Go(func() {
		target.UpdateExtras(update)
	})

	return true
}

func (m *Messages) Delete(id discord.Snowflake) bool {
	m.guard.Lock()
	defer m.guard.Unlock()

	for i, message := range m.messages {
		if message.ID != id {
			continue
		}

		m.messages = append(m.messages[:i], m.messages[i+1:]...)
		must(m.Messages.Remove, message)
		return true
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
