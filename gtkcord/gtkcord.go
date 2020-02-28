package gtkcord

import (
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/window"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/diamondburned/gtkcord3/ningen"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var HTTPClient = http.Client{
	Timeout: 10 * time.Second,
}

var App *application

func discordSettings() {
	discord.DefaultEmbedColor = 0x808080
	api.UserAgent = "linux"
	gateway.Identity = gateway.IdentifyProperties{
		OS: "linux",
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
	discordSettings()

	App = new(application)
}

type application struct {
	gtkutils.ExtendedWidget

	Grid *gtk.Grid

	State *ningen.State

	// self stufff
	Me *discord.User

	Header *Header

	// Dynamic sidebars and main pages
	Sidebar  gtkutils.ExtendedWidget
	Messages gtkutils.ExtendedWidget

	// Stuff
	Guilds *Guilds
	Guild  *Guild

	// nil after finalize()
	sbox      *gtk.Box
	spinner   *gtk.Spinner
	iconTheme *gtk.IconTheme

	css       *gtk.CssProvider
	parser    *md.Parser
	clipboard *gtk.Clipboard

	// used for events
	busy sync.RWMutex
	done chan struct{}

	completionQueue chan func()
}

func Init() error {
	if App.done != nil {
		return nil
	}

	App.done = make(chan struct{})
	a := App

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
	sbox.SetVAlign(gtk.ALIGN_CENTER)
	sbox.SetHAlign(gtk.ALIGN_CENTER)
	sbox.SetSizeRequest(50, 50)

	// Add the spinner into the window instead of the spinner:
	window.Display(sbox)
	a.sbox = sbox

	s, err := gtk.SpinnerNew()
	if err != nil {
		return errors.Wrap(err, "Failed to create spinner")
	}
	s.SetVAlign(gtk.ALIGN_CENTER)
	s.SetHAlign(gtk.ALIGN_CENTER)
	s.SetSizeRequest(50, 50)
	s.Start()

	sbox.Add(s)
	a.spinner = s

	window.Resize(1200, 850)
	window.ShowAll()

	return nil
}

func Ready(s *ningen.State) error {
	// Set gateway error functions to our own:
	s.Gateway.ErrorLog = func(err error) {
		log.Errorln("Discord error:", err)
	}

	must(window.Resize, 1200, 850)

	// Create a new header placeholder:
	h, err := newHeader()
	if err != nil {
		return errors.Wrap(err, "Failed to create header")
	}
	App.Header = h
	must(window.HeaderDisplay, h)

	// Set variables, etc.
	App.State = s
	App.parser = md.NewParser(s.State)
	App.parser.UserPressed = userMentionPressed
	App.parser.ChannelPressed = channelMentionPressed

	u, err := s.Me()
	if err != nil {
		return errors.Wrap(err, "Failed to get current user")
	}
	App.Me = u
	App.Header.Hamburger.User.Update(*u)
	App.Header.Hamburger.User.UpdateStatus(s.Ready.Settings.Status)

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
		gtkutils.InjectCSS(gs, "guilds", "")

		must(gw.Add, gs)
		must(App.Grid.Add, gw)

		s, err := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
		if err != nil {
			return errors.Wrap(err, "Failed to create separator")
		}
		must(App.Grid.Add, s)
	}

	// Finalize the window:
	must(window.Display, App.Grid)
	must(window.ShowAll)

	// Finalize the spinner so it can be reused:
	must(App.spinner.Stop)
	must(App.sbox.SetHExpand, true)
	must(App.sbox.SetHAlign, gtk.ALIGN_CENTER)
	must(App.sbox.SetVAlign, gtk.ALIGN_CENTER)

	// Start the completion queue:
	App.completionQueue = make(chan func())
	go func() {
		for fn := range App.completionQueue {
			fn()
		}
	}()

	App.hookEvents()
	App.hookReads()

	// Start the garbage collector:
	// (Too unstable right now)
	// go App.cleanUp()

	return nil
}

func (a *application) GuildID() discord.Snowflake {
	if a.Guild == nil {
		return 0
	}
	return a.Guild.ID
}

func (a *application) ChannelID() discord.Snowflake {
	mw, ok := App.Messages.(*Messages)
	if !ok {
		return 0
	}
	return mw.Channel.ID
}

func (a *application) close() {
	if err := a.State.Close(); err != nil {
		log.Errorln("Failed to close Discord:", err)
	}
}

func (a *application) setChannelCol(w gtkutils.ExtendedWidget) {
	a.Sidebar = w
	a.Grid.Attach(w, 2, 0, 1, 1)
}
func (a *application) setMessagesCol(w gtkutils.ExtendedWidget) {
	a.Messages = w
	a.Grid.Attach(w, 4, 0, 1, 1)
}

func (a *application) loadGuild(g *Guild) {
	a._loadGuild(g)

	ch := g.Current()
	if ch == nil {
		return
	}
	a.loadChannel(g, ch)
}

func (a *application) _loadGuild(g *Guild) {
	must(a.Guilds.SetSensitive, false)

	a.busy.Lock()
	defer a.busy.Unlock()

	if a.Sidebar != nil {
		must(a.Grid.Remove, a.Sidebar)
	}

	must(a.spinner.Start)
	must(a.sbox.SetSizeRequest, ChannelsWidth, -1)
	must(a.setChannelCol, a.sbox)

	if err := g.loadChannels(); err != nil {
		must(a._loadGuildDone, g)

		logWrap(err, "Failed to load channels")
		return
	}

	a.Guild = g
	first := a.Sidebar == nil
	must(a._loadGuildDone, g)

	if first {
		s := must(gtk.SeparatorNew, gtk.ORIENTATION_VERTICAL).(*gtk.Separator)
		must(a.Grid.Add, s)
	}

	a.Header.UpdateGuild(g.Name)
}

func (a *application) _loadGuildDone(g *Guild) {
	a.spinner.Stop()
	a.Grid.Remove(a.sbox)
	a.setChannelCol(g.Channels)
	g.Channels.ShowAll()
	a.Guilds.SetSensitive(true)
}

func (a *application) loadChannel(g *Guild, ch *Channel) {
	must(g.Channels.Main.SetSensitive, false)
	defer must(g.Channels.Main.SetSensitive, true)

	a.busy.Lock()
	defer a.busy.Unlock()

	old := a.Messages
	if old != nil {
		if old == ch.Messages {
			return
		}

		must(a.Grid.Remove, old)
		defer must(old.Destroy)
	} else {
		s := must(gtk.SeparatorNew, gtk.ORIENTATION_VERTICAL).(*gtk.Separator)
		must(s.Show)
		must(App.Grid.Add, s)
	}

	a.Messages = nil

	must(a.spinner.Start)
	must(a.sbox.SetSizeRequest, -1, -1)
	must(a.Grid.Attach, a.sbox, 4, 0, 1, 1)

	// Run hook
	a.Header.UpdateChannel(ch.Name, ch.Topic)

	if err := g.GoTo(ch); err != nil {
		must(a._loadChannelDone, g, ch)

		logWrap(err, "Failed to go to channel")
		return
	}

	must(a._loadChannelDone, g, ch)
	must(a.setMessagesCol, ch.Messages)
	must(ch.Messages.Show)
}

func (a *application) _loadChannelDone(g *Guild, ch *Channel) {
	a.spinner.Stop()
	a.Grid.Remove(a.sbox)
}

func (a *application) cleanUp() {
	for range time.Tick(30 * time.Second) {
		a._cleanUp()
	}
}

func (a *application) _cleanUp() {
	a.busy.Lock()
	defer a.busy.Unlock()

	if a.Guilds == nil {
		return
	}

	for _, guild := range a.Guilds.Guilds {
		if guild.Channels == nil {
			continue
		}

		for _, channel := range guild.Channels.Channels {
			if channel.Messages == nil {
				continue
			}
			if channel.Messages == a.Messages {
				continue
			}

			m := channel.Messages
			m.guard.Lock()

			var count = 0

			for i, msg := range m.messages {
				if msg == nil || msg.isBusy() {
					continue
				}
				count++

				m.Main.Remove(msg)
				m.messages[i].main.Unref()
				m.messages[i] = nil
			}

			m.guard.Unlock()

			log.Infoln(
				"Garbage collected", count,
				"messages in channel",
				channel.Name, "of guild", guild.Name,
			)
		}
	}
}
