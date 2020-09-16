package message

import (
	"fmt"
	"html"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/extras"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/reactions"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Message struct {
	*gtk.ListBoxRow
	style *gtk.StyleContext

	Nonce    string
	ID       discord.MessageID
	AuthorID discord.UserID
	Author   string
	Webhook  bool

	Timestamp time.Time
	Edited    time.Time

	// main container
	main *gtk.Box

	// Left side, nil everything if compact mode
	avatarEv *gtk.EventBox
	avatar   *gtk.Image

	// Right container:
	right *gtk.Box

	// Right-top container, has author and time:
	rightTop  *gtk.Box
	author    *gtk.Label
	timestamp *gtk.Label

	// Right-bottom container, has message contents:
	rightBottom *gtk.Box
	textReveal  *gtk.Revealer
	textView    *gtk.TextView
	content     *gtk.TextBuffer
	reactions   *reactions.Container
	extras      []gtk.IWidget // embeds, images, etc

	Condensed      bool
	CondenseOffset time.Duration

	OnUserClick  func(m *Message)
	OnRightClick func(m *Message, btn *gdk.EventButton)

	busy int32
}

func newMessage(s *ningen.State, m *discord.Message) *Message {
	return semaphore.IdleMust(newMessageUnsafe, s, m).(*Message)
}

func newMessageUnsafe(s *ningen.State, m *discord.Message) *Message {
	// defer log.Benchmark("newMessage")()

	message := newMessageCustomUnsafe(m)
	message.reactions.SetState(s)

	// Message without a valid ID is probably a sending message. Either way,
	// it's unavailable.
	if !m.ID.IsValid() {
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
		message.UpdateContentUnsafe(s, m)
	} else {
		message.customContentUnsafe(`<i>` + messageText + `</i>`)
		message.setAvailableUnsafe(false)
	}

	return message
}

func newMessageCustom(m *discord.Message) (message *Message) {
	return semaphore.IdleMust(newMessageCustomUnsafe, m).(*Message)
}

func newMessageCustomUnsafe(m *discord.Message) (message *Message) {
	// icon := icons.GetIconUnsafe("user-info", variables.AvatarSize)

	var (
		row, _ = gtk.ListBoxRowNew()

		avatar, _   = gtk.ImageNew()
		avatarEv, _ = gtk.EventBoxNew()

		// event box to wrap around main
		mainEv, _ = gtk.EventBoxNew()
		main, _   = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

		right, _       = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		rightTop, _    = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		rightBottom, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

		author, _    = gtk.LabelNew("???")
		timestamp, _ = gtk.LabelNew("")

		txtRv, _ = gtk.RevealerNew()
		msgTv, _ = gtk.TextViewNew()
		msgTb, _ = msgTv.GetBuffer()
	)

	gtkutils.ImageSetIcon(avatar, "user-info", variables.AvatarSize)

	style, _ := main.GetStyleContext()
	style.AddClass("message")

	message = &Message{
		Nonce:     m.Nonce,
		ID:        m.ID,
		AuthorID:  m.Author.ID,
		Author:    m.Author.Username,
		Webhook:   m.WebhookID.IsValid(),
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
		textReveal:  txtRv,
		textView:    msgTv,
		content:     msgTb,
	}

	// Wrap main around an event box
	mainEv.Add(message.main)
	message.ListBoxRow.Add(mainEv)

	message.main.SetHExpand(true)

	// On message (which is in event box) right click:
	mainEv.Connect("button-press-event", func(_ *gtk.EventBox, ev *gdk.Event) bool {
		btn := gdk.EventButtonNewFromEvent(ev)
		if btn.Button() != gdk.BUTTON_SECONDARY {
			return false
		}

		message.OnRightClick(message, btn)
		return true
	})

	gtkutils.InjectCSSUnsafe(message.avatar, "avatar", "")

	message.rightBottom.SetHExpand(true)
	message.rightBottom.SetMarginBottom(5)
	message.rightBottom.SetMarginEnd(variables.AvatarPadding)
	// message.rightBottom.Connect("size-allocate", func() {
	// 	// Hack to force Gtk to recalculate size on changes
	// 	message.rightBottom.SetVExpand(true)
	// 	message.rightBottom.SetVExpand(false)
	// })

	message.avatarEv.SetMarginTop(6)
	message.avatarEv.SetMarginStart(variables.AvatarPadding - 2)
	message.avatarEv.SetMarginEnd(variables.AvatarPadding)
	message.avatarEv.SetEvents(int(gdk.BUTTON_PRESS_MASK))
	message.avatarEv.Add(message.avatar)
	message.avatarEv.Connect("button_press_event", func(_ *gtk.EventBox, ev *gdk.Event) {
		btn := gdk.EventButtonNewFromEvent(ev)
		if btn.Button() != gdk.BUTTON_PRIMARY {
			return
		}

		message.OnUserClick(message)
	})

	message.avatar.SetSizeRequest(variables.AvatarSize, variables.AvatarSize)
	message.avatar.SetVAlign(gtk.ALIGN_START)

	message.author.SetMarkup(
		`<span weight="bold">` + html.EscapeString(m.Author.Username) + `</span>`)
	message.author.SetTooltipText(m.Author.Username)
	message.author.SetSingleLineMode(true)
	message.author.SetLineWrap(false)
	message.author.SetEllipsize(pango.ELLIPSIZE_END)
	message.author.SetXAlign(0.0)

	message.rightTop.Add(message.author)
	gtkutils.InjectCSSUnsafe(message.rightTop, "content", "")

	timestampSize := variables.AvatarSize - 2
	message.timestamp.SetSizeRequest(timestampSize, -1)
	message.timestamp.SetOpacity(0.5)
	message.timestamp.SetYAlign(0.0)
	message.timestamp.SetSingleLineMode(true)
	message.timestamp.SetMarginTop(2)
	message.timestamp.SetMarginStart(variables.AvatarPadding)
	message.timestamp.SetMarginEnd(variables.AvatarPadding)
	message.timestamp.SetTooltipText(m.Timestamp.Format(time.Stamp))
	gtkutils.InjectCSSUnsafe(message.timestamp, "timestamp", "")

	message.right.Add(message.rightTop)
	message.right.SetHExpand(true)

	message.right.SetMarginTop(6)

	msgTv.SetWrapMode(gtk.WRAP_WORD_CHAR)
	msgTv.SetHAlign(gtk.ALIGN_FILL)
	msgTv.SetCursorVisible(false)
	msgTv.SetEditable(false)
	msgTv.SetCanFocus(false)

	// Add the message view into the revealer
	txtRv.Add(msgTv)
	txtRv.SetRevealChild(false)

	message.rightBottom.Add(txtRv)

	// Add a placeholder for reactions
	message.reactions = reactions.NewContainer(m)
	message.rightBottom.Add(message.reactions)

	message.setCondensed()
	return
}

func (m *Message) getAvailableUnsafe() bool {
	return m.rightBottom.GetOpacity() > 0.9
}

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

func (m *Message) updateAuthor(s *ningen.State, gID discord.GuildID, u discord.User) {
	// Webhooks don't have users.
	if gID.IsValid() && !m.Webhook {
		n, err := s.Store.Member(gID, m.AuthorID)
		if err != nil {
			s.MemberState.RequestMember(gID, m.AuthorID)
		} else {
			m.updateMember(s, gID, *n)
			return
		}
	}

	m.UpdateAvatar(u.AvatarURL())
	m.author.SetMarkup(`<span weight="bold">` + html.EscapeString(u.Username) + `</span>`)
}

func (m *Message) UpdateMember(s *ningen.State, gID discord.GuildID, n discord.Member) {
	semaphore.IdleMust(m.updateMember, s, gID, n)
}

func (m *Message) updateMember(s *ningen.State, gID discord.GuildID, n discord.Member) {
	var name = html.EscapeString(n.User.Username)
	if n.Nick != "" {
		name = html.EscapeString(n.Nick)
	}

	m.author.SetTooltipMarkup(name)

	if gID.IsValid() {
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
	cache.AsyncFetchUnsafe(
		url+"?size=64", m.avatar, variables.AvatarSize, variables.AvatarSize, cache.Round)
}

func (m *Message) customContentUnsafe(s string) {
	m.content.Delete(m.content.GetStartIter(), m.content.GetEndIter())
	m.content.InsertMarkup(m.content.GetEndIter(), s)
	m.textReveal.SetRevealChild(true)
}

func (m *Message) UpdateContentUnsafe(s *ningen.State, update *discord.Message) {
	if update.Content != "" {
		md.ParseMessageContent(m.textView, s.Store, update)
		m.textReveal.SetRevealChild(true)
	}

	for _, mention := range update.Mentions {
		if mention.ID == s.Ready.User.ID {
			m.style.AddClass("mentioned")
			return
		}
	}
}

func (m *Message) updateExtras(s *ningen.State, update *discord.Message) {
	for _, extra := range m.extras {
		m.rightBottom.Remove(extra)
	}

	// set to nil so the old slice can be GC'd
	m.extras = nil
	m.extras = append(m.extras, extras.NewEmbedUnsafe(s, update)...)
	m.extras = append(m.extras, extras.NewAttachmentUnsafe(update)...)

	for _, extra := range m.extras {
		m.rightBottom.Add(extra)
	}

	m.rightBottom.ShowAll()
}

func smaller(text string) string {
	return `<span size="smaller">` + text + "</span>"
}
