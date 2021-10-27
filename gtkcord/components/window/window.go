package window

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
	"github.com/pkg/errors"
)

const SwitchFade = 200 // 200ms to fade between main view and loading screen.

func newStack() *gtk.Stack {
	s := gtk.NewStack()
	s.SetTransitionDuration(SwitchFade)
	s.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	s.Show()
	return s
}

func stackRemove(s *gtk.Stack, name string) {
	if w := s.ChildByName(name); w != nil {
		s.Remove(w)
	}
}

func stackSet(s *gtk.Stack, name string, w gtk.Widgetter) {
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

	Main   *gtk.Stack
	header gtk.Widgetter

	// CursorDefault *gdk.Cursor
	// CursorPointer *gdk.Cursor

	Settings *gtk.Settings

	// since files can be changed while the application is running:
	fileCSS *gtk.CSSProvider
}

func WithApplication(app *gtk.Application) error {
	if Window != nil {
		return nil
	}

	Window = &Container{App: app}

	// w := handy.NewApplicationWindow()
	// w.SetApplication(app)
	w := gtk.NewApplicationWindow(app)
	Window.ApplicationWindow = w

	l := logo.Pixbuf(64)
	w.SetIcon(l)
	w.Connect("destroy", app.Quit)
	w.Show()

	a := gtk.NewAccelGroup()
	Window.Accel = a
	w.AddAccelGroup(a)

	d := gdk.DisplayGetDefault()
	s := d.DefaultScreen()
	Window.Screen = s

	root := s.RootWindow()
	Window.Root = root.BaseWindow()

	// Load CSS for the first time.
	initCSS()

	if err := animations.LoadCSS(s); err != nil {
		return errors.Wrap(err, "Failed to load animations CSS")
	}

	settings := gtk.SettingsGetDefault()
	Window.Settings = settings
	overrideSettings(settings)

	// w.SetVAlign(gtk.AlignCenter)
	// w.SetHAlign(gtk.AlignCenter)
	// w.SetDefaultSize(500, 250)

	c := gtk.ClipboardGetDefault(d)
	Window.Clipboard = c

	// Make the main view: the stack.
	main := newStack()
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

func SetHeader(h gtk.Widgetter) {
	Window.header = h
	Window.SetTitlebar(h)
}

func HeaderShowAll() {
	w := Window.Titlebar()
	w.BaseWidget().ShowAll()
}

// Destroy closes the application as well.
func Destroy() {
	Window.Window.Close()
}

// Blur disables the window.
func Blur() {
	Window.SetSensitive(false)
}

// Unblur enables the window.
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

func Display(w gtk.Widgetter) {
	if wasLoading && Window.header != nil {
		Window.SetTitlebar(Window.header)
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
	if title == "" {
		title = "gtkcord3"
	} else {
		title += " — gtkcord3"
	}

	Window.ApplicationWindow.SetTitle(title)
}
