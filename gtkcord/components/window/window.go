package window

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var Window *Container

type Container struct {
	*gtk.ApplicationWindow
	Accel *gtk.AccelGroup

	Root   *gdk.Window
	Widget gtk.IWidget

	Header *Header

	CSS       *gtk.CssProvider
	Clipboard *gtk.Clipboard

	CursorDefault *gdk.Cursor
	CursorPointer *gdk.Cursor

	Settings *gtk.Settings
}

func WithApplication(app *gtk.Application) error {
	if Window != nil {
		return nil
	}

	Window = &Container{}

	w, err := gtk.ApplicationWindowNew(app)
	if err != nil {
		return errors.Wrap(err, "Failed to create window")
	}
	Window.ApplicationWindow = w

	w.Show()
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

func SetTitle(title string) {
	Window.ApplicationWindow.SetTitle(title)
}
