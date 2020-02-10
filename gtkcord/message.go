package gtkcord

import (
	"fmt"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/httpcache"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	DefaultFetch = 25
	AvatarSize   = 48 // gtk.ICON_SIZE_DIALOG
)

type Messages struct {
	gtk.IWidget
	Main      *gtk.Box
	Scroll    *gtk.ScrolledWindow
	Viewport  *gtk.Viewport
	ChannelID discord.Snowflake
	Messages  []*Message
	guard     sync.Mutex
}

func (m *Messages) Reset(s *state.State, parser *md.Parser) error {
	m.guard.Lock()
	defer m.guard.Unlock()

	if m.Main == nil {
		b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		if err != nil {
			return errors.Wrap(err, "Failed to make box")
		}
		b.SetVExpand(true)
		b.SetHExpand(true)
		m.Main = b

		v, err := gtk.ViewportNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to create viewport")
		}
		must(v.Add, b)
		m.Viewport = v

		s, err := gtk.ScrolledWindowNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to create channel scroller")
		}
		s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)
		m.IWidget = s
		m.Scroll = s

		must(s.Add, v)
	}

	for _, w := range m.Messages {
		must(m.Main.Remove, w.IWidget)
	}
	m.Messages = m.Messages[:0]

	messages, err := s.Messages(m.ChannelID)
	if err != nil {
		return errors.Wrap(err, "Failed to get messages")
	}

	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		w, err := newMessage(s, parser, message)
		if err != nil {
			return errors.Wrap(err, "Failed to render message")
		}

		must(m.Main.Add, w)
		m.Messages = append(m.Messages, w)
	}

	must(m.Main.ShowAll)
	must(m.Viewport.ShowAll)
	must(m.SmartScroll)
	return nil
}

func (m *Messages) Insert(s *state.State, parser *md.Parser, message discord.Message) error {
	m.guard.Lock()
	defer m.guard.Unlock()

	w, err := newMessage(s, parser, message)
	if err != nil {
		return errors.Wrap(err, "Failed to render message")
	}

	m.guard.Lock()
	defer m.guard.Unlock()

	must(m.Main.Add, w)
	must(m.Main.ShowAll)
	must(m.Viewport.ShowAll)
	must(m.SmartScroll)
	m.Messages = append(m.Messages, w)
	return nil
}

func (m *Messages) Update(s *state.State, parser *md.Parser, update discord.Message, async bool) {
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
		go target.UpdateContent(update)
	}
	go target.UpdateExtras(update)
}

func (m *Messages) SmartScroll() {
	adj, err := m.Viewport.GetVAdjustment()
	if err != nil {
		logWrap(err, "Failed to get viewport")
		return
	}

	max := adj.GetUpper()
	cur := adj.GetValue()

	// If the user has scrolled past 10% from the bottom:
	if (max-cur)/max < 0.1 {
		return
	}

	adj.SetValue(max)
	m.Viewport.SetVAdjustment(adj)
}

type Message struct {
	gtk.IWidget

	ID discord.Snowflake

	State  *state.State
	Parser *md.Parser

	Main *gtk.Box

	// Left side:
	Avatar *gtk.Image
	Pixbuf *Pixbuf

	// Right container:
	Right *gtk.Box

	// Right-top container, has author and time:
	RightTop  *gtk.Box
	Author    *gtk.Label
	Timestamp *gtk.Label

	// Right-bottom container, has message contents:
	RightBottom *gtk.Box
	Content     *gtk.TextBuffer  // view declared implicitly
	Extras      []*MessageExtras // embeds, images, etc
}

type MessageExtras struct {
	gtk.IWidget
}

func newMessage(s *state.State, parser *md.Parser, m discord.Message) (*Message, error) {
	main, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create main box")
	}
	margin2(&main.Widget, 5, 15)

	//
	//

	avatar, err := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar user-info")
	}
	avatar.SetSizeRequest(AvatarSize, AvatarSize)
	must(main.Add, avatar)

	//
	//

	right, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create right box")
	}
	must(main.Add, right)

	//
	//

	rtop, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create right top box")
	}
	must(right.Add, rtop)

	rbottom, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create right bottom box")
	}
	must(right.Add, rbottom)

	//
	//

	author, err := gtk.LabelNew("")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create author label")
	}
	author.SetMarkup(bold(m.Author.Username))
	must(rtop.Add, author)

	timestamp, err := gtk.LabelNew("")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create timestamp label")
	}
	timestamp.SetMarkup(
		`<span font_size="smaller">` + m.Timestamp.Format(time.Kitchen) + "</span>")
	timestamp.SetOpacity(0.75)
	timestamp.SetMarginStart(10)
	must(rtop.Add, timestamp)

	go func() {
		var nick = bold(s.AuthorDisplayName(m))
		if color := s.AuthorColor(m); color > 0 {
			nick = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, nick)
		}
		must(author.SetMarkup, nick)
	}()

	//
	//

	ttt, err := gtk.TextTagTableNew()
	if err != nil {
		return nil, errors.Wrap(err, "Faield to create a text tag table")
	}

	msgTb, err := gtk.TextBufferNew(ttt)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a text buffer")
	}
	must(msgTb.SetText, m.Content)

	must(func() {
		msgTv, err := gtk.TextViewNewWithBuffer(nil)
		if err != nil {
			panic("Die: " + err.Error())
		}
		msgTv.SetBuffer(msgTb)
		rbottom.Add(msgTv)
	})

	message := Message{
		ID:          m.ID,
		IWidget:     main,
		State:       s,
		Parser:      parser,
		Main:        main,
		Avatar:      avatar,
		Right:       right,
		RightTop:    rtop,
		Author:      author,
		Timestamp:   timestamp,
		RightBottom: rbottom,
		Content:     msgTb,
	}

	go message.UpdateAvatar(m.Author)
	go message.UpdateContent(m)
	go message.UpdateExtras(m)

	return &message, nil
}

func (m *Message) UpdateAvatar(user discord.User) {
	var url = user.AvatarURL()
	var animated = url[:len(url)-4] == ".gif"

	b, err := httpcache.HTTPGet(url + "?size=64")
	if err != nil {
		logWrap(err, "Failed to GET URL "+url)
		return
	}

	if !animated {
		p, err := NewPixbuf(b, PbSize(AvatarSize, AvatarSize))
		if err != nil {
			logWrap(err, "Failed to get the pixbuf guild icon")
			return
		}

		m.Pixbuf = &Pixbuf{p, nil}
		m.Pixbuf.Set(m.Avatar)
	} else {
		p, err := NewAnimator(b, PbSize(AvatarSize, AvatarSize))
		if err != nil {
			logWrap(err, "Failed to get the pixbuf guild animation")
			return
		}

		m.Pixbuf = &Pixbuf{nil, p}
		m.Pixbuf.Set(m.Avatar)
	}
}

func (m *Message) UpdateContent(update discord.Message) {
	m.Content.Delete(m.Content.GetStartIter(), m.Content.GetEndIter())
	m.Parser.ParseMessage(&update, []byte(update.Content), m.Content)
}

func (m *Message) UpdateExtras(update discord.Message) {
	// TODO
}
