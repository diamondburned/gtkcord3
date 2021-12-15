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

// func stackRemove(s *gtk.Stack, name string) {
// 	if w := s.ChildByName(name); w != nil {
// 		s.Remove(w)
// 	}
// }

// func stackSet(s *gtk.Stack, name string, w gtk.Widgetter) {
// 	stackRemove(s, name)
// 	s.AddNamed(w, name)
// 	s.SetVisibleChildName(name)
// }

var Window *Container

type Container struct {
	*handy.ApplicationWindow
	App   *gtk.Application
	Accel *gtk.AccelGroup

	Screen    *gdk.Screen
	Root      *gdk.Window
	Clipboard *gtk.Clipboard

	main  *gtk.Stack
	pages map[string]*Page

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

	Window = &Container{
		App:   app,
		pages: make(map[string]*Page),
	}

	w := handy.NewApplicationWindow()
	w.SetApplication(app)
	w.SetDefaultSize(850, 650)
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
	Window.main = main

	// Add the stack into the window:
	w.Add(main)

	// Play the loading animation:
	NowLoading()

	return nil
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

type Page struct {
	body    *gtk.Box
	header  gtk.Widgetter
	content gtk.Widgetter
}

func SwitchToPage(name string) *Page {
	p, ok := Window.pages[name]
	if ok {
		Window.main.SetVisibleChild(p.body)
		return p
	}

	p = &Page{}
	p.body = gtk.NewBox(gtk.OrientationVertical, 0)
	p.body.SetHExpand(true)
	p.body.SetVExpand(true)
	p.body.Show()

	Window.main.AddNamed(p.body, name)
	Window.main.SetVisibleChild(p.body)
	Window.pages[name] = p

	return p
}

func (p *Page) SetHeader(h gtk.Widgetter) {
	if p.header != nil {
		p.body.Remove(p.header)
	}
	p.header = h
	if h != nil {
		p.body.PackStart(h, false, false, 0)
	}
}

func (p *Page) SetChild(w gtk.Widgetter) {
	if p.content != nil {
		p.body.Remove(p.content)
	}
	p.content = w
	if w != nil {
		p.body.PackEnd(w, true, true, 0)
	}
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
