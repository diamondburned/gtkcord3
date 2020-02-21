package window

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Header struct {
	*gtk.HeaderBar
	Widget gtkutils.ExtendedWidget
}

func initHeader() error {
	h, err := gtk.HeaderBarNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create headerbar")
	}
	h.SetShowCloseButton(true)

	// empty box
	b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return errors.Wrap(err, "Failed to create an empty box")
	}
	h.SetCustomTitle(b)

	Window.Header = &Header{
		HeaderBar: h,
	}
	Window.Window.SetTitlebar(h)

	return nil
}

func HeaderDisplay(w gtkutils.ExtendedWidget) {
	if Window.Header.Widget != nil {
		Window.Header.HeaderBar.Remove(Window.Header.Widget)
	}

	Window.Header.Widget = w
	Window.Header.HeaderBar.PackStart(w)
}

// func HeaderCenter(w gtkutils.ExtendedWidget) {
// 	Window.Header.Widget = w
// 	Window.Header.HeaderBar.SetCustomTitle(w)
// }

func HeaderShowAll() {
	Window.Header.ShowAll()
}
