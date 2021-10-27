package header

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/overview"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

type ChMenuButton struct {
	*gtk.Revealer
	Button  *gtk.MenuButton
	Popover *popup.Popover
	spawn   func(p *gtk.Popover) gtk.Widgetter
	// Callbacks! AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
}

func NewChMenuButton() *ChMenuButton {
	r := gtk.NewRevealer()
	r.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	r.SetTransitionDuration(150)
	r.SetRevealChild(false)
	r.Show()

	b := gtk.NewMenuButton()
	b.Show()
	b.SetHAlign(gtk.AlignCenter)

	i := gtk.NewImageFromIconName("open-menu-symbolic", int(gtk.IconSizeSmallToolbar))
	i.Show()

	r.Add(b)
	b.Add(i)

	btn := &ChMenuButton{
		Revealer: r,
		Button:   b,
	}
	btn.Popover = popup.NewDynamicPopover(b, func(p *gtk.Popover) gtk.Widgetter {
		if btn.spawn == nil {
			log.Errorln("chmenu: missing btn.spawn")
			return nil
		}
		return btn.spawn(p)
	})

	return btn
}

func (b *ChMenuButton) SetSpawner(fn func(p *gtk.Popover) gtk.Widgetter) {
	b.spawn = fn
}

func (b *ChMenuButton) Cleanup() {
	b.SetRevealChild(false)
}

func NewChMenuBody(
	p *gtk.Popover, s *ningen.State, gID discord.GuildID, chID discord.ChannelID) *gtk.Box {

	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.Show()
	gtkutils.Margin(b, 10)

	details := popup.NewButton("Details", func() {
		p.Popdown()
		overview.SpawnDialog(overview.NewContainer(s, gID, chID))
	})

	b.Add(details)

	return b
}
