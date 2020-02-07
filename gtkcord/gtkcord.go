package gtkcord

import (
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/state"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var (
	HTTPClient = http.Client{
		Timeout: 10 * time.Second,
	}
)

type Application struct {
	State  *state.State
	Window *gtk.Window
	Grid   *gtk.Grid

	// nil after finalize()
	spinner   *gtk.Spinner
	iconTheme *gtk.IconTheme
	css       *gtk.CssProvider
}

func New() (*Application, error) {
	var a = new(Application)

	if err := a.init(); err != nil {
		return nil, errors.Wrap(err, "Failed to start Gtk")
	}

	// Things beyond this point must use must() or gdk.IdleAdd.
	return a, nil
}

func (a *Application) UseState(s *state.State) error {
	a.State = s

	gs, err := a.newGuilds(s)
	if err != nil {
		return errors.Wrap(err, "Failed to make guilds view")
	}
	must(a.Grid.Add, gs.TreeView)

	// Finalize the window:
	a.finalize()

	// I wonder if you really need to do this:
	gtk.Main()

	return nil
}

func (a *Application) init() error {
	gtk.Init(nil)

	w, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return errors.Wrap(err, "Failed to create window")
	}
	w.Connect("destroy", func() {
		a.close()
		gtk.MainQuit()
	})
	a.Window = w

	i, err := gtk.IconThemeGetDefault()
	if err != nil {
		return errors.Wrap(err, "Can't get Gtk icon theme")
	}
	a.iconTheme = i

	g, err := gtk.GridNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create grid")
	}
	g.SetOrientation(gtk.ORIENTATION_VERTICAL)
	a.Grid = g

	// Instead of adding the above grid, we should add the spinning circle.
	s, err := gtk.SpinnerNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create spinner")
	}
	s.Start()
	a.spinner = s
	w.Add(a.spinner)
	w.ShowAll()

	return nil
}

func (a *Application) finalize() {
	must(a.Window.Remove, a.spinner)
	must(a.Window.Add, a.Grid)
	must(a.Window.ShowAll)
	a.spinner = nil
}

func (a *Application) close() {
	if err := a.State.Close(); err != nil {
		logError(errors.Wrap(err, "Failed to close Discord"))
	}
}
