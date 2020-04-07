package window

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const SwitchFade = 200 // 200ms to fade between main view and loading screen.

func newStack() (*gtk.Stack, error) {
	s, err := gtk.StackNew()
	if err != nil {
		return nil, err
	}
	s.Show()
	s.SetTransitionDuration(SwitchFade)
	s.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	return s, nil
}

func stackRemove(s *gtk.Stack, name string) {
	if w := s.GetChildByName(name); w != nil {
		s.Remove(w)
	}
}

func stackSet(s *gtk.Stack, name string, w gtk.IWidget) {
	stackRemove(s, name)
	s.AddNamed(w, name)
	s.SetVisibleChildName(name)
}

var Window *Container

type Container struct {
	*gtk.ApplicationWindow
	App   *gtk.Application
	Accel *gtk.AccelGroup

	Screen    *gdk.Screen
	Root      *gdk.Window
	Clipboard *gtk.Clipboard

	Header *Header
	Main   *gtk.Stack

	// CursorDefault *gdk.Cursor
	// CursorPointer *gdk.Cursor

	Settings *gtk.Settings

	// since files can be changed while the application is running:
	fileCSS *gtk.CssProvider
}

func WithApplication(app *gtk.Application) error {
	if Window != nil {
		return nil
	}

	Window = &Container{App: app}

	w, err := gtk.ApplicationWindowNew(app)
	if err != nil {
		return errors.Wrap(err, "Failed to create window")
	}
	Window.ApplicationWindow = w

	l, err := logo.Pixbuf(64)
	if err != nil {
		return errors.Wrap(err, "Failed to load logo")
	}
	w.SetIcon(l)

	w.Show()
	w.Connect("destroy", app.Quit)

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
	Window.Screen = s

	root, err := s.GetRootWindow()
	if err != nil {
		return errors.Wrap(err, "Failed to get root window")
	}
	Window.Root = root

	// Load CSS for the first time.
	initCSS()

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

	// Make the main view: the stack.
	main, err := newStack()
	if err != nil {
		return errors.Wrap(err, "Failed to make stack")
	}
	Window.Main = main

	// Add the stack into the window:
	w.Add(main)

	// Play the loading animation:
	NowLoading()

	// Window.CursorDefault, err = gdk.CursorNewFromName(d, "default")
	// if err != nil {
	// 	return errors.Wrap(err, "Failed to create a default cursor")
	// }
	// Window.CursorPointer, err = gdk.CursorNewFromName(d, "pointer")
	// if err != nil {
	// 	return errors.Wrap(err, "Failed to create a pointer cursor")
	// }

	return nil
}

func Notify(id string, notification *glib.Notification) {
	Window.App.SendNotification(id, notification)
}

// Destroy closes the application as well.
func Destroy() {
	Window.Window.Destroy()
}

func Blur() {
	Window.SetSensitive(false)
}
func Unblur() {
	Window.SetSensitive(true)
}

// func SetPointerCursor() {
// 	Window.Root.SetCursor(Window.CursorPointer)
// }
// func SetDefaultCursor() {
// 	Window.Root.SetCursor(Window.CursorDefault)
// }

func Resize(w, h int) {
	Window.Window.Resize(w, h)
}

func Display(w gtk.IWidget) {
	// Check if loading:
	if Window.Main.GetVisibleChildName() == "loading" {
		defer stackRemove(Window.Main, "loading")
	}
	stackSet(Window.Main, "main", w)
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
