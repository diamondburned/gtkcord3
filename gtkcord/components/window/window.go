package window

import (
	"os"
	"os/signal"
	"runtime"

	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var Window struct {
	*gtk.Window
	Root   *gdk.Window
	Widget gtk.IWidget

	Header *Header

	CSS       *gtk.CssProvider
	Clipboard *gtk.Clipboard
	IconTheme *gtk.IconTheme

	Closer func()

	CursorDefault *gdk.Cursor
	CursorPointer *gdk.Cursor

	done chan struct{}
}

func Init() error {
	if Window.Window != nil {
		return nil
	}

	runtime.LockOSThread()
	gtk.Init(nil)

	Window.Closer = func() {}
	Window.done = make(chan struct{})

	d, err := gdk.DisplayGetDefault()
	if err != nil {
		return errors.Wrap(err, "Failed to get default GDK display")
	}
	s, err := d.GetDefaultScreen()
	if err != nil {
		return errors.Wrap(err, "Failed to get default screen")
	}

	root, err := s.GetRootWindow()
	if err != nil {
		return errors.Wrap(err, "Failed to get root window")
	}
	Window.Root = root

	if err := loadCSS(s); err != nil {
		return errors.Wrap(err, "Failed to load CSS")
	}

	if err := animations.LoadCSS(s); err != nil {
		return errors.Wrap(err, "Failed to load animations CSS")
	}

	w, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return errors.Wrap(err, "Failed to create window")
	}
	Window.Window = w

	w.Connect("destroy", func() {
		gtk.MainQuit()
		Window.Closer()
	})

	// w.SetVAlign(gtk.ALIGN_CENTER)
	// w.SetHAlign(gtk.ALIGN_CENTER)
	// w.SetDefaultSize(500, 250)

	c, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		return errors.Wrap(err, "Failed to get clipboard")
	}
	Window.Clipboard = c

	i, err := gtk.IconThemeGetDefault()
	if err != nil {
		return errors.Wrap(err, "Can't get Gtk icon theme")
	}
	Window.IconTheme = i

	if err := initHeader(); err != nil {
		return errors.Wrap(err, "Failed to make a headerbar")
	}

	Window.CursorDefault, err = gdk.CursorNewFromName(d, "default")
	if err != nil {
		return errors.Wrap(err, "Failed to create a default cursor")
	}
	Window.CursorPointer, err = gdk.CursorNewFromName(d, "pointer")
	if err != nil {
		return errors.Wrap(err, "Failed to create a pointer cursor")
	}

	go func() {
		runtime.LockOSThread()
		gtk.Main()

		close(Window.done)
	}()

	return nil
}

func Blur() {
	Window.SetSensitive(false)
}
func Unblur() {
	Window.SetSensitive(true)
}

func SetPointerCursor() {
	Window.Root.SetCursor(Window.CursorPointer)
}
func SetDefaultCursor() {
	Window.Root.SetCursor(Window.CursorDefault)
}

func Resize(w, h int) {
	Window.Window.Resize(w, h)
}

func Display(w gtk.IWidget) {
	if Window.Widget != nil {
		Window.Window.Remove(Window.Widget)
	}

	Window.Widget = w
	Window.Window.Add(w)
}

func ShowAll() {
	Window.Window.ShowAll()
}

func Wait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-Window.done:
	case <-sig:
	}
}
