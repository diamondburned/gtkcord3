package header

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
)

type Back struct {
	*gtk.Revealer
	Button *gtk.MenuButton

	OnClick func()
}

func NewBack() *Back {
	r := gtk.NewRevealer()
	r.SetRevealChild(false)
	r.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)
	r.SetTransitionDuration(150)
	r.SetMarginStart(variables.AvatarPadding)
	r.SetMarginEnd(variables.AvatarPadding)
	r.Show()

	mb := gtk.NewMenuButton()
	mb.SetSensitive(true)
	mb.SetHAlign(gtk.AlignCenter)
	mb.Show()
	mb.SetSizeRequest(variables.AvatarSize, -1)
	r.Add(mb)

	i := gtk.NewImageFromIconName("go-next-symbolic", int(gtk.IconSizeLargeToolbar))
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
