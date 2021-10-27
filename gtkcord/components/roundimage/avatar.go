package roundimage

/*
import (
	"context"

	"github.com/diamondburned/gotk4-handy"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

// TODO: GIF support

// TextSetter is an interface for setting texts.
type TextSetter interface {
	SetText(text string)
}

func TrySetText(imager Imager, text string) {
	if setter, ok := imager.(TextSetter); ok {
		setter.SetText(text)
	}
}

// Avatar is a static HdyAvatar container.
type Avatar struct {
	handy.Avatar
	pixbuf *gdk.Pixbuf
	url    string
	size   int
	cancel context.CancelFunc
}

// Make a better API that allows scaling.

var (
	_ Imager                  = (*Avatar)(nil)
	_ TextSetter              = (*Avatar)(nil)
	_ httputil.ImageContainer = (*Avatar)(nil)
)

func NewAvatar(size int) *Avatar {
	avatar := Avatar{
		Avatar: *handy.AvatarNew(size, "", true),
		size:   size,
	}
	// Set the load function. This should hopefully trigger a reload.
	avatar.SetImageLoadFunc(avatar.loadFunc)

	return &avatar
}

// GetSizeRequest returns the virtual size.
func (a *Avatar) GetSizeRequest() (int, int) {
	return a.size, a.size
}

// SetSizeRequest sets the avatar size. The actual size is min(w, h).
func (a *Avatar) SetSizeRequest(w, h int) {
	var min = w
	if w > h {
		min = h
	}

	a.size = min
	a.Avatar.SetSize(min)
	a.Avatar.SetSizeRequest(w, h)
}

func (a *Avatar) loadFunc(size int) *gdk.Pixbuf {
	if a.url == "" {
		return nil
	}

	if a.pixbuf != nil && a.size == size {
		return a.pixbuf
	}

	a.size = size
	a.refetch()

	return nil
}

func (a *Avatar) refetch() {
	a.cancelCtx()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	httputil.AsyncImage(ctx, a, a.url)
}

func (a *Avatar) cancelCtx() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}

// SetRadius is a no-op.
func (a *Avatar) SetRadius(float64) {}

// SetImageURL sets the avatar's source URL and reloads it asynchronously.
func (a *Avatar) SetImageURL(url string) {
	a.url = url
	a.refetch()
}

// SetFromPixbuf sets the pixbuf.
func (a *Avatar) SetFromPixbuf(pb *gdk.Pixbuf) {
	a.cancelCtx()
	a.pixbuf = pb
	// a.Avatar.SetImageLoadFunc(a.loadFunc)
	a.Avatar.QueueDraw()
}

// SetFromAnimation sets the first frame of the animation.
func (a *Avatar) SetFromAnimation(pa *gdk.PixbufAnimation) {
	a.cancelCtx()
	a.pixbuf = pa.GetStaticImage()
	// a.Avatar.SetImageLoadFunc(a.loadFunc)
	a.Avatar.QueueDraw()
}

// GetPixbuf returns the underlying pixbuf.
func (a *Avatar) GetPixbuf() *gdk.Pixbuf {
	return a.pixbuf
}

// GetAnimation returns nil.
func (a *Avatar) GetAnimation() *gdk.PixbufAnimation {
	return nil
}

// GetImage returns nil.
func (a *Avatar) GetImage() *gtk.Image {
	return nil
}

// GetStorageType always returns IMAGE_PIXBUF.
func (a *Avatar) GetStorageType() gtk.ImageType {
	return gtk.IMAGE_PIXBUF
}
*/
