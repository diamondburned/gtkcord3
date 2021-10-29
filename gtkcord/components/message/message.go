package message

import (
	"fmt"
	"html"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/avatar"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/extras"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message/reactions"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/humanize"
	"github.com/diamondburned/ningen/v2"
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
	avatar   *avatar.Image

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
	extras      []gtk.Widgetter // embeds, images, etc

	errorLabel *gtk.Label

	Condensed      bool
	CondenseOffset time.Duration

	OnUserClick  func(m *Message)
	OnRightClick func(m *Message, btn *gdk.EventButton)

	busy int32
}

func NewMessage(s *ningen.State, m *discord.Message) *Message {
	message := NewMessageCustom(m)
	message.reactions.SetState(s)

	// Message without a valid ID is probably a sending message. Either way,
	// it's unavailable.
	if !m.ID.IsValid() {
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
		message.UpdateContent(s, m)
	} else {
		message.customContent(`<i>` + messageText + `</i>`)
		message.setAvailable(false)
	}

	return message
}

func NewMessageCustom(message *discord.Message) *Message {
	m := Message{
		Nonce:     message.Nonce,
		ID:        message.ID,
		AuthorID:  message.Author.ID,
		Author:    message.Author.Username,
		Webhook:   message.WebhookID.IsValid(),
		Timestamp: message.Timestamp.Time().Local(),
		Edited:    message.EditedTimestamp.Time().Local(),

		ListBoxRow: gtk.NewListBoxRow(),
		Condensed:  false,

		main:        gtk.NewBox(gtk.OrientationHorizontal, 0),
		avatarEv:    gtk.NewEventBox(),
		avatar:      avatar.NewUnwrapped(variables.AvatarSize),
		right:       gtk.NewBox(gtk.OrientationVertical, 0),
		rightTop:    gtk.NewBox(gtk.OrientationHorizontal, 0),
		rightBottom: gtk.NewBox(gtk.OrientationVertical, 0),
		author:      gtk.NewLabel("???"),
		timestamp:   gtk.NewLabel("since the dawn of time"),
		textReveal:  gtk.NewRevealer(),
		textView:    gtk.NewTextView(),
		errorLabel:  gtk.NewLabel(""),
	}

	m.content = m.textView.Buffer()

	m.style = m.main.StyleContext()
	m.style.AddClass("message")

	m.main.SetHExpand(true)

	// Wrap main around an event box
	mainEv := gtk.NewEventBox()
	mainEv.AddEvents(int(0 |
		gdk.ButtonPressMask |
		gdk.EnterNotifyMask |
		gdk.LeaveNotifyMask,
	))
	mainEv.Add(m.main)
	m.ListBoxRow.Add(mainEv)

	// On message (which is in event box) right click:
	mainEv.Connect("button-press-event", func(ev *gdk.Event) bool {
		btn := ev.AsButton()
		if btn.Button() != gdk.BUTTON_SECONDARY {
			return false
		}

		m.OnRightClick(&m, btn)
		return true
	})

	m.avatar.SetFromIconName("user-info", 0)
	m.avatar.SetInitials(message.Author.Username)

	// On message hover, play the avatar animation.
	mainEv.Connect("enter-notify-event", func() { m.avatar.SetPlayAnimation(true) })
	mainEv.Connect("leave-notify-event", func() { m.avatar.SetPlayAnimation(false) })

	gtkutils.InjectCSS(m.avatar, "avatar", "")

	m.rightBottom.SetHExpand(true)
	m.rightBottom.SetMarginBottom(5)
	m.rightBottom.SetMarginEnd(variables.AvatarPadding)
	// message.rightBottom.Connect("size-allocate", func() {
	// 	// Hack to force Gtk to recalculate size on changes
	// 	message.rightBottom.SetVExpand(true)
	// 	message.rightBottom.SetVExpand(false)
	// })

	m.avatarEv.SetMarginTop(6)
	m.avatarEv.SetMarginStart(variables.AvatarPadding - 2)
	m.avatarEv.SetMarginEnd(variables.AvatarPadding)
	m.avatarEv.AddEvents(int(gdk.ButtonPressMask))
	m.avatarEv.Add(m.avatar)
	m.avatarEv.Connect("button-press-event", func(ev *gdk.Event) {
		btn := ev.AsButton()
		if btn.Button() != gdk.BUTTON_PRIMARY {
			return
		}

		m.OnUserClick(&m)
	})

	m.avatar.SetSizeRequest(variables.AvatarSize, variables.AvatarSize)
	m.avatar.SetVAlign(gtk.AlignStart)

	m.author.SetMarkup(
		`<span weight="bold">` + html.EscapeString(message.Author.Username) + `</span>`)
	m.author.SetTooltipText(message.Author.Username)
	m.author.SetSingleLineMode(true)
	m.author.SetLineWrap(false)
	m.author.SetEllipsize(pango.EllipsizeEnd)
	m.author.SetXAlign(0.0)

	m.rightTop.Add(m.author)
	gtkutils.InjectCSS(m.rightTop, "content", "")

	timestampSize := variables.AvatarSize - 2
	m.timestamp.SetSizeRequest(timestampSize, -1)
	m.timestamp.SetOpacity(0.5)
	m.timestamp.SetYAlign(0.0)
	m.timestamp.SetSingleLineMode(true)
	m.timestamp.SetMarginTop(2)
	m.timestamp.SetMarginStart(variables.AvatarPadding)
	m.timestamp.SetMarginEnd(variables.AvatarPadding)
	m.timestamp.SetTooltipText(m.Timestamp.Format(time.Stamp))
	gtkutils.InjectCSS(m.timestamp, "timestamp", "")

	m.right.Add(m.rightTop)
	m.right.SetHExpand(true)
	m.right.SetMarginTop(6)

	m.right.Add(m.errorLabel)
	m.errorLabel.Hide()
	m.errorLabel.ConnectShow(func() {
		m.errorLabel.SetVisible(m.errorLabel.Label() != "")
	})

	m.textView.SetWrapMode(gtk.WrapWordChar)
	m.textView.SetHAlign(gtk.AlignFill)
	m.textView.SetCursorVisible(false)
	m.textView.SetEditable(false)
	m.textView.SetCanFocus(false)

	// Add the message view into the revealer
	m.textReveal.Add(m.textView)
	m.textReveal.SetRevealChild(false)

	m.rightBottom.Add(m.textReveal)

	// Add a placeholder for reactions
	m.reactions = reactions.NewContainer(message)
	m.rightBottom.Add(m.reactions)

	m.setCondensed()
	return &m
}

func (m *Message) getAvailable() bool {
	return m.rightBottom.Opacity() > 0.9
}

func (m *Message) setAvailable(available bool) {
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

// ShowError shows an error on a message.
func (m *Message) ShowError(err error) {
	if err == nil {
		m.errorLabel.SetText("")
		m.errorLabel.Hide()
		return
	}

	m.errorLabel.SetMarkup(fmt.Sprintf(
		`<span color="red"><b>Error:</b> %s</span>`,
		html.EscapeString(err.Error()),
	))
	m.errorLabel.Show()
}

// HideError hides a message's error, if any.
func (m *Message) HideError() { m.ShowError(nil) }

func (m *Message) SetCondensed(condensed bool) {
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

func (m *Message) UpdateAuthor(s *ningen.State, gID discord.GuildID, u discord.User) {
	// Webhooks don't have users.
	if gID.IsValid() && !m.Webhook {
		n, err := s.Cabinet.Member(gID, m.AuthorID)
		if err != nil {
			s.MemberState.RequestMember(gID, m.AuthorID)
		} else {
			m.UpdateMember(s, gID, *n)
			return
		}
	}

	m.UpdateAvatar(u.AvatarURL())
	m.author.SetMarkup(`<span weight="bold">` + html.EscapeString(u.Username) + `</span>`)
}

func (m *Message) UpdateMember(s *ningen.State, gID discord.GuildID, n discord.Member) {
	var name = html.EscapeString(n.User.Username)
	if n.Nick != "" {
		name = html.EscapeString(n.Nick)
	}

	m.author.SetTooltipMarkup(name)

	if gID.IsValid() {
		if g, err := s.Cabinet.Guild(gID); err == nil {
			if color := discord.MemberColor(*g, n); color > 0 {
				name = fmt.Sprintf(`<span fgcolor="#%06X">%s</span>`, color, name)
			}
		}
	}

	m.UpdateAvatar(n.User.AvatarURL())
	m.author.SetMarkup(`<span weight="bold">` + name + `</span>`)
}

func (m *Message) UpdateAvatar(url string) {
	cache.SetImageURLScaled(m.avatar, url+"?size=64", variables.AvatarSize, variables.AvatarSize)
}

func (m *Message) customContent(s string) {
	m.content.SetText("")
	m.content.InsertMarkup(m.content.EndIter(), s)
	m.textReveal.SetRevealChild(true)
}

func (m *Message) UpdateContent(s *ningen.State, update *discord.Message) {
	if update.Content != "" {
		md.ParseMessageContent(m.textView, s, update)
		m.textReveal.SetRevealChild(true)
	}

	me, _ := s.Me()

	for _, mention := range update.Mentions {
		if mention.ID == me.ID {
			m.style.AddClass("mentioned")
			return
		}
	}
}

func (m *Message) UpdateExtras(s *ningen.State, update *discord.Message) {
	for _, extra := range m.extras {
		m.rightBottom.Remove(extra)
	}

	// set to nil so the old slice can be GC'd
	m.extras = nil
	m.extras = append(m.extras, extras.NewEmbed(s, update)...)
	m.extras = append(m.extras, extras.NewAttachment(update)...)

	for _, extra := range m.extras {
		m.rightBottom.Add(extra)
	}

	m.rightBottom.ShowAll()
}

func smaller(text string) string {
	return `<span size="smaller">` + text + "</span>"
}
