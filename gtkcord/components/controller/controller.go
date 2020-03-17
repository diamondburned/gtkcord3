package controller

import (
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

type Container struct {
	*gtk.Box
	Buttons []*Button
}

type Button struct {
	Controlled

	*gtk.MenuButton
	Icon *gtk.Image
}

type Controlled interface {
	OnClick(b *Button)
}

func New() *Container {
	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.Show()
	b.SetHAlign(gtk.ALIGN_END)

	return &Container{
		Box:     b,
		Buttons: []*Button{},
	}
}

func (c *Container) Add(icon string, ctrl Controlled, active bool) *Button {
	mb, _ := gtk.MenuButtonNew()
	mb.Show()
	mb.SetHAlign(gtk.ALIGN_CENTER)
	mb.SetActive(active)
	mb.SetSensitive(true)

	i, err := gtk.ImageNewFromIconName(icon, gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		log.Panicln("Failed to load icon:", err)
	}
	i.Show()
	mb.Add(i)

	b := &Button{
		Controlled: ctrl,
		MenuButton: mb,
		Icon:       i,
	}

	mb.Connect("button-release-event", func() bool {
		ctrl.OnClick(b)
		return true
	})

	c.Box.Add(b)
	c.Buttons = append(c.Buttons, b)

	return b
}

func (b *Button) SetActive(active bool) {
	b.MenuButton.SetActive(active)
}

func (b *Button) ToggleActive() {
	b.MenuButton.SetActive(!b.MenuButton.GetActive())
}
