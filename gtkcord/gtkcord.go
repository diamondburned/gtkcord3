package gtkcord

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
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
	Header *Header
	Grid   *gtk.Grid

	// Dynamic sidebars and main pages
	Sidebar  gtk.IWidget
	Messages gtk.IWidget

	// Stuff
	Guilds *Guilds

	// nil after finalize()
	sbox      *gtk.Box
	spinner   *gtk.Spinner
	iconTheme *gtk.IconTheme

	busy sync.Mutex

	css *gtk.CssProvider

	parser *md.Parser

	done chan struct{}
}

func New() (*Application, error) {
	a := new(Application)
	a.done = make(chan struct{})

	if err := a.init(); err != nil {
		return nil, errors.Wrap(err, "Failed to start Gtk")
	}

	// Things beyond this point must use must() or gdk.IdleAdd.
	return a, nil
}

func (a *Application) UseState(s *state.State) error {
	a.State = s
	a.Window.Remove(a.sbox)
	a.parser = md.NewParser(s)

	if err := a.Header.Hamburger.Refresh(s); err != nil {
		return errors.Wrap(err, "Failed to refresh hamburger")
	}

	{
		gw, err := gtk.ScrolledWindowNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to make guilds scrollbar")
		}
		gw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)

		gs, err := newGuilds(s, a.loadGuild)
		if err != nil {
			return errors.Wrap(err, "Failed to make guilds view")
		}
		a.Guilds = gs

		must(gw.Add, gs.ListBox)
		must(a.Grid.Add, gw)

		s, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
		if err != nil {
			return errors.Wrap(err, "Failed to create separator")
		}
		must(a.Grid.Add, s)
	}

	// Finalize the window:
	a.finalize()

	// 100 goroutines is pretty cheap (lol)
	for _, g := range a.Guilds.Guilds {
		_, err := s.Channels(g.ID)
		if err != nil {
			logWrap(err, "Failed to pre-fetch channels")
		}

		if g.Folder != nil {
			for _, g := range g.Folder.Guilds {
				_, err := s.Channels(g.ID)
				if err != nil {
					logWrap(err, "Failed to pre-fetch channels")
				}
			}
		}
	}

	// s.AddHandler(func(m *gateway.MessageCreateEvent) {

	// })

	a.wait()

	return nil
}

func (a *Application) init() error {
	gtk.Init(nil)

	if err := a.loadCSS(); err != nil {
		return errors.Wrap(err, "Failed to load CSS")
	}

	w, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return errors.Wrap(err, "Failed to create window")
	}
	w.Connect("destroy", func() {
		a.close()
		gtk.MainQuit()
		close(a.done)
	})
	w.SetDefaultSize(1000, 750)
	a.Window = w

	h, err := newHeader()
	if err != nil {
		return errors.Wrap(err, "Failed to create header")
	}
	w.SetTitlebar(h)
	a.Header = h

	i, err := gtk.IconThemeGetDefault()
	if err != nil {
		return errors.Wrap(err, "Can't get Gtk icon theme")
	}
	a.iconTheme = i

	g, err := gtk.GridNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create grid")
	}
	g.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
	g.SetRowHomogeneous(true)
	a.Grid = g

	// Instead of adding the above grid, we should add the spinning circle.
	sbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return errors.Wrap(err, "Failed to create spinner box")
	}
	sbox.SetSizeRequest(50, 50)
	sbox.SetVAlign(gtk.ALIGN_CENTER)
	w.Add(sbox)
	a.sbox = sbox

	s, err := gtk.SpinnerNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create spinner")
	}
	s.SetSizeRequest(50, 50)
	s.Start()
	sbox.Add(s)
	a.spinner = s

	w.ShowAll()
	go gtk.Main()

	return nil
}

func (a *Application) finalize() {
	must(a.Window.Remove, a.spinner)
	must(a.Window.Add, a.Grid)
	must(a.Window.ShowAll)
	a.spinner.Stop()
	a.sbox.SetSizeRequest(ChannelsWidth, -1)
}

func (a *Application) close() {
	if err := a.State.Close(); err != nil {
		logError(errors.Wrap(err, "Failed to close Discord"))
	}
}

func (a *Application) setChannelCol(w gtk.IWidget) {
	a.Sidebar = w
	a.Grid.Attach(w, 2, 0, 1, 1)
}
func (a *Application) setMessageCol(w gtk.IWidget) {
	a.Messages = w
	a.Grid.Attach(w, 4, 0, 1, 1)
}

func (a *Application) loadGuild(g *Guild) {
	if a.Sidebar != nil {
		a.Grid.Remove(a.Sidebar)
	}

	// Start the busy mutex early on, so we could use the shared spinner.
	a.busy.Lock()
	a.spinner.Start()
	a.setChannelCol(a.sbox)

	go func() {
		a._loadGuild(g)
		a.loadChannel(g, g.Current())
	}()
}

func (a *Application) _loadGuild(g *Guild) {
	defer a.busy.Unlock()

	dg, err := a.State.Guild(g.ID)
	if err != nil {
		logWrap(err, "Failed to get guild")
		return
	}

	if err := g.loadChannels(a.State, *dg, a.loadChannel); err != nil {
		logWrap(err, "Failed to load channels")
		return
	}

	must(a.spinner.Stop)
	must(a.Grid.Remove, a.sbox)
	must(a.setChannelCol, g.Channels.IWidget)

	if a.Sidebar == nil {
		s, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
		if err != nil {
			logWrap(err, "Failed to make a separator")
		} else {
			must(a.Grid.Add, s)
		}
	}

	must(a.Grid.ShowAll)
	a.Header.hookGuild(dg)
}

func (a *Application) loadChannel(g *Guild, ch *Channel) {
	if a.Messages != nil {
		a.Grid.Remove(a.Messages)
	}

	// Start the busy mutex early on, so we could use the shared spinner.
	a.busy.Lock()
	a.spinner.Start()
	a.setMessageCol(a.sbox)

	go a._loadChannel(g, ch)
}

func (a *Application) _loadChannel(g *Guild, ch *Channel) {
	defer a.busy.Unlock()

	if a.Messages != nil {
		a.Grid.Remove(a.Messages)
	}

	dch, err := a.State.Channel(ch.ID)
	if err != nil {
		logWrap(err, "Failed to load channel "+ch.ID.String())
		return
	}

	// Run hook
	a.Header.hookChannel(dch)

	if err := g.GoTo(a.State, a.parser, ch); err != nil {
		logWrap(err, "Failed to go to channel")
		return
	}

	must(a.setMessageCol, ch.Messages)
	must(a.Grid.ShowAll)
	go func() {
		// Workaround for Gtk being crappy:
		time.Sleep(50 * time.Millisecond)
		must(ch.Messages.SmartScroll)
	}()
}

func (a *Application) wait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-sig:
	case <-a.done:
	}
}
