package avatar

import (
	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/internal/log"
)

// Image decribes a basic avatar image.
type Image struct {
	gtk.Widget
	Image *roundimage.Image
	// animation
	// *gdkpixbuf.Pixbuf
	// *cairo.Surface
	data interface{}

	size    int
	hovered bool
}

type animation struct {
	anim  *gdkpixbuf.PixbufAnimation
	frame *cairo.Surface
}

func convertAnimation(anim *gdkpixbuf.PixbufAnimation) animation {
	frame := anim.StaticImage()
	return animation{
		anim:  anim,
		frame: gdk.CairoSurfaceCreateFromPixbuf(frame, 1, window.GDKWindow()),
	}
}

// New creates a new Image instance.
func New(size int) *Image {
	return newImage(roundimage.NewImage(0), size, true)
}

// NewUnwrapped creates a new image without wrapping it in an eventbox.
func NewUnwrapped(size int) *Image {
	return newImage(roundimage.NewImage(0), size, false)
}

func newImage(image *roundimage.Image, size int, wrap bool) *Image {
	img := &Image{}
	img.size = size

	img.Image = image
	img.Image.SetVAlign(gtk.AlignCenter)
	img.Image.SetHAlign(gtk.AlignCenter)
	img.Image.SetPixelSize(size)
	img.Image.SetSizeRequest(size, size)

	if wrap {
		evbox := gtk.NewEventBox()
		evbox.SetVAlign(gtk.AlignCenter)
		evbox.SetHAlign(gtk.AlignCenter)
		evbox.AddEvents(int(0 |
			gdk.EnterNotifyMask |
			gdk.LeaveNotifyMask,
		))
		evbox.Connect("enter-notify-event", func() { img.SetPlayAnimation(true) })
		evbox.Connect("leave-notify-event", func() { img.SetPlayAnimation(false) })
		evbox.Add(img.Image)
		img.Image.Show()
		img.Widget = *gtk.BaseWidget(evbox)
	} else {
		img.Widget = *gtk.BaseWidget(img.Image)
	}

	return img
}

func (img *Image) Size() int {
	return img.size
}

func (img *Image) SetInitials(initials string) {
	img.Image.SetInitials(initials)
}

// SetURL makes the image display a URL.
func (img *Image) SetURL(url string) {
	cache.SetImageURLScaled(img, url, img.size, img.size)
}

// SetFromIconName sets the icon name.
func (img *Image) SetFromIconName(icon string, _ int) {
	img.Image.SetFromIconName(icon, 0)
}

// SetFromPixbuf sets a pixbuf into the image.
func (img *Image) SetFromPixbuf(pixbuf *gdkpixbuf.Pixbuf) {
	img.setData(pixbuf)
}

func (img *Image) SetFromAnimation(anim *gdkpixbuf.PixbufAnimation) {
	img.setData(convertAnimation(anim))
}

// SetFromAnimationWithSurface sets an animation with a supplemental Cairo
// surface of the static frame scaled up. This helps achieve a consistent look
// on HiDPI devices without sacrificing GIFs.
func (im *Image) SetFromAnimationWithSurface(anim *gdkpixbuf.PixbufAnimation, s *cairo.Surface) {
	panic("unimplemented")
}

// SetFromSurface sets
func (img *Image) SetFromSurface(surface *cairo.Surface) {
	img.setData(surface)
}

func (img *Image) setData(data interface{}) {
	img.data = data
	img.updateImage()
}

// SetPlayAnimation sets whether or not the image should be playing the
// animation.
func (img *Image) SetPlayAnimation(play bool) {
	img.hovered = play
	img.updateImage()
}

func (img *Image) isAnimation() bool {
	_, ok := img.data.(animation)
	return ok
}

func (img *Image) updateImage() {
	switch data := img.data.(type) {
	case nil:
		// nothing
	case animation:
		if img.hovered {
			img.Image.SetFromAnimation(data.anim)
		} else {
			img.Image.SetFromSurface(data.frame)
		}
	case *gdkpixbuf.Pixbuf:
		img.Image.SetFromPixbuf(data)
	case *cairo.Surface:
		img.Image.SetFromSurface(data)
	default:
		log.Panicf("avatar: unknown data type %T", img.data)
	}
}
