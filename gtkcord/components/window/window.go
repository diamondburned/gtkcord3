package window

import (
	"github.com/diamondburned/gotk4-handy/pkg/handy"
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
	*handy.ApplicationWindow
	App   *gtk.Application
	Accel *gtk.AccelGroup

	Screen    *gdk.Screen
	Root      *gdk.Window
	Clipboard *gtk.Clipboard

	main    *gtk.Stack
	body    *gtk.Box
	header  gtk.Widgetter
	content gtk.Widgetter

	// CursorDefault *gdk.Cursor
	// CursorPointer *gdk.Cursor

	Settings *gtk.Settings

	// since files can be changed while the application is running:
	fileCSS *gtk.CSSProvider
}

func GDKWindow() gdk.Windower {
	return Window.ApplicationWindow.Window.Window()
}

func WithApplication(app *gtk.Application) error {
	if Window != nil {
		return nil
	}

	Window = &Container{App: app}

	w := handy.NewApplicationWindow()
	w.SetApplication(app)
	w.SetDefaultSize(850, 650)
	Window.ApplicationWindow = w

	Window.body = gtk.NewBox(gtk.OrientationVertical, 0)
	Window.body.SetHExpand(true)
	Window.body.SetVExpand(true)
	Window.ApplicationWindow.Add(Window.body)

	// Window.Header = gtk.NewBox(gtk.OrientationVertical, 0)

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
	Window.Root = gdk.BaseWindow(root)

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

	c := gtk.ClipboardGetDefault(d)
	Window.Clipboard = c

	// Make the main view: the stack.
	main := newStack()
	main.AddNamed(Window.body, "main")
	Window.main = main

	// Add the stack into the window:
	w.Add(main)

	// Play the loading animation:
	NowLoading()

	return nil
}

func SetHeader(h gtk.Widgetter) {
	if Window.header != nil {
		Window.body.Remove(Window.header)
	}
	Window.header = h
	if h != nil {
		Window.body.PackStart(h, false, false, 0)
	}
}

func HeaderShowAll() {
	w := Window.Titlebar()
	gtk.BaseWidget(w).ShowAll()
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
	if previousLoadingChild != nil {
		SetHeader(previousLoadingChild)
		previousLoadingChild = nil
	}

	if Window.content != nil {
		Window.body.Remove(Window.content)
	}

	Window.content = w
	Window.body.PackEnd(Window.content, true, true, 0)

	Window.main.SetVisibleChildName("main")
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
