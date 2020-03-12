package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

func (m *Messages) injectPopup() {
	md.UserPressed = m.userMentionPressed
}

// thread-safe functions

func (m *Messages) userMentionPressed(ev md.PressedEvent, user discord.GuildUser) {
	// Get the relative position to ev.TextView
	var rect gdk.Rectangle
	rect.SetX(int(ev.X()))
	rect.SetY(int(ev.Y()))

	// Make a new popover relatively to TextView
	p := popup.NewPopover(ev.TextView)
	p.SetPointingTo(rect)

	body := popup.NewStatefulPopupBody(m.c, user.ID, m.GuildID)
	body.ParentStyle, _ = p.GetStyleContext()

	p.SetChildren(body)
	p.Show()
}

func (m *Messages) onAvatarClick(msg *Message) {
	// Webhooks don't have users.
	if msg.Webhook {
		return
	}

	p := popup.NewPopover(msg.avatar)

	body := popup.NewStatefulPopupBody(m.c, msg.AuthorID, m.GuildID)
	body.ParentStyle, _ = p.GetStyleContext()

	p.SetChildren(body)
	p.Show()
}

func (m *Messages) onRightClick(msg *Message, btn *gdk.EventButton) {
	menu, _ := gtk.MenuNew()

	m.menuAddAdmin(msg, menu)
	m.menuAddDebug(msg, menu)

	menu.Show()
	menu.PopupAtPointer(btn.Event)
	menu.GrabFocus()
}
