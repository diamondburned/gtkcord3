package gtkcord

import (
	"fmt"
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
	ChannelID discord.Snowflake
	Messages  []*Message
}

// func (m *Messages) Update(s *state.State, parser *md.Parser) error {

// }

type Message struct {
	gtk.IWidget

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

	//
	//

	avatar, err := gtk.ImageNewFromIconName("user-info", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar user-info")
	}
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

	author, err := gtk.LabelNew("")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create author label")
	}
	author.SetMarkup(bold(m.Author.Username))
	must(rtop.Add, author)

	timestamp, err := gtk.LabelNew(m.Timestamp.Format(time.Kitchen))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create timestamp label")
	}
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

	rbottom, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create right bottom box")
	}
	must(right.Add, rbottom)

	msgTt, err := md.NewTagTable()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a tag table")
	}

	msgTb, err := gtk.TextBufferNew(msgTt)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a text buffer")
	}

	msgTv, err := gtk.TextViewNewWithBuffer(msgTb)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a text view")
	}
	must(rbottom.Add, msgTv)

	message := Message{
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
		p, err := NewPixbuf(b, PbSize(IconSize, IconSize))
		if err != nil {
			logWrap(err, "Failed to get the pixbuf guild icon")
			return
		}

		m.Pixbuf = &Pixbuf{p, nil}
		m.Pixbuf.Set(m.Avatar)
	} else {
		p, err := NewAnimator(b, PbSize(IconSize, IconSize))
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
