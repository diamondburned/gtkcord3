package message

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
)

func (m *Messages) injectPopup() {
	md.UserPressed = m.userMentionPressed
}

// thread-safe functions

func (m *Messages) userMentionPressed(ev md.PressedEvent, user *discord.GuildUser) {
	// Get the relative position to ev.TextView
	rect := gdk.NewRectangle(int(ev.X()), int(ev.Y()), 0, 0)

	// Make a new popover relatively to TextView
	p := popup.NewPopover(ev.TextView)
	p.SetPosition(gtk.PosRight)
	p.SetPointingTo(&rect)

	body := popup.NewStatefulPopupUser(m.c, user.User, m.GuildID())
	body.ParentStyle = p.StyleContext()

	p.SetChildren(body)
	p.Popup()
}

func (m *Messages) onAvatarClick(msg *Message) {
	// Webhooks don't have users.
	if msg.Webhook {
		return
	}

	p := popup.NewPopover(msg.avatar)
	p.SetPosition(gtk.PosRight)

	body := popup.NewStatefulPopupBody(m.c, msg.AuthorID, m.GuildID())
	body.ParentStyle = p.StyleContext()

	p.SetChildren(body)
	p.Popup()
}

func (m *Messages) onRightClick(msg *Message, btn *gdk.EventButton) {
	menu := gtk.NewMenu()

	m.menuAddAdmin(msg, menu)
	m.menuAddDebug(msg, menu)

	menu.PopupAtPointer(gdk.CopyEventer(btn))
	menu.GrabFocus()
}
