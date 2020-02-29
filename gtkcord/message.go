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
	"github.com/gotk3/gotk3/gdk"
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
	avatarEv *gtk.EventBox
	avatar   *gtk.Image
	pbURL    string

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
	defer log.Benchmark("newMessage")()

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
		go message.UpdateContent(m)
	} else {
		message.updateContent(`<i>` + messageText + `</i>`)
		message.setAvailable(false)
	}

	return message, nil
}

func newMessageCustom(m discord.Message) (message *Message, err error) {
	icon := App.parser.GetIcon("user-info", AvatarSize)

	// What the fuck?
	must(func() {
		var (
			avatar, _   = gtk.ImageNewFromPixbuf(icon)
			avatarEv, _ = gtk.EventBoxNew()

			main, _        = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
			right, _       = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
			rightTop, _    = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
			rightBottom, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

			author, _    = gtk.LabelNew("???")
			timestamp, _ = gtk.LabelNew("")
		)

		mstyle, _ := main.GetStyleContext()
		gtkutils.AddCSSUnsafe(mstyle, `
			.message {
				border-left: 2px solid transparent;
			}
			.message.mentioned {
				border-left: 2px solid rgb(250, 166, 26);
				background-color: rgba(250, 166, 26, 0.05);
			}
		`)

		message = &Message{
			Nonce:     m.Nonce,
			ID:        m.ID,
			AuthorID:  m.Author.ID,
			Timestamp: m.Timestamp.Time().Local(),
			Edited:    m.EditedTimestamp.Time().Local(),

			ExtendedWidget: main,
			Condensed:      false,

			main:        main,
			mainStyle:   mstyle,
			avatarEv:    avatarEv,
			avatar:      avatar,
			right:       right,
			rightTop:    rightTop,
			author:      author,
			timestamp:   timestamp,
			rightBottom: rightBottom,
		}
		defer message.markBusy()()

		gtkutils.InjectCSSUnsafe(message.avatar, "avatar", "")

		message.rightBottom.SetHExpand(true)
		message.rightBottom.SetMarginBottom(5)
		message.rightBottom.SetMarginEnd(AvatarPadding * 2)

		message.avatarEv.SetMarginStart(AvatarPadding * 2)
		message.avatarEv.SetMarginEnd(AvatarPadding)
		message.avatarEv.SetEvents(int(gdk.BUTTON_PRESS_MASK))
		message.avatarEv.Add(message.avatar)
		message.avatarEv.Connect("button_press_event", func() {
			p := NewGuildMemberPopup(message.Messages.Channel.Guild, message.AuthorID)
			p.SetRelativeTo(message.avatar)
			p.Show()
		})

		message.avatar.SetSizeRequest(AvatarSize, AvatarSize)
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

		message.avatarEv.SetMarginTop(6)
		message.right.SetMarginTop(6)

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
		m.mainStyle.AddClass("condensed")
		m.timestamp.SetXAlign(0.5)
		m.timestamp.SetMarkup(smaller("+" + m.CondenseOffset.String()))

		m.main.Remove(m.avatarEv)
		m.main.Remove(m.right)

		// We need to move Timestamp and RightBottom:
		m.rightTop.Remove(m.timestamp)
		m.right.Remove(m.rightBottom)

		m.main.Add(m.timestamp)
		m.main.Add(m.rightBottom)

		return
	}

	m.mainStyle.RemoveClass("condensed")
	m.timestamp.SetXAlign(0.0) // left align
	m.timestamp.SetMarkup(smaller(humanize.TimeAgo(m.Timestamp)))

	m.main.Remove(m.timestamp)
	m.main.Remove(m.rightBottom)

	m.rightTop.Add(m.timestamp)
	m.right.Add(m.rightBottom)

	m.main.Add(m.avatarEv)
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

	async(m.author.SetMarkup, name)
}

func (m *Message) UpdateAuthor(user discord.User) {
	defer m.markBusy()()

	if guildID := m.Messages.Channel.Guild; guildID.Valid() {
		n, err := App.State.Store.Member(guildID, user.ID)
		if err != nil {
			go m.Messages.Channel.Channels.Guild.requestMember(user.ID)
		} else {
			// Update the author name:
			m.updateAuthorName(*n)
			m.markBusy()
		}
	} else {
		async(m.author.SetMarkup, bold(escape(user.Username)))
	}

	var url = user.AvatarURL()
	if url == "" {
		url = AvatarFallbackURL
	}

	if m.pbURL == url {
		return
	}
	m.pbURL = url

	err := cache.SetImageScaled(url+"?size=64", m.avatar, AvatarSize, AvatarSize, cache.Round)
	if err != nil {
		log.Errorln("Failed to get the pixbuf guild icon:", err)
		return
	}
}

func (m *Message) updateContent(s string) {
	defer m.markBusy()()

	m.assertContent()
	must(func() {
		m.content.Delete(m.content.GetStartIter(), m.content.GetEndIter())
		m.content.InsertMarkup(m.content.GetEndIter(), s)
	})
}

func (m *Message) UpdateContent(update discord.Message) {
	defer m.markBusy()()

	if update.Content != "" {
		m.assertContent()
		App.parser.ParseMessage(App.State.Store, &update, []byte(update.Content), m.content)
	}

	for _, mention := range update.Mentions {
		if mention.ID == App.Me.ID {
			async(m.mainStyle.AddClass, "mentioned")
			return
		}
	}

	// We only try this if we know the message is edited. If it's new, there
	// wouldn't be a .mentioned class to remove.
	if update.EditedTimestamp.Valid() {
		async(m.mainStyle.RemoveClass, "mentioned")
	}
}

func (m *Message) assertContent() {
	if m.textView == nil {
		must(func() {
			msgTv, _ := gtk.TextViewNew()
			m.textView = msgTv
			msgTb, _ := msgTv.GetBuffer()
			m.content = msgTb

			msgTv.SetWrapMode(gtk.WRAP_WORD_CHAR)
			msgTv.SetCursorVisible(false)
			msgTv.SetEditable(false)
			msgTv.SetCanFocus(false)

			m.rightBottom.Add(msgTv)
			m.rightBottom.ShowAll()
		})
	}
}

func (m *Message) UpdateExtras(update discord.Message) {
	defer m.markBusy()()

	must(func() {
		for _, extra := range m.extras {
			m.rightBottom.Remove(extra)
		}
	})

	// set to nil so the old slice can be GC'd
	m.extras = nil
	m.extras = append(m.extras, NewEmbed(update)...)
	m.extras = append(m.extras, NewAttachment(update)...)

	async(func() {
		for _, extra := range m.extras {
			m.rightBottom.Add(extra)
		}

		m.rightBottom.ShowAll()
	})
}

func smaller(text string) string {
	return `<span size="smaller">` + text + "</span>"
}
