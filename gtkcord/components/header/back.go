package header

import (
	"log"

	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/gotk3/gotk3/gtk"
)

type Back struct {
	*gtk.Revealer
	Button *gtk.MenuButton

	OnClick func()
}

func NewBack() *Back {
	r, _ := gtk.RevealerNew()
	r.Show()
	r.SetRevealChild(false)
	r.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	r.SetTransitionDuration(150)
	r.SetMarginStart(message.AvatarPadding)
	r.SetMarginEnd(message.AvatarPadding)

	mb, _ := gtk.MenuButtonNew()
	mb.SetSensitive(true)
	mb.SetHAlign(gtk.ALIGN_CENTER)
	mb.Show()
	mb.SetSizeRequest(message.AvatarSize, -1)
	r.Add(mb)

	i, err := gtk.ImageNewFromIconName("go-previous-symbolic", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		log.Fatalln("Failed to load icon:", err)
	}
	i.Show()
	mb.Add(i)

	back := &Back{Revealer: r, Button: mb}

	mb.Connect("button-release-event", func() bool {
		if back.OnClick != nil {
			back.OnClick()
		}

		return true
	})

	return back
}
