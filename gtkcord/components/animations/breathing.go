package animations

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
)

const breathingChar = "●"

func NewBreathing() (gtk.IWidget, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}

	c1, err := gtk.LabelNew(breathingChar)
	if err != nil {
		return nil, err
	}
	c2, err := gtk.LabelNew(breathingChar)
	if err != nil {
		return nil, err
	}
	c3, err := gtk.LabelNew(breathingChar)
	if err != nil {
		return nil, err
	}

	b.Add(c1)
	b.Add(c2)
	b.Add(c3)

	gtkutils.InjectCSSUnsafe(b, "anim-breathing", "")

	return b, nil
}
