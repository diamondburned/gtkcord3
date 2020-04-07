package header

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/overview"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

type ChMenuButton struct {
	*gtk.Revealer
	Button  *gtk.MenuButton
	Popover *popup.Popover
	spawn   func(p *gtk.Popover) gtkutils.WidgetDestroyer
	// Callbacks! AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
}

func NewChMenuButton() *ChMenuButton {
	r, _ := gtk.RevealerNew()
	r.Show()
	r.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	r.SetTransitionDuration(150)
	r.SetRevealChild(false)

	b, _ := gtk.MenuButtonNew()
	b.Show()
	b.SetHAlign(gtk.ALIGN_CENTER)

	i, err := gtk.ImageNewFromIconName("open-menu-symbolic", gtk.ICON_SIZE_SMALL_TOOLBAR)
	if err != nil {
		log.Fatalln("Failed to create ch menu button:", err)
	}
	i.Show()

	r.Add(b)
	b.Add(i)

	btn := &ChMenuButton{
		Revealer: r,
		Button:   b,
	}
	btn.Popover = popup.NewDynamicPopover(b, func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		if btn.spawn == nil {
			return nil
		}
		return btn.spawn(p)
	})

	return btn
}

func (b *ChMenuButton) SetSpawner(fn func(p *gtk.Popover) gtkutils.WidgetDestroyer) {
	b.spawn = fn
}

func (b *ChMenuButton) Cleanup() {
	semaphore.IdleMust(func() {
		b.SetRevealChild(false)
	})
}

func NewChMenuBody(p *gtk.Popover, s *ningen.State, gID, chID discord.Snowflake) *gtk.Box {
	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.Show()
	gtkutils.Margin(b, 10)

	details := popup.NewButton("Details", func() {
		p.Hide()

		c, err := overview.NewContainer(s, gID, chID)
		if err != nil {
			log.Errorln("Failed to spawn container:", err)
			return
		}

		overview.SpawnDialog(c)
	})

	b.Add(details)

	return b
}
