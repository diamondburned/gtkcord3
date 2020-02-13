package gtkcord

import (
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var (
	HTTPClient = http.Client{
		Timeout: 10 * time.Second,
	}

	App *application
)

func init() {
	runtime.LockOSThread()
	runtime.GOMAXPROCS(1)
}

type ExtendedWidget interface {
	gtk.IWidget
	SetSensitive(bool)
	Show()
	ShowAll()
}

type application struct {
	State  *state.State
	Window *gtk.Window
	Header *Header
	Grid   *gtk.Grid

	// Dynamic sidebars and main pages
	Sidebar  ExtendedWidget
	Messages ExtendedWidget

	// Stuff
	Guilds *Guilds
	Guild  *Guild

	// nil after finalize()
	sbox      *gtk.Box
	spinner   *gtk.Spinner
	iconTheme *gtk.IconTheme

	css    *gtk.CssProvider
	parser *md.Parser

	// used for events
	busy sync.Mutex
	done chan struct{}
}

func Init() error {
	App = new(application)
	App.done = make(chan struct{})

	if err := App.init(); err != nil {
		return errors.Wrap(err, "Failed to start Gtk")
	}

	// Things beyond this point must use must() or gdk.IdleAdd.
	return nil
}

func UseState(s *state.State) error {
	App.State = s
	App.Window.Remove(App.sbox)
	App.parser = md.NewParser(s)

	if err := App.Header.Hamburger.Refresh(s); err != nil {
		return errors.Wrap(err, "Failed to refresh hamburger")
	}

	{
		gw, err := gtk.ScrolledWindowNew(nil, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to make guilds scrollbar")
		}
		gw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)

		gs, err := newGuilds()
		if err != nil {
			return errors.Wrap(err, "Failed to make guilds view")
		}
		App.Guilds = gs

		must(gw.Add, gs.ListBox)
		must(App.Grid.Add, gw)

		s, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
		if err != nil {
			return errors.Wrap(err, "Failed to create separator")
		}
		must(App.Grid.Add, s)
	}

	// Finalize the window:
	App.finalize()

	// 100 goroutines is pretty cheap (lol)
	for _, g := range App.Guilds.Guilds {
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

	App.hookEvents()
	App.wait()

	return nil
}

func (a *application) init() error {
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
	go func() {
		runtime.LockOSThread()
		gtk.Main()
	}()

	return nil
}

func (a *application) finalize() {
	must(func() {
		a.Window.Remove(a.spinner)
		a.Window.Add(a.Grid)
		a.Window.ShowAll()
		a.spinner.Stop()
		a.sbox.SetSizeRequest(ChannelsWidth, -1)
	})
}

func (a *application) close() {
	if err := a.State.Close(); err != nil {
		log.Errorln("Failed to close Discord:", err)
	}
}

func (a *application) setChannelCol(w ExtendedWidget) {
	a.Sidebar = w
	a.Grid.Attach(w, 2, 0, 1, 1)
}
func (a *application) setMessageCol(w ExtendedWidget) {
	a.Messages = w
	a.Grid.Attach(w, 4, 0, 1, 1)
}

func (a *application) loadGuild(g *Guild) {
	a.busy.Lock()

	must(func() {
		if a.Sidebar != nil {
			a.Grid.Remove(a.Sidebar)
		}

		a.Guilds.SetSensitive(false)
		a.spinner.Start()
		a.setChannelCol(a.sbox)

		go a._loadGuild(g)
	})
}

func (a *application) _loadGuild(g *Guild) {
	defer a.busy.Unlock()
	defer must(func() {
		a.spinner.Stop()
		a.Grid.Remove(a.sbox)
		a.setChannelCol(g.Channels)
		g.Channels.ShowAll()
		a.Guilds.SetSensitive(true)
	})

	if err := g.loadChannels(); err != nil {
		logWrap(err, "Failed to load channels")
		return
	}

	a.Guild = g

	if a.Sidebar == nil {
		s, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
		if err != nil {
			logWrap(err, "Failed to make a separator")
		} else {
			must(a.Grid.Add, s)
		}
	}

	a.Header.UpdateGuild(g.Name)
	go a.loadChannel(g, g.Current())
}

func (a *application) loadChannel(g *Guild, ch *Channel) {
	a.busy.Lock()

	must(func() {
		if a.Messages != nil {
			a.Grid.Remove(a.Messages)
		}

		g.Channels.Main.SetSensitive(false)
		a.spinner.Start()
		a.setMessageCol(a.sbox)

		go a._loadChannel(g, ch)
	})
}

func (a *application) _loadChannel(g *Guild, ch *Channel) {
	defer a.busy.Unlock()
	defer must(func() {
		a.spinner.Stop()
		a.Grid.Remove(a.sbox)
		a.setMessageCol(ch.Messages)
		ch.Messages.ShowAll()
		g.Channels.Main.SetSensitive(true)
	})

	// Run hook
	a.Header.UpdateChannel(ch.Name, ch.Topic)

	if err := g.GoTo(ch); err != nil {
		logWrap(err, "Failed to go to channel")
		return
	}
}

func (a *application) wait() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-sig:
	case <-a.done:
	}
}
