package gtkcord

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/pbpool"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	AvatarSize    = 42 // gtk.ICON_SIZE_DND
	AvatarPadding = 10
)

type Message struct {
	ExtendedWidget
	Messages *Messages

	Nonce    string
	ID       discord.Snowflake
	AuthorID discord.Snowflake

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

	Condensed      bool
	CondenseOffset time.Duration
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

	msgTb, err := App.parser.NewTextBuffer()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a text buffer")
	}

	message := &Message{
		Nonce:     m.Nonce,
		ID:        m.ID,
		AuthorID:  m.Author.ID,
		Timestamp: m.Timestamp.Time().Local(),
		Edited:    m.EditedTimestamp.Time().Local(),

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
	must(func() {
		main.SetMarginBottom(2)
		mstyle.AddClass("message")

		rbottom.SetHExpand(true)
		rbottom.SetMarginEnd(AvatarPadding)

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
		timestamp.SetMarginStart(AvatarPadding)

		msgTv, err := gtk.TextViewNewWithBuffer(msgTb)
		if err != nil {
			log.Panicln("Die: " + err.Error())
		}
		msgTv.SetWrapMode(gtk.WRAP_WORD_CHAR)
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

		// Message without a valid ID is probably a sending message. Either way,
		// it's unavailable.
		if !m.ID.Valid() {
			message.setAvailable(false)
		}
	})

	var messageText string

	switch m.Type {
	case discord.GuildMemberJoinMessage:
		messageText = "Joined the server."
	case discord.CallMessage:
		messageText = "Calling you."
	case discord.ChannelIconChangeMessage:
		messageText = "Changed the channel icon."
	case discord.ChannelNameChangeMessage:
		messageText = "Changed the channel name to " + m.Content + "."
	case discord.ChannelPinnedMessage:
		messageText = "Pinned message " + m.ID.String() + "."
	case discord.RecipientAddMessage:
		messageText = "Added " + m.Mentions[0].Username + " to the group."
	case discord.RecipientRemoveMessage:
		messageText = "Removed " + m.Mentions[0].Username + " from the group."
	case discord.NitroBoostMessage:
		messageText = "Boosted the server!"
	case discord.NitroTier1Message:
		messageText = "The server is now Nitro Boosted to Tier 1."
	case discord.NitroTier2Message:
		messageText = "The server is now Nitro Boosted to Tier 2."
	case discord.NitroTier3Message:
		messageText = "The server is now Nitro Boosted to Tier 3."
	default:
	}

	if messageText == "" {
		message.UpdateContent(m)
	} else {
		message.updateContent(`<i>` + messageText + `</i>`)
		message.setAvailable(false)
	}

	return message, nil
}

func (m *Message) getAvailable() bool {
	return m.rightBottom.GetOpacity() > 0.9
}

func (m *Message) setAvailable(available bool) {
	if available {
		must(m.rightBottom.SetOpacity, 1.0)
	} else {
		must(m.rightBottom.SetOpacity, 0.5)
	}
}

func (m *Message) setOffset(last *Message) {
	if last == nil {
		return
	}

	offs := humanize.DuraCeil(m.Timestamp.Sub(last.Timestamp), time.Second)
	m.CondenseOffset = offs
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
		m.main.SetMarginTop(5)
		m.mainStyle.AddClass("condensed")
		m.timestamp.SetXAlign(0.5)
		m.timestamp.SetMarkup(smaller("+" + m.CondenseOffset.String()))

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
	m.mainStyle.RemoveClass("condensed")
	m.timestamp.SetXAlign(0.0) // left align
	m.timestamp.SetMarkup(smaller(humanize.TimeAgo(m.Timestamp)))

	m.main.Remove(m.timestamp)
	m.main.Remove(m.rightBottom)

	m.main.Add(m.avatar)
	m.main.Add(m.right)

	m.rightTop.Add(m.timestamp)
}

func (m *Message) UpdateAuthor(user discord.User) {
	state := App.State

	if guildID := App.Guild.ID; guildID.Valid() {
		var name = escape(user.Username)

		n, err := state.MemberDisplayName(guildID, user.ID)
		if err == nil {
			name = bold(escape(n))

			if color := state.MemberColor(guildID, user.ID); color > 0 {
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

func (m *Message) updateContent(s string) {
	must(func() {
		m.content.InsertMarkup(m.content.GetEndIter(), s)
	})
}

func (m *Message) UpdateContent(update discord.Message) {
	must(func() {
		m.content.Delete(m.content.GetStartIter(), m.content.GetEndIter())
		go App.parser.ParseMessage(&update, []byte(update.Content), m.content)
	})
}

func (m *Message) UpdateExtras(update discord.Message) {
	// TODO
	// must(m.ShowAll)
}

func smaller(text string) string {
	return `<span size="smaller">` + text + "</span>"
}
