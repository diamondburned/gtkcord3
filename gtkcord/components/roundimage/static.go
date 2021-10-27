package roundimage

/*
import (
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
)

// StillImage is an image that only plays a GIF if it's hovered on top of.
type StillImage struct {
	*Image
	animating bool
	animation *gdk.PixbufAnimation
}

var (
	_ Imager                  = (*StillImage)(nil)
	_ Connector               = (*StillImage)(nil)
	_ httputil.ImageContainer = (*StillImage)(nil)
)

// NewStillImage creates a new static that  binds to the parent's handler so
// that the image only animates when parent is hovered over.
func NewStillImage(parent primitives.Connector, radius float64) *StillImage {
	i := NewImage(radius)

	s := StillImage{i, false, nil}
	s.ConnectHandlers(parent)

	return &s
}

func (s *StillImage) ConnectHandlers(connector primitives.Connector) {
	connector.Connect("enter-notify-event", func() {
		if s.animation != nil && !s.animating {
			s.animating = true
			s.Image.SetFromAnimation(s.animation)
		}
	})
	connector.Connect("leave-notify-event", func() {
		if s.animation != nil && s.animating {
			s.animating = false
			s.Image.SetFromPixbuf(s.animation.GetStaticImage())
		}
	})
}

// SetImageURL sets the image's URL.
func (s *StillImage) SetImageURL(url string) {
	s.Image.SetImageURLInto(url, s)
}

func (s *StillImage) SetFromPixbuf(pb *gdk.Pixbuf) {
	s.animation = nil
	s.Image.SetFromPixbuf(pb)
}

func (s *StillImage) SetFromSurface(sf *cairo.Surface) {
	s.animation = nil
	s.Image.SetFromSurface(sf)
}

func (s *StillImage) SetFromAnimation(anim *gdk.PixbufAnimation) {
	s.animation = anim
	s.Image.SetFromPixbuf(anim.GetStaticImage())
}

func (s *StillImage) GetAnimation() *gdk.PixbufAnimation {
	return s.animation
}
*/
