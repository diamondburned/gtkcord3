package roundimage

/*
import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

// TODO: move roundimage.Button to rich.ImageButton.

// Button implements a rounded button with a rounded image. This widget only
// supports a full circle for rounding.
type Button struct {
	*gtk.Button
	Image Imager
}

var roundButtonCSS = primitives.PrepareClassCSS("round-button", `
	.round-button {
		padding: 0;
		border-radius: 50%;
	}
`)

func NewButton() (*Button, error) {
	image := NewImage(0)
	image.Show()

	b := NewEmptyButton()
	b.SetImage(image)

	return b, nil
}

func NewEmptyButton() *Button {
	b := gtk.ButtonNew()
	b.SetRelief(gtk.RELIEF_NONE)
	roundButtonCSS(b)

	return &Button{Button: b}
}

// NewCustomButton creates a new rounded button with the given Imager. If the
// given Imager implements the Connector interface (aka *StillImage), then the
// function will implicitly connect its handlers to the button.
func NewCustomButton(img Imager) (*Button, error) {
	b := NewEmptyButton()
	b.SetImage(img)

	if connector, ok := img.(Connector); ok {
		connector.ConnectHandlers(b)
	}

	return b, nil
}

func (b *Button) SetImage(img Imager) {
	b.Image = img
	b.Button.SetImage(img)
}
*/
