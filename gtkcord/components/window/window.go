package window

import (
	"os"
	"os/signal"
	"runtime"

	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var Window struct {
	*gtk.Window
	Accel *gtk.AccelGroup

	Root   *gdk.Window
	Widget gtk.IWidget

	Header *Header

	CSS       *gtk.CssProvider
	Clipboard *gtk.Clipboard

	Closer func()

	CursorDefault *gdk.Cursor
	CursorPointer *gdk.Cursor

	Settings *gtk.Settings

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

	settings, err := gtk.SettingsGetDefault()
	if err != nil {
		return errors.Wrap(err, "Failed to get settings")
	}
	overrideSettings(settings)
	Window.Settings = settings

	w, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return errors.Wrap(err, "Failed to create window")
	}
	Window.Window = w

	w.Connect("destroy", func() {
		gtk.MainQuit()
	})

	l, err := logo.Pixbuf(64)
	if err != nil {
		return errors.Wrap(err, "Failed to load logo")
	}
	w.SetIcon(l)

	a, err := gtk.AccelGroupNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create accel group")
	}
	Window.Accel = a
	w.AddAccelGroup(a)

	// w.SetVAlign(gtk.ALIGN_CENTER)
	// w.SetHAlign(gtk.ALIGN_CENTER)
	// w.SetDefaultSize(500, 250)

	c, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	if err != nil {
		return errors.Wrap(err, "Failed to get clipboard")
	}
	Window.Clipboard = c

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

		Window.Closer()
		Window.done <- struct{}{}
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

func Show() {
	Window.Window.Show()
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
