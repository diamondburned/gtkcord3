package gtkcord

import (
	"github.com/gotk3/gotk3/gtk"
)

const DefaultFetch = 25

type Messages struct {
	Messages []*Message
}

type Message struct {
	gtk.IWidget

	Main *gtk.Box

	// Left side:
	Avatar *gtk.Image
	Pixbuf *Pixbuf
}


