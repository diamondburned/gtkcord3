package message

import (
	"fmt"
	"html"
	"sync/atomic"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
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
	*gtk.ListBoxRow
	style *gtk.StyleContext

	Nonce    string
	ID       discord.Snowflake
	AuthorID discord.Snowflake

	Timestamp time.Time
	Edited    time.Time

	// main container
	main *gtk.Box

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

	OnUserClick func(m *Message)

	busy int32
}

func newMessage(s *ningen.State, m *discord.Message) *Message {
	return semaphore.IdleMust(newMessageUnsafe, s, m).(*Message)
}

func newMessageUnsafe(s *ningen.State, m *discord.Message) *Message {
	defer log.Benchmark("newMessage")()

	message := newMessageCustomUnsafe(m)
	defer message.markBusy()()

	// Message without a valid ID is probably a sending message. Either way,
	// it's unavailable.
	if !m.ID.Valid() {
		message.setAvailableUnsafe(false)
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
		go message.UpdateContent(s, m)
	} else {
		message.updateContentUnsafe(`<i>` + messageText + `</i>`)
		message.setAvailableUnsafe(false)
	}

	return message
}

func newMessageCustom(m *discord.Message) (message *Message) {
	return semaphore.IdleMust(newMessageCustomUnsafe, m).(*Message)
}

func newMessageCustomUnsafe(m *discord.Message) (message *Message) {
	// icon := icons.GetIconUnsafe("user-info", AvatarSize)

	var (
		row, _ = gtk.ListBoxRowNew()

		avatar, _   = gtk.ImageNew()
		avatarEv, _ = gtk.EventBoxNew()

		main, _        = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		right, _       = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		rightTop, _    = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		rightBottom, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

		author, _    = gtk.LabelNew("???")
		timestamp, _ = gtk.LabelNew("")
	)

	gtkutils.ImageSetIcon(avatar, "user-info", AvatarSize)

	style, _ := row.GetStyleContext()
	style.AddClass("message")
	gtkutils.AddCSSUnsafe(style, `
		.message {
			padding: 0;
		}
	`)

	// Set the message's underlying box style:
	gtkutils.InjectCSSUnsafe(main, "", `
		.message > box {
			border-left: 2px solid transparent;
		}
		.message.mentioned > box {
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

		ListBoxRow: row,
		style:      style,
		Condensed:  false,

		main:        main,
		avatarEv:    avatarEv,
		avatar:      avatar,
		right:       right,
		rightTop:    rightTop,
		author:      author,
		timestamp:   timestamp,
		rightBottom: rightBottom,
	}
	defer message.markBusy()()

	message.ListBoxRow.Add(message.main)

	gtkutils.InjectCSSUnsafe(message.avatar, "avatar", "")

	message.rightBottom.SetHExpand(true)
	message.rightBottom.SetMarginBottom(5)
	message.rightBottom.SetMarginEnd(AvatarPadding * 2)
	message.rightBottom.Connect("size-allocate", func() {
		// Hack to force Gtk to recalculate size on changes
		message.rightBottom.SetVExpand(true)
		message.rightBottom.SetVExpand(false)
	})

	message.avatarEv.SetMarginStart(AvatarPadding * 2)
	message.avatarEv.SetMarginEnd(AvatarPadding)
	message.avatarEv.SetEvents(int(gdk.BUTTON_PRESS_MASK))
	message.avatarEv.Add(message.avatar)
	message.avatarEv.Connect("button_press_event", func() {
		if message.OnUserClick != nil {
			message.OnUserClick(message)
		}
		// p := msgs.c.SpawnUserPopup(message.Messages.GuildID, message.AuthorID)
		// p.SetRelativeTo(message.avatar)
		// p.Show()
	})

	message.avatar.SetSizeRequest(AvatarSize, AvatarSize)
	message.avatar.SetVAlign(gtk.ALIGN_START)

	message.author.SetMarkup(
		`<span weight="bold">` + html.EscapeString(m.Author.Username) + `</span>`)
	message.author.SetSingleLineMode(true)

	message.rightTop.Add(message.author)
	gtkutils.InjectCSSUnsafe(message.rightTop, "content", "")

	timestampSize := AvatarSize + AvatarPadding - 1
	message.timestamp.SetSizeRequest(timestampSize, -1)
	message.timestamp.SetOpacity(0.5)
	message.timestamp.SetYAlign(0.0)
	message.timestamp.SetSingleLineMode(true)
	message.timestamp.SetMarginTop(2)
	message.timestamp.SetMarginStart(AvatarPadding)
	message.timestamp.SetMarginEnd(AvatarPadding)
	gtkutils.InjectCSSUnsafe(message.timestamp, "timestamp", `
		.message.condensed .timestamp {
			opacity: 0;
		}
		.message.condensed:hover .timestamp {
			opacity: 1;
		}
	`)

	message.right.Add(message.rightTop)

	message.avatarEv.SetMarginTop(6)
	message.right.SetMarginTop(6)

	message.setCondensed()

	return
}

func (m *Message) markBusy() func() {
	atomic.AddInt32(&m.busy, 1)
	return func() { atomic.AddInt32(&m.busy, -1) }
}

func (m *Message) isBusy() bool {
	return atomic.LoadInt32(&m.busy) == 0
}

// func (m *Message) getAvailable() bool {
// 	return semaphore.IdleMust(m.rightBottom.GetOpacity).(float64) > 0.9
// }

func (m *Message) getAvailableUnsafe() bool {
	return m.rightBottom.GetOpacity() > 0.9
}

// func (m *Message) setAvailable(available bool) {
// 	if available {
// 		semaphore.IdleMust(m.rightBottom.SetOpacity, 1.0)
// 	} else {
// 		semaphore.IdleMust(m.rightBottom.SetOpacity, 0.5)
// 	}
// }

func (m *Message) setAvailableUnsafe(available bool) {
	if available {
		m.rightBottom.SetOpacity(1.0)
	} else {
		m.rightBottom.SetOpacity(0.5)
	}
}

func (m *Message) setOffset(last *Message) {
	if last == nil {
		return
	}

	offs := humanize.DuraCeil(m.Timestamp.Sub(last.Timestamp), time.Second)
	m.CondenseOffset = offs
}

func (m *Message) SetCondensedUnsafe(condensed bool) {
	defer m.markBusy()()

	if m.Condensed == condensed {
		return
	}
	m.Condensed = condensed
	m.setCondensed()
}

func (m *Message) setCondensed() {
	if m.Condensed {
		m.style.AddClass("condensed")
		m.timestamp.SetXAlign(1.0)
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

	m.style.RemoveClass("condensed")
	m.timestamp.SetXAlign(0.0) // left align
	m.timestamp.SetMarkup(smaller(humanize.TimeAgo(m.Timestamp)))

	m.main.Remove(m.timestamp)
	m.main.Remove(m.rightBottom)

	m.rightTop.Add(m.timestamp)
	m.right.Add(m.rightBottom)

	m.main.Add(m.avatarEv)
	m.main.Add(m.right)
}

func (m *Message) UpdateAuthor(s *ningen.State, gID discord.Snowflake, u discord.User) {
	semaphore.IdleMust(m.updateAuthor, s, gID, u)
}

func (m *Message) updateAuthor(s *ningen.State, gID discord.Snowflake, u discord.User) {
	if gID.Valid() {
		n, err := s.Store.Member(gID, u.ID)
		if err != nil {
			go s.RequestMember(gID, u.ID)
		} else {
			m.updateMember(s, gID, *n)
			return
		}
	}

	m.UpdateAvatar(u.AvatarURL())
	m.author.SetMarkup(`<span weight="bold">` + html.EscapeString(u.Username) + `</span>`)
}

func (m *Message) UpdateMember(s *ningen.State, gID discord.Snowflake, n discord.Member) {
	semaphore.IdleMust(m.updateMember, s, gID, n)
}

func (m *Message) updateMember(s *ningen.State, gID discord.Snowflake, n discord.Member) {
	var name = html.EscapeString(n.User.Username)
	if n.Nick != "" {
		name = html.EscapeString(n.Nick)
	}

	if gID.Valid() {
		if g, err := s.Store.Guild(gID); err == nil {
			if color := discord.MemberColor(*g, n); color > 0 {
				name = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, name)
			}
		}
	}

	m.UpdateAvatar(n.User.AvatarURL())
	m.author.SetMarkup(`<span weight="bold">` + name + `</span>`)
}

func (m *Message) UpdateAvatar(url string) {
	defer m.markBusy()()

	if url == "" {
		url = AvatarFallbackURL
	}

	if m.pbURL == url {
		return
	}
	m.pbURL = url

	// o := m.avatar.Object
	// runtime.KeepAlive(o)

	go func(img *gtk.Image) {

		// image := &gtk.Image{
		// 	Widget: gtk.Widget{
		// 		InitiallyUnowned: glib.InitiallyUnowned{
		// 			Object: o,
		// 		},
		// 	},
		// }

		err := cache.SetImageScaled(url+"?size=64", img, AvatarSize, AvatarSize, cache.Round)
		if err != nil {
			log.Errorln("Failed to get the pixbuf guild icon:", err)
			return
		}
	}(m.avatar)
}

func (m *Message) updateContentUnsafe(s string) {
	m.assertContentUnsafe()
	m.content.Delete(m.content.GetStartIter(), m.content.GetEndIter())
	m.content.InsertMarkup(m.content.GetEndIter(), s)
}

func (m *Message) UpdateContent(s *ningen.State, update *discord.Message) {
	defer m.markBusy()()

	if update.Content != "" {
		semaphore.IdleMust(m.assertContentUnsafe)
		md.ParseMessage(s, update, []byte(update.Content), m.content)
	}

	for _, mention := range update.Mentions {
		if mention.ID == s.Ready.User.ID {
			semaphore.Async(m.style.AddClass, "mentioned")
			return
		}
	}

	// We only try this if we know the message is edited. If it's new, there
	// wouldn't be a .mentioned class to remove.
	if update.EditedTimestamp.Valid() {
		semaphore.Async(m.style.RemoveClass, "mentioned")
	}
}

func (m *Message) assertContentUnsafe() {
	if m.textView == nil {
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
	}
}

func (m *Message) UpdateExtras(s *ningen.State, update *discord.Message) {
	defer m.markBusy()()

	semaphore.IdleMust(func() {
		for _, extra := range m.extras {
			m.rightBottom.Remove(extra)
		}
	})

	// set to nil so the old slice can be GC'd
	m.extras = nil
	m.extras = append(m.extras, NewEmbed(s, update)...)
	m.extras = append(m.extras, NewAttachment(update)...)

	semaphore.Async(func() {
		for _, extra := range m.extras {
			m.rightBottom.Add(extra)
		}

		m.rightBottom.ShowAll()
	})
}

func smaller(text string) string {
	return `<span size="smaller">` + text + "</span>"
}
