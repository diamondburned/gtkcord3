package animations

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const SadFaceSize = 72

func NewSadFace() (gtk.IWidget, error) {
	i, err := gtk.ImageNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new gtk.Image")
	}
	gtkutils.ImageSetIcon(i, "face-sad-symbolic", SadFaceSize)
	i.SetHExpand(true)
	i.SetVAlign(gtk.ALIGN_CENTER)
	i.SetHAlign(gtk.ALIGN_CENTER)
	i.ShowAll()

	gtkutils.InjectCSSUnsafe(i, "", `
		image { opacity: 0.5; }
	`)

	return i, nil
}

func NewSizedSadFace() (gtkutils.WidgetSizeRequester, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return nil, err
	}
	b.SetHExpand(true)
	b.SetVAlign(gtk.ALIGN_CENTER)
	b.SetVAlign(gtk.ALIGN_CENTER)

	i, err := NewSadFace()
	if err != nil {
		return nil, err
	}

	b.Add(i)
	b.ShowAll()

	return b, nil
}
