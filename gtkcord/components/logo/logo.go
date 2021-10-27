package logo

import (
	"log"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
)

// PNG is set by package main on init.
var PNG []byte

func Pixbuf(sz int) *gdkpixbuf.Pixbuf {
	l := gdkpixbuf.NewPixbufLoader()
	if sz > 0 {
		l.SetSize(sz, sz)
	}

	if err := l.Write(PNG); err != nil {
		log.Panicln("BUG: failed to write logo for pixbuf:", err)
	}

	if err := l.Close(); err != nil {
		log.Panicln("BUG: close logo pixbuf error:", err)
	}

	return l.Pixbuf()
}

func Surface(sz, scale int) *cairo.Surface {
	pixbuf := Pixbuf(sz * scale)
	return gdk.CairoSurfaceCreateFromPixbuf(pixbuf, scale, nil)
}
