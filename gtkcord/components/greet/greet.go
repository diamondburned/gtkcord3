package greet

import (
	"fmt"
	"html"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
)

const IconSize = 56

type Greeter struct {
	gtk.Box
	i *gtk.Image
	l *gtk.Label
}

func NewGreeter() *Greeter {
	g := &Greeter{Box: *gtk.NewBox(gtk.OrientationVertical, 8)}
	g.StyleContext().AddClass("greeter")

	g.i = gtk.NewImage()
	g.i.SetHAlign(gtk.AlignCenter)
	g.i.SetPixelSize(IconSize)

	g.l = gtk.NewLabel("")
	g.l.SetJustify(gtk.JustifyCenter)
	g.l.SetHExpand(true)
	g.l.SetEllipsize(pango.EllipsizeEnd)
	g.l.SetLines(2)
	g.l.SetLineWrap(true)
	g.l.SetLineWrapMode(pango.WrapWordChar)

	g.Add(g.i)
	g.Add(g.l)

	return g
}

func (g *Greeter) SetPixbuf(p *gdkpixbuf.Pixbuf) {
	g.i.SetFromPixbuf(p)
}

func (g *Greeter) SetSurface(s *cairo.Surface) {
	g.i.SetFromSurface(s)
}

func (g *Greeter) SetIconName(iconName string) {
	g.i.SetFromIconName(iconName, 0)
}

func (g *Greeter) SetText(title, desc string) {
	g.l.SetMarkup(fmt.Sprintf(
		`<span size="large">%s</span>`+"\n"+`%s`,
		html.EscapeString(title), html.EscapeString(desc),
	))
}
