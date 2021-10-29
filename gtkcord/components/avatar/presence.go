package avatar

import (
	"math"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

const (
	statusSizeBorder    = 0.35
	statusSizeMaxRadius = 9.0
)

// WithStatus is the user avatar with the activity status dot.
type WithStatus struct {
	*Image
	status gateway.Status

	size int
}

// NewWithStatus creates a new image with status.
func NewWithStatus(size int) *WithStatus {
	return NewFromImageWithStatus(New(size), size)
}

// NewFromImageWithStatus creates a new image with status from a custom image.
func NewFromImageWithStatus(image *Image, size int) *WithStatus {
	s := &WithStatus{
		Image:  image,
		status: "",
		size:   size,
	}
	image.ConnectAfter("draw", s.draw)
	return s
}

// SetStatus sets the status indicator.
func (s *WithStatus) SetStatus(status gateway.Status) {
	s.status = status
	gtk.BaseWidget(s.Image).QueueDraw()
}

func (s *WithStatus) draw(w gtk.Widgetter, cc *cairo.Context) {
	var color uint32

	switch s.status {
	case gateway.OnlineStatus:
		color = 0x43B581
	case gateway.DoNotDisturbStatus:
		color = 0xF04747
	case gateway.IdleStatus:
		color = 0xFAA61A
	case gateway.OfflineStatus, gateway.InvisibleStatus:
		color = 0x747F8D
	default:
		return
	}

	alloc := gtk.BaseWidget(w).Allocation()
	width := float64(alloc.Width())
	height := float64(alloc.Height())

	indicatorDiameter := math.Pow(float64(s.size), 0.75)
	indicatorRadius := math.Min(indicatorDiameter/2, statusSizeMaxRadius)

	borderThickness := indicatorRadius * statusSizeBorder
	borderStart := indicatorRadius - borderThickness

	centerPointX := width - indicatorRadius
	centerPointY := height - indicatorRadius

	cc.Arc(centerPointX, centerPointY, indicatorRadius, 0, 2*math.Pi)
	cc.Clip()
	cc.Paint()

	// Draw the indicator.
	cc.SetSourceRGB(hexRGB(color))
	cc.Arc(centerPointX, centerPointY, borderStart, 0, 2*math.Pi)
	cc.Fill()

	cc.SetSourceRGBA(1, 1, 1, 0)
	cc.Paint()
	return
}

const (
	maskR = 0xFF0000
	maskG = 0x00FF00
	maskB = 0x0000FF
)

func hexRGB(hex uint32) (r, g, b float64) {
	r = float64((hex&maskR)>>16) / 0xFF
	g = float64((hex&maskG)>>8) / 0xFF
	b = float64((hex&maskB)>>0) / 0xFF
	return
}
