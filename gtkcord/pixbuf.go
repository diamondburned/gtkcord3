package gtkcord

import (
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// WHEN THE ENUM
//
// WHEN THE ENUM
type Pixbuf struct {
	// enum
	Pixbuf    *gdk.Pixbuf
	Animation *gdk.PixbufAnimation
}

func (pb *Pixbuf) Set(img *gtk.Image) {
	switch {
	case pb.Pixbuf != nil:
		semaphore.IdleMust(img.SetFromPixbuf, pb.Pixbuf)
	case pb.Animation != nil:
		semaphore.IdleMust(img.SetFromAnimation, pb.Animation)
	}
}
