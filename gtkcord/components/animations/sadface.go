package animations

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

const SadFaceSize = 72

func NewSadFace() gtk.Widgetter {
	i := gtk.NewImage()
	i.SetFromIconName("face-sad-symbolic", 0)
	i.SetPixelSize(SadFaceSize)
	i.SetHExpand(true)
	i.SetVAlign(gtk.AlignCenter)
	i.SetHAlign(gtk.AlignCenter)
	i.Show()

	gtkutils.InjectCSS(i, "", `
		image { opacity: 0.5; }
	`)

	return i
}

func NewSizedSadFace() gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.SetHExpand(true)
	box.SetVAlign(gtk.AlignCenter)
	box.SetVAlign(gtk.AlignCenter)

	img := NewSadFace()
	box.Add(img)
	box.Show()

	return box
}
