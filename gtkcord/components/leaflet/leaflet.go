package leaflet

import "github.com/gotk3/gotk3/gtk"

type Container struct {
	*gtk.Overlay
	Grid *gtk.Grid

	Main gtk.IWidget // 2

	Left  Revealer // 1
	Right Revealer // 3

	leftRevealed  bool
	rightRevealed bool
}

type Revealer interface {
	gtk.IWidget
	SetRevealChild(bool)
	GetRevealChild() bool
}

func (c *Container) SetMobile() {
	// For our mobile UI, we need to take the left and right revealers and put
	// them onto the overlay:

	// Before we collapse the revealers, we should store its state:
	c.leftRevealed = c.Left.GetRevealChild()
	c.rightRevealed = c.Right.GetRevealChild()

	// Then we uncollapse the revealers:
	c.Left.SetRevealChild(false)
	c.Right.SetRevealChild(false)

}
