package animations

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

const breathingChar = "●"

// NewBreathing creates a new breathing animation of 3 fading dots.
func NewBreathing() gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	c1 := gtk.NewLabel(breathingChar)
	c2 := gtk.NewLabel(breathingChar)
	c3 := gtk.NewLabel(breathingChar)

	box.Add(c1)
	box.Add(c2)
	box.Add(c3)

	gtkutils.InjectCSS(box, "anim-breathing", "")

	return box
}
