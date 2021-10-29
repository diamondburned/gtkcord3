package roundimage

import (
	"context"
	"math"

	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

const (
	pi     = math.Pi
	circle = 2 * math.Pi
)

// Image represents an image with abstractions for asynchronously fetching
// images from a URL as well as having interchangeable fallbacks.
type Image struct {
	gtk.Image
	Radius        float64
	initials      string
	initialsDrawn bool
	useInitials   bool
}

// NewImage creates a new round image. If radius is 0, then it will be half the
// dimensions. If the radius is less than 0, then nothing is rounded.
func NewImage(radius float64) *Image {
	image := &Image{
		Image:  *gtk.NewImage(),
		Radius: radius,
	}

	// Connect to the draw callback and clip the context.
	image.Connect("draw", image.draw)

	return image
}

// SetRadius sets the radius to be drawn with. If 0 is given, then a full circle
// is drawn, which only works best for images guaranteed to be square.
// Otherwise, the radius is either the number given or the minimum of either the
// width or height.
func (i *Image) SetRadius(r float64) {
	i.Radius = r
	i.QueueDraw()
}

// Clip clips the image from the Cairo context.
func (i *Image) Clip(cc *cairo.Context) {
	i.draw(&i.Image, cc)
}

func (i *Image) draw(image *gtk.Image, cc *cairo.Context) {
	// Draw the initials if we haven't already.
	if i.useInitials && !i.initialsDrawn {
		rect := image.Allocation()
		i.drawInitials(rect.Width(), rect.Height())
		return
	}

	if i.StorageType() == gtk.ImageIconName {
		// Don't round if we're displaying a stock icon.
		return
	}

	rect := image.Allocation()
	w := float64(rect.Width())
	h := float64(rect.Height())

	min := w
	// Use the largest side for radius calculation.
	if h > w {
		min = h
	}

	switch {
	// If radius is less than 0, then don't round.
	case i.Radius < 0:
		return

	// If radius is 0, then we have to calculate our own radius.:This only
	// works if the image is a square.
	case i.Radius == 0:
		// Calculate the radius by dividing a side by 2.
		r := (min / 2)

		// Draw an arc from 0deg to 360deg.
		cc.Arc(w/2, h/2, r, 0, circle)

		// We have to do this so the arc paint doesn't leave back a black
		// background instead of the usual alpha.
		cc.SetSourceRGBA(255, 255, 255, 0)

		// Clip the image with the arc we drew.
		cc.Clip()

	// If radius is more than 0, then we have to calculate the radius from
	// the edges.
	case i.Radius > 0:
		// StackOverflow is godly.
		// https://stackoverflow.com/a/6959843.

		// Copy the variables so we can change them later.
		r := i.Radius

		// Radius should be largest a single side divided by 2.
		if max := min / 2; r > max {
			r = max
		}

		// Draw 4 arcs at 4 corners.
		cc.Arc(0+r, 0+r, r, 2*(pi/2), 3*(pi/2)) // top left
		cc.Arc(w-r, 0+r, r, 3*(pi/2), 4*(pi/2)) // top right
		cc.Arc(w-r, h-r, r, 0*(pi/2), 1*(pi/2)) // bottom right
		cc.Arc(0+r, h-r, r, 1*(pi/2), 2*(pi/2)) // bottom left

		// Close the created path.
		cc.ClosePath()
		cc.SetSourceRGBA(255, 255, 255, 0)

		// Clip the image with the arc we drew.
		cc.Clip()
	}

	// Paint the changes.
	cc.Paint()
}

func (i *Image) Clear() {
	i.SetFromPixbuf(nil)
}

func (i *Image) SetFromPixbuf(p *gdkpixbuf.Pixbuf) {
	i.Image.SetFromPixbuf(p)
	i.initialsDrawn = false
	i.useInitials = p == nil
}

func (i *Image) SetFromAnimation(p *gdkpixbuf.PixbufAnimation) {
	i.Image.SetFromAnimation(p)
	i.initialsDrawn = false
	i.useInitials = p == nil
}

func (i *Image) SetFromSurface(s *cairo.Surface) {
	i.Image.SetFromSurface(s)
	i.initialsDrawn = false
	i.useInitials = s == nil
}

func (i *Image) Initials() string {
	return i.initials
}

func (i *Image) SetInitials(initials string) {
	i.initials = initials

	if !i.useInitials {
		switch i.StorageType() {
		case gtk.ImageEmpty, gtk.ImageIconName:
			i.useInitials = true
		default:
			// Initials already drawn; we can override that.
			i.useInitials = i.initialsDrawn
		}
	}

	i.initialsDrawn = false
}

func (i *Image) drawInitials(w, h int) {
	i.initialsDrawn = true
	scale := i.ScaleFactor()

	var size int
	if w > h {
		size = h
	} else {
		size = w
	}

	a := handy.NewAvatar(size, i.initials, true)
	a.DrawToPixbufAsync(context.Background(), size, scale, func(fin gio.AsyncResulter) {
		p := a.DrawToPixbufFinish(fin)
		s := gdk.CairoSurfaceCreateFromPixbuf(p, scale, nil)
		i.Image.SetFromSurface(s)
	})
}
