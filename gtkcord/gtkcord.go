package gtkcord

import (
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/header"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var HTTPClient = http.Client{
	Timeout: 10 * time.Second,
}

// var App *application

func discordSettings() {
	discord.DefaultEmbedColor = 0x808080
	api.UserAgent = "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/79.0.3945.130 " +
		"Safari/537.36"

	gateway.Identity = gateway.IdentifyProperties{
		OS: "linux",
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
	discordSettings()
}

const SpinnerSize = 56

type Application struct {
	State *ningen.State

	// Main Grid
	Grid *gtk.Grid
	// <item> <separator> <item> <separator> <item>
	//  0      1           2      3           4

	// Application states
	Header   *header.Header
	Guilds   *guild.Guilds
	Privates *channel.PrivateChannels
	Channels *channel.Channels
	Messages *message.Messages

	busy sync.Mutex
}

// New is not thread-safe.
func New() (*Application, error) {
	var a = &Application{}

	// Pre-make the grid but don't use it:
	g, err := gtk.GridNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create grid")
	}
	g.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
	g.SetRowHomogeneous(true)
	a.Grid = g

	// Instead, use the spinner:
	s, err := animations.NewSpinner(75)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create spinner")
	}

	window.Display(s)
	window.Resize(1200, 850)
	window.ShowAll()

	return a, nil
}

func (a *Application) setCol(w gtk.IWidget, n int) {
	a.Grid.Attach(w, n, 0, 1, 1)
}

func (a *Application) Ready(s *ningen.State) error {
	log.Println("Logged in.")

	a.State = s

	// Set gateway error functions to our own:
	s.Gateway.ErrorLog = func(err error) {
		log.Errorln("Discord error:", err)
	}

	semaphore.IdleMust(window.Resize, 1200, 850)

	log.Println("Making headers")

	// Create a new Header:
	h, err := header.NewHeader(s)
	if err != nil {
		return errors.Wrap(err, "Failed to create header")
	}

	log.Println("Making guilds")

	g, err := guild.NewGuilds(s)
	if err != nil {
		return errors.Wrap(err, "Failed to make guilds")
	}
	g.OnSelect = a.SwitchGuild
	g.DMButton.OnClick = a.SwitchDM

	log.Println("Making channels")

	c := channel.NewChannels(s)
	c.OnSelect = a.SwitchChannel

	p := channel.NewPrivateChannels()
	p.OnSelect = a.SwitchDMChannel

	log.Println("Making messages")

	m, err := message.NewMessages(s)
	if err != nil {
		return errors.Wrap(err, "Failed to make messages")
	}

	a.Header = h
	a.Guilds = g
	a.Channels = c
	a.Privates = p
	a.Messages = m

	semaphore.IdleMust(func() {
		log.Println("Set the guilds column")
		a.setCol(a.Guilds, 0)

		log.Println("Set the separators")
		a.setCol(newSeparator(), 1)
		a.setCol(newSeparator(), 3)

		log.Println("Display everything")
		window.Display(a.Grid)
		window.HeaderDisplay(h)

		log.Println("Show all")
		window.ShowAll()
	})

	return nil
}

func (a *Application) SwitchGuild(g *guild.Guild) {
	a.changeCol(a.Channels, 2, func() func() bool {
		a.Channels.Cleanup()
		a.Privates.Cleanup()

		return func() bool {
			err := a.Channels.LoadGuild(g.ID)
			if err != nil {
				log.Errorln("Failed to load guild:", err)
			}
			log.Println("Loaded channels in guild")
			return err == nil
		}
	})
}

func (a *Application) SwitchDM() {
	a.changeCol(a.Privates, 2, func() func() bool {
		a.Channels.Cleanup()
		a.Privates.Cleanup()

		return func() bool {
			a.Privates.LoadChannels(a.State, a.State.Ready.PrivateChannels)
			return true
		}
	})
}

func (a *Application) SwitchChannel(ch *channel.Channel) {
	a.changeCol(a.Messages, 4, func() func() bool {
		a.Messages.Cleanup()

		return func() bool {
			err := a.Messages.Load(ch.ID)
			if err != nil {
				log.Errorln("Failed to load messages:", err)
			}
			return err == nil
		}
	})
}

func (a *Application) SwitchDMChannel(pc *channel.PrivateChannel) {
	a.changeCol(a.Messages, 4, func() func() bool {
		a.Messages.Cleanup()

		return func() bool {
			err := a.Messages.Load(pc.ID)
			if err != nil {
				log.Errorln("Failed to load messages:", err)
			}
			return err == nil
		}
	})
}

func (a *Application) changeCol(w gtkutils.ExtendedWidget, n int, cleanup func() func() bool) {
	// Lock
	a.busy.Lock()
	defer a.busy.Unlock()

	// Clean up channels
	fn := cleanup()

	// Blur the grid
	semaphore.Async(a.Grid.SetSensitive, false)
	defer semaphore.Async(a.Grid.SetSensitive, true)

	// Add a spinner here
	var spinner gtk.IWidget
	semaphore.IdleMust(func() {
		a.Grid.Remove(w)
		spinner, _ = animations.NewSpinner(SpinnerSize)
		a.setCol(spinner, n)
	})

	if !fn() {
		a.Grid.Remove(spinner)
		return
	}

	// Replace the spinner with the actual channel:
	semaphore.IdleMust(func() {
		a.Grid.Remove(spinner)
		a.setCol(w, n)
		w.ShowAll()
	})
}

func newSeparator() *gtk.Separator {
	s, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	return s
}

// func (a *application) GuildID() discord.Snowflake {
// 	if a.Guild == nil {
// 		return 0
// 	}
// 	return a.Guild.ID
// }

// func (a *application) ChannelID() discord.Snowflake {
// 	mw, ok := App.Messages.(*message.Messages)
// 	if !ok {
// 		return 0
// 	}
// 	return mw.ChannelID
// }

// func (a *application) close() {
// 	if err := a.State.Close(); err != nil {
// 		log.Errorln("Failed to close Discord:", err)
// 	}
// }

// func (a *application) setChannelCol(w gtkutils.ExtendedWidget) {
// 	if w == nil {
// 		if a.Sidebar != nil {
// 			a.Grid.Remove(a.Sidebar)
// 			a.Sidebar = nil
// 		}

// 		return
// 	}

// 	a.Sidebar = w
// 	a.Grid.Attach(w, 2, 0, 1, 1)
// }

// func (a *application) setMessagesCol(w gtkutils.ExtendedWidget) {
// 	if w == nil {
// 		if a.Messages != nil {
// 			a.Grid.Remove(a.Messages)
// 			a.Messages = nil
// 		}

// 		return
// 	}

// 	a.Messages = w
// 	a.Grid.Attach(w, 4, 0, 1, 1)
// }

// func (a *application) loadGuild(g *Guild) {
// 	a.busy.Lock()
// 	defer a.busy.Unlock()

// 	// We don't need a spinner if it's a DM guild:
// 	if g == nil {
// 		a.Guild = nil

// 		a.Header.UpdateGuild("Private Messages")
// 		must(a.setChannelCol, App.Privates)

// 		go a.loadPrivate(App.Privates.Selected())
// 		return
// 	}

// 	a._loadGuild(g)

// 	ch := g.Current()
// 	if ch == nil {
// 		return
// 	}

// 	go a.loadChannel(g, ch)
// }

// func (a *application) _loadGuild(g *Guild) {
// 	must(a.Guilds.SetSensitive, false)

// 	if a.Sidebar != nil {
// 		must(a.Grid.Remove, a.Sidebar)
// 	}

// 	must(a.spinner.Start)
// 	must(a.sbox.SetSizeRequest, ChannelsWidth, -1)
// 	must(a.setChannelCol, a.sbox)

// 	if err := g.loadChannels(); err != nil {
// 		must(a._loadGuildDone, g)

// 		logWrap(err, "Failed to load channels")
// 		return
// 	}

// 	a.Guild = g
// 	first := a.Sidebar == nil
// 	must(a._loadGuildDone, g)

// 	if first {
// 		s := must(gtk.SeparatorNew, gtk.ORIENTATION_VERTICAL).(*gtk.Separator)
// 		must(a.Grid.Add, s)
// 	}

// 	a.Header.UpdateGuild(g.Name)
// }

// func (a *application) _loadGuildDone(g *Guild) {
// 	a.spinner.Stop()
// 	a.Grid.Remove(a.sbox)
// 	a.setChannelCol(g.Channels)
// 	g.Channels.ShowAll()
// 	a.Guilds.SetSensitive(true)
// }

// func (a *application) loadChannel(g *Guild, ch *Channel) {
// 	must(g.Channels.Main.SetSensitive, false)
// 	defer must(g.Channels.Main.SetSensitive, true)

// 	done := a.checkMessages(ch.Messages)
// 	if done == nil {
// 		return
// 	}
// 	defer done()

// 	a._loadMessages(func() (*message.Messages, error) {
// 		must(a.Header.UpdateChannel, ch.Name, ch.Topic)
// 		if err := g.GoTo(ch); err != nil {
// 			return nil, err
// 		}
// 		return ch.Messages, nil
// 	})
// }

// func (a *application) loadPrivate(p *PrivateChannel) {
// 	must(a.Privates.SetSensitive, false)
// 	defer must(a.Privates.SetSensitive, true)

// 	done := a.checkMessages(p.Messages)
// 	if done == nil {
// 		return
// 	}
// 	defer done()

// 	a._loadMessages(func() (*message.Messages, error) {
// 		must(a.Header.UpdateChannel, p.Name, "")
// 		if err := p.loadMessages(); err != nil {
// 			return nil, err
// 		}
// 		return p.Messages, nil
// 	})
// }

// func (a *application) checkMessages(m *message.Messages) func() {
// 	a.busy.Lock()
// 	defer a.busy.Unlock()

// 	old := a.Messages
// 	if old != nil {
// 		if old == m {
// 			return nil
// 		}

// 		must(a.Grid.Remove, old)
// 		return func() { must(old.Destroy) }
// 	}

// 	return func() {}
// }

// func (a *application) _loadMessages(load func() (*message.Messages, error)) {
// 	a.busy.Lock()
// 	defer a.busy.Unlock()

// 	a.Messages = nil

// 	must(func() {
// 		a.spinner.Start()
// 		a.sbox.SetSizeRequest(-1, -1)
// 		a.Grid.Attach(a.sbox, 4, 0, 1, 1)
// 	})

// 	m, err := load()
// 	if err != nil {
// 		must(a._loadChannelDone)
// 		logWrap(err, "Failed to go to channel")
// 		return
// 	}

// 	must(func() {
// 		a._loadChannelDone()
// 		a.setMessagesCol(m)
// 		m.Show()
// 	})
// }

// func (a *application) _loadChannelDone() {
// 	a.spinner.Stop()
// 	a.Grid.Remove(a.sbox)
// }
