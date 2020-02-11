package gtkcord

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const DefaultFetch = 25

type Messages struct {
	ExtendedWidget
	Main      *gtk.Box
	Scroll    *gtk.ScrolledWindow
	Viewport  *gtk.Viewport
	ChannelID discord.Snowflake
	Messages  []*Message
	guard     sync.Mutex

	Resetting atomic.Value
}

func (m *Messages) Reset(s *state.State, parser *md.Parser) error {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.Main == nil {
		b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		if err != nil {
			return errors.Wrap(err, "Failed to make box")
		}
		m.Main = b

		v, err := gtk.ViewportNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to create viewport")
		}
		m.Viewport = v

		s, err := gtk.ScrolledWindowNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to create channel scroller")
		}
		m.ExtendedWidget = s
		m.Scroll = s

		must(func() {
			b.SetVExpand(true)
			b.SetHExpand(true)
			b.SetMarginBottom(15)

			s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)

			v.Connect("size-allocate", m.onSizeAlloc)
			v.Add(b)
			s.Add(v)
		})
	}

	// Mark that we're loading messages.
	m.Resetting.Store(true)

	for _, w := range m.Messages {
		must(m.Main.Remove, w)
	}

	// Order: latest is first.
	messages, err := s.Messages(m.ChannelID)
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
		for _, w := range m.Messages {
			if w.ID == message.ID {
				msg = w
				break
			}
		}

		if msg == nil {
			w, err := newMessage(s, parser, message)
			if err != nil {
				return errors.Wrap(err, "Failed to render message")
			}
			msg = w
		}

		if shouldCondense(newMessages, message) {
			must(msg.SetCondensed, true)
		}

		must(func() {
			m.Main.Add(msg)
			msg.ShowAll()
		})

		// Messages are added, earliest first.
		newMessages = append(newMessages, msg)
	}

	// Set the new slice.
	m.Messages = newMessages

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
			message.UpdateAuthor(s, discordm.Author)
			message.UpdateExtras(parser, discordm)
		}

		// We're done resetting, set this to false.
		m.Resetting.Store(false)
	}()

	return nil
}

func (m *Messages) ShouldCondense(msg discord.Message) bool {
	return shouldCondense(m.Messages, msg)
}

func shouldCondense(msgs []*Message, msg discord.Message) bool {
	if len(msgs) == 0 {
		return false
	}

	var last = msgs[len(msgs)-1]

	return last.AuthorID == msg.Author.ID &&
		msg.Timestamp.Time().Sub(last.Timestamp) < 5*time.Minute
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

	adj.SetValue(max)
	m.Viewport.SetVAdjustment(adj)
}

func (m *Messages) Insert(s *state.State, parser *md.Parser, message discord.Message) error {
	w, err := newMessage(s, parser, message)
	if err != nil {
		return errors.Wrap(err, "Failed to render message")
	}

	semaphore.Go(func() {
		w.UpdateAuthor(s, message.Author)
		w.UpdateExtras(parser, message)
	})

	m.guard.Lock()
	defer m.guard.Unlock()

	if m.ShouldCondense(message) {
		must(w.SetCondensed, true)
	}

	must(func() {
		must(m.Main.Add, w)
		must(m.Main.ShowAll)
	})

	m.Messages = append(m.Messages, w)
	return nil
}

func (m *Messages) Update(s *state.State, parser *md.Parser, update discord.Message) {
	var target *Message

	m.guard.Lock()
	for _, message := range m.Messages {
		if message.ID == update.ID {
			target = message
		}
	}
	m.guard.Unlock()

	if target == nil {
		return
	}
	if update.Content != "" {
		target.UpdateContent(parser, update)
	}
	semaphore.Go(func() {
		target.UpdateExtras(parser, update)
	})
}

func (m *Messages) Delete(id discord.Snowflake) bool {
	m.guard.Lock()
	defer m.guard.Unlock()

	for i, message := range m.Messages {
		if message.ID != id {
			continue
		}

		m.Messages = append(m.Messages[:i], m.Messages[i+1:]...)
		m.Main.Remove(message)
		return true
	}

	return false
}
