package gtkcord

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/pbpool"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	AvatarSize    = 42 // gtk.ICON_SIZE_DND
	AvatarPadding = 10
)

type Message struct {
	ExtendedWidget

	ID       discord.Snowflake
	AuthorID discord.Snowflake
	GuildID  discord.Snowflake

	Timestamp time.Time
	Edited    time.Time

	// State *state.State

	main      *gtk.Box
	mainStyle *gtk.StyleContext

	// Left side, nil everything if compact mode
	avatar *gtk.Image
	pixbuf *Pixbuf
	pbURL  string

	// Right container:
	right *gtk.Box

	// Right-top container, has author and time:
	rightTop  *gtk.Box
	author    *gtk.Label
	timestamp *gtk.Label

	// Right-bottom container, has message contents:
	rightBottom *gtk.Box
	content     *gtk.TextBuffer  // view declared implicitly
	extras      []*MessageExtras // embeds, images, etc

	Condensed bool
}

type MessageExtras struct {
	ExtendedWidget
}

func newMessage(s *state.State, p *md.Parser, m discord.Message) (*Message, error) {
	main, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create main box")
	}
	mstyle, err := main.GetStyleContext()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get main box's style context")
	}
	mstyle.AddClass("message")

	right, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create right box")
	}

	avatar, err := gtk.ImageNewFromPixbuf(p.GetIcon("user-info", AvatarSize))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar user-info")
	}

	rtop, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create right top box")
	}
	rbottom, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create right bottom box")
	}

	author, err := gtk.LabelNew("")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create author label")
	}
	timestamp, err := gtk.LabelNew("")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create timestamp label")
	}

	ttt, err := gtk.TextTagTableNew()
	if err != nil {
		return nil, errors.Wrap(err, "Faield to create a text tag table")
	}

	msgTb, err := gtk.TextBufferNew(ttt)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a text buffer")
	}

	message := Message{
		ID:        m.ID,
		GuildID:   m.GuildID,
		AuthorID:  m.Author.ID,
		Timestamp: m.Timestamp.Time(),
		Edited:    m.EditedTimestamp.Time(),

		ExtendedWidget: main,
		Condensed:      false,

		main:        main,
		mainStyle:   mstyle,
		avatar:      avatar,
		right:       right,
		rightTop:    rtop,
		author:      author,
		timestamp:   timestamp,
		rightBottom: rbottom,
		content:     msgTb,
	}

	// What the fuck?
	must(func() bool {
		main.SetMarginBottom(2)

		rbottom.SetHExpand(true)

		avatar.SetSizeRequest(AvatarSize, AvatarSize)
		avatar.SetProperty("yalign", 0.0)
		avatar.SetMarginStart(AvatarPadding * 2)
		avatar.SetMarginEnd(AvatarPadding)

		author.SetMarkup(bold(m.Author.Username))
		author.SetSingleLineMode(true)

		timestampSize := AvatarSize + AvatarPadding*2 - 1
		timestamp.SetSizeRequest(timestampSize, -1)
		timestamp.SetOpacity(0.5)
		timestamp.SetYAlign(0.0)
		timestamp.SetSingleLineMode(true)
		timestamp.SetMarginTop(2)
		timestamp.SetMarkup(`<span size="smaller">` +
			m.Timestamp.Format(time.Kitchen) +
			`</span>`)

		msgTv, err := gtk.TextViewNewWithBuffer(msgTb)
		if err != nil {
			panic("Die: " + err.Error())
		}
		msgTv.SetMarginEnd(AvatarPadding)
		msgTv.SetWrapMode(gtk.WRAP_WORD)
		msgTv.SetCursorVisible(false)
		msgTv.SetEditable(false)

		// Add in what's not covered by SetCondensed.
		rtop.Add(author)
		rbottom.Add(msgTv)

		right.Add(rtop)
		right.Add(rbottom)

		avatar.SetMarginTop(10)
		right.SetMarginTop(10)

		message.setCondensed()

		return false
	})

	message.UpdateContent(p, m)

	return &message, nil
}

func (m *Message) SetCondensed(condensed bool) {
	if m.Condensed == condensed {
		return
	}
	m.Condensed = condensed
	m.setCondensed()
}

func (m *Message) setCondensed() {
	if m.Condensed {
		m.main.SetMarginTop(2)
		m.timestamp.SetMarginStart(AvatarPadding)
		m.timestamp.SetXAlign(0.5) // center align
		m.mainStyle.AddClass("condensed")

		m.main.Remove(m.avatar)
		m.main.Remove(m.right)

		// We need to move Timestamp and RightBottom:
		m.rightTop.Remove(m.timestamp)
		m.right.Remove(m.rightBottom)

		m.main.Add(m.timestamp)
		m.main.Add(m.rightBottom)

		return
	}

	m.main.SetMarginTop(7)
	m.timestamp.SetMarginStart(7)
	m.timestamp.SetXAlign(0.0) // left align
	m.mainStyle.RemoveClass("condensed")

	m.main.Remove(m.timestamp)
	m.main.Remove(m.rightBottom)

	m.main.Add(m.avatar)
	m.main.Add(m.right)

	m.rightTop.Add(m.timestamp)
}

func (m *Message) UpdateAuthor(state *state.State, user discord.User) {
	if m.GuildID.Valid() {
		var name = user.Username

		n, err := state.MemberDisplayName(m.GuildID, user.ID)
		if err == nil {
			name = bold(escape(n))

			if color := state.MemberColor(m.GuildID, user.ID); color > 0 {
				name = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, name)
			}
		}

		must(m.author.SetMarkup, name)
	}

	var url = user.AvatarURL()
	var animated = url[:len(url)-4] == ".gif"

	if m.pbURL == url {
		return
	}
	m.pbURL = url

	if !animated {
		p, err := pbpool.GetScaled(url+"?size=64", AvatarSize, AvatarSize, pbpool.Round)
		if err != nil {
			// logWrap(err, "Failed to get the pixbuf guild icon")
			return
		}

		m.pixbuf = &Pixbuf{p, nil}
		m.pixbuf.Set(m.avatar)
	} else {
		p, err := pbpool.GetAnimationScaled(url+"?size=64", AvatarSize, AvatarSize, pbpool.Round)
		if err != nil {
			// logWrap(err, "Failed to get the pixbuf guild animation")
			return
		}

		m.pixbuf = &Pixbuf{nil, p}
		m.pixbuf.Set(m.avatar)
	}
}

func (m *Message) UpdateContent(parser *md.Parser, update discord.Message) {
	m.content.Delete(m.content.GetStartIter(), m.content.GetEndIter())
	parser.ParseMessage(&update, []byte(update.Content), m.content)
}

func (m *Message) UpdateExtras(parser *md.Parser, update discord.Message) {
	// TODO
	// must(m.ShowAll)
}
