package gtkcord

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/humanize"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

const (
	AvatarSize    = 42 // gtk.ICON_SIZE_DND
	AvatarPadding = 10

	AvatarFallbackURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
)

type Message struct {
	gtkutils.ExtendedWidget
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
	pbURL  string

	// Right container:
	right *gtk.Box

	// Right-top container, has author and time:
	rightTop  *gtk.Box
	author    *gtk.Label
	timestamp *gtk.Label

	// Right-bottom container, has message contents:
	rightBottom *gtk.Box
	textView    gtk.IWidget
	content     *gtk.TextBuffer           // view declared implicitly
	extras      []gtkutils.ExtendedWidget // embeds, images, etc

	Condensed      bool
	CondenseOffset time.Duration

	busy atomic.Value
}

func newMessage(m discord.Message) (*Message, error) {
	message, err := newMessageCustom(m)
	if err != nil {
		return nil, err
	}

	defer message.markBusy()()

	// Message without a valid ID is probably a sending message. Either way,
	// it's unavailable.
	if !m.ID.Valid() {
		message.setAvailable(false)
	}

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
	}

	if messageText == "" {
		message.UpdateContent(m)
	} else {
		message.updateContent(`<i>` + messageText + `</i>`)
		message.setAvailable(false)
	}

	return message, nil
}

func newMessageCustom(m discord.Message) (*Message, error) {
	main := must(gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box)
	mstyle := must(main.GetStyleContext).(*gtk.StyleContext)

	message := &Message{
		Nonce:     m.Nonce,
		ID:        m.ID,
		AuthorID:  m.Author.ID,
		Timestamp: m.Timestamp.Time().Local(),
		Edited:    m.EditedTimestamp.Time().Local(),

		ExtendedWidget: main,
		Condensed:      false,

		main:      main,
		mainStyle: mstyle,
		avatar: must(
			gtk.ImageNewFromPixbuf, App.parser.GetIcon("user-info", AvatarSize)).(*gtk.Image),
		right: must(
			gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box),
		rightTop: must(
			gtk.BoxNew, gtk.ORIENTATION_HORIZONTAL, 0).(*gtk.Box),
		author: must(
			gtk.LabelNew, "").(*gtk.Label),
		timestamp: must(
			gtk.LabelNew, "").(*gtk.Label),
		rightBottom: must(
			gtk.BoxNew, gtk.ORIENTATION_VERTICAL, 0).(*gtk.Box),
	}

	defer message.markBusy()()
	gtkutils.InjectCSS(message.avatar, "avatar", "")

	// What the fuck?
	must(func() {
		main.SetMarginBottom(2)
		mstyle.AddClass("message")

		message.rightBottom.SetHExpand(true)
		message.rightBottom.SetMarginEnd(AvatarPadding * 2)

		message.avatar.SetSizeRequest(AvatarSize, AvatarSize)
		message.avatar.SetMarginStart(AvatarPadding * 2)
		message.avatar.SetMarginEnd(AvatarPadding)
		message.avatar.SetVAlign(gtk.ALIGN_START)

		message.author.SetMarkup(bold(m.Author.Username))
		message.author.SetSingleLineMode(true)

		message.rightTop.Add(message.author)

		timestampSize := AvatarSize + AvatarPadding*2 - 1
		message.timestamp.SetSizeRequest(timestampSize, -1)
		message.timestamp.SetOpacity(0.5)
		message.timestamp.SetYAlign(0.0)
		message.timestamp.SetSingleLineMode(true)
		message.timestamp.SetMarginTop(2)
		message.timestamp.SetMarginStart(AvatarPadding)

		message.right.Add(message.rightTop)

		message.avatar.SetMarginTop(10)
		message.right.SetMarginTop(10)

		message.setCondensed()
	})

	return message, nil
}

func (m *Message) markBusy() func() {
	m.busy.Store(true)
	return func() { m.busy.Store(false) }
}

func (m *Message) isBusy() bool {
	v, ok := m.busy.Load().(bool)
	return v && ok
}

func (m *Message) getAvailable() bool {
	return must(m.rightBottom.GetOpacity).(float64) > 0.9
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
	defer m.markBusy()()

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

	m.rightTop.Add(m.timestamp)
	m.right.Add(m.rightBottom)

	m.main.Add(m.avatar)
	m.main.Add(m.right)
}

func (m *Message) updateAuthorName(n discord.Member) {
	defer m.markBusy()()

	var name = bold(escape(n.User.Username))

	if n.Nick != "" {
		name = bold(escape(n.Nick))
	}

	if g, err := App.State.Guild(m.Messages.Channel.Guild); err == nil {
		if color := discord.MemberColor(*g, n); color > 0 {
			name = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, name)
		}
	}

	must(m.author.SetMarkup, name)
}

func (m *Message) UpdateAuthor(user discord.User) {
	defer m.markBusy()()

	if guildID := m.Messages.Channel.Guild; guildID.Valid() {
		n, err := App.State.Store.Member(guildID, user.ID)
		if err != nil {
			m.Messages.Channel.Channels.Guild.requestMember(user.ID)
		} else {
			// Update the author name:
			m.updateAuthorName(*n)
			m.markBusy()
		}
	} else {
		must(m.author.SetMarkup, bold(escape(user.Username)))
	}

	var url = user.AvatarURL()
	if url == "" {
		url = AvatarFallbackURL
	}

	var animated = url[:len(url)-4] == ".gif"

	if m.pbURL == url {
		return
	}
	m.pbURL = url

	var err error

	if !animated {
		err = cache.SetImage(url+"?size=64", m.avatar,
			cache.Resize(AvatarSize, AvatarSize), cache.Round)
	} else {
		err = cache.SetAnimation(url+"?size=64", m.avatar,
			cache.Resize(AvatarSize, AvatarSize), cache.Round)
	}

	if err != nil {
		log.Errorln("Failed to get the pixbuf guild icon:", err)
		return
	}
}

func (m *Message) updateContent(s string) {
	defer m.markBusy()()

	m.assertContent()
	must(func(m *Message) {
		m.content.Delete(m.content.GetStartIter(), m.content.GetEndIter())
		m.content.InsertMarkup(m.content.GetEndIter(), s)
	}, m)
}

func (m *Message) UpdateContent(update discord.Message) {
	defer m.markBusy()()

	if update.Content != "" {
		m.assertContent()
		App.parser.ParseMessage(&update, []byte(update.Content), m.content)
	}
}

func (m *Message) assertContent() {
	if m.textView == nil {
		msgTb := must(App.parser.NewTextBuffer).(*gtk.TextBuffer)
		m.content = msgTb

		msgTv := must(gtk.TextViewNewWithBuffer, msgTb).(*gtk.TextView)
		m.textView = msgTv

		must(msgTv.SetWrapMode, gtk.WRAP_WORD_CHAR)
		must(msgTv.SetCursorVisible, false)
		must(msgTv.SetEditable, false)
		must(msgTv.SetCanFocus, false)

		// Add in what's not covered by SetCondensed.
		must(m.rightBottom.Add, msgTv)
		must(m.rightBottom.ShowAll)
	}
}

func (m *Message) UpdateExtras(update discord.Message) {
	defer m.markBusy()()

	must(func(m *Message) {
		for _, extra := range m.extras {
			m.rightBottom.Remove(extra)
		}
	}, m)

	m.extras = m.extras[:0]
	m.extras = append(m.extras, NewEmbed(update)...)
	m.extras = append(m.extras, NewAttachment(update)...)

	must(func(m *Message) {
		for _, extra := range m.extras {
			m.rightBottom.Add(extra)
		}

		m.rightBottom.ShowAll()
	}, m)
}

func smaller(text string) string {
	return `<span size="smaller">` + text + "</span>"
}
