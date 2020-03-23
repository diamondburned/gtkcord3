package window

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/handy"
)

type Header struct {
	*handy.TitleBar
	Widget gtkutils.ExtendedWidget
}

func initHeader() error {
	h := handy.TitleBarNew()

	// // empty box for 0 width
	// b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	// if err != nil {
	// 	return errors.Wrap(err, "Failed to create an empty box")
	// }
	// h.SetCustomTitle(b)

	Window.Header = &Header{
		TitleBar: h,
	}
	Window.Window.SetTitlebar(h)

	return nil
}

func HeaderDisplay(w gtkutils.ExtendedWidget) {
	if Window.Header.Widget != nil {
		Window.Header.TitleBar.Remove(Window.Header.Widget)
	}

	Window.Header.Widget = w

	if w == nil {
		return
	}

	Window.Header.TitleBar.Add(w)
}

func HeaderShowAll() {
	Window.Header.ShowAll()
}
