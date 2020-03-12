package message

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/gotk3/gotk3/gdk"
)

func (m *Messages) injectPopup() {
	md.UserPressed = m.userMentionPressed
}

func (m *Messages) userMentionPressed(ev *gdk.EventButton, user discord.GuildUser) {
	var rect gdk.Rectangle
	rect.SetX(int(ev.X()))
	rect.SetY(int(ev.Y()))

	p := popup.NewPopover(nil)
	p.SetPointingTo(rect)

	body := popup.NewStatefulPopupBody(m.c, user.ID, m.GuildID)
	body.ParentStyle, _ = p.GetStyleContext()

	p.SetChildren(body)
	p.Show()
}
