package window

import (
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Header struct {
	*handy.TitleBar
	Main *gtk.Stack
}

func initHeader() error {
	h := handy.TitleBarNew()
	h.Show()

	// // empty box for 0 width
	// b, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	// if err != nil {
	// 	return errors.Wrap(err, "Failed to create an empty box")
	// }
	// h.SetCustomTitle(b)

	// Main stack
	s, err := newStack()
	if err != nil {
		return errors.Wrap(err, "Failed to create main stack")
	}

	Window.Header = &Header{
		TitleBar: h,
		Main:     s,
	}
	h.Add(s)
	Window.Window.SetTitlebar(h)

	return nil
}

func HeaderDisplay(w gtkutils.ExtendedWidget) {
	// Check if loading:
	if Window.Header.Main.GetVisibleChildName() == "loading" {
		// Remove the loading screen last:
		defer stackRemove(Window.Header.Main, "loading")
	}
	stackSet(Window.Header.Main, "main", w)
}

func HeaderShowAll() {
	Window.Header.ShowAll()
}
