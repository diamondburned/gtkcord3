package gtkcord

import (
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/session"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/hamburger"
	"github.com/diamondburned/gtkcord3/gtkcord/components/header"
	"github.com/diamondburned/gtkcord3/gtkcord/components/login"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/components/quickswitcher"
	"github.com/diamondburned/gtkcord3/gtkcord/components/singlebox"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/config/lastread"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils/gdbus"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/gtkcord/variables"
	"github.com/diamondburned/gtkcord3/internal/keyring"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/glib"
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

const SettingsFile = "settings.json"
const LastReadFile = "lastread.json"

type Application struct {
	*gtk.Application
	Window *window.Container

	Notifier *gdbus.Notifier
	MPRIS    *gdbus.MPRISWatcher

	Plugins []*Plugin

	State *ningen.State

	// Preferences window, hidden by default
	Settings *Settings

	// Main Grid, left is always LeftGrid - *gtk.Grid
	Main  *handy.Leaflet // LeftGrid -- Right
	Right *singlebox.Box // Stack of Messages or full screen server details TODO
	// <grid>        <item>       <item>
	// | Left        | Right    |
	// | Left Grid   | Messages |

	// Left Grid
	LeftGrid *gtk.Grid
	leftCols map[int]gtk.IWidget
	// <item>     <item>
	// | Guilds   | Channels

	// Application states
	Header   *header.Header
	Guilds   *guild.Guilds
	Privates *channel.PrivateChannels
	Channels *channel.Channels
	Messages *message.Messages

	// LastRead contains the persistent state of the latest channels mapped from
	// guilds.
	LastRead lastread.State

	busy moreatomic.BusyMutex
	// done chan int // exit code
}

// New is not thread-safe.
func New(app *gtk.Application) *Application {
	plugins, err := loadPlugins()
	if err != nil {
		log.Fatalln("Failed to load plugins:", err)
	}

	discordSettings()
	return &Application{
		Application: app,
		Plugins:     plugins,
		LastRead:    lastread.New(LastReadFile),
		Settings:    MakeSettings(),
	}
}

func (a *Application) Close() {
	// Mark application as exited:
	a.Application = nil

	// Close session on exit:
	if a.State != nil {
		a.State.Close()
	}
}

func (a *Application) Activate() {
	// Register the dbus connection:
	conn := gdbus.FromApplication(&a.Application.Application)
	a.Notifier = gdbus.NewNotifier(conn)
	a.MPRIS = gdbus.NewMPRISWatcher(conn) // notify.go

	// Activate the window singleton:
	if err := window.WithApplication(a.Application); err != nil {
		log.Fatalln("Failed to initialize the window:", err)
	}
	a.Window = window.Window

	a.leftCols = map[int]gtk.IWidget{}

	// Set the window specs:
	window.Resize(variables.WindowWidth, variables.WindowHeight)
	window.SetTitle("gtkcord")

	// Create the preferences/settings window, which applies settings as a side
	// effect:
	a.Settings.InitWidgets(a)
}

func (a *Application) init() {
	// Pre-make the leaflet but don't use it:
	l := handy.LeafletNew()
	l.SetModeTransitionDuration(150)
	l.SetTransitionType(handy.LEAFLET_TRANSITION_TYPE_SLIDE)
	l.SetInterpolateSize(true)
	l.SetCanSwipeBack(true)
	l.Show()
	a.Main = l

	var _folded bool

	l.Connect("size-allocate", func() {
		// If we're not ready:
		if a.State == nil {
			return
		}

		// Avoid repeating:
		folded := l.GetFold() == handy.FOLD_FOLDED
		if _folded == folded {
			return
		}
		_folded = folded

		// Fold the header too:
		a.Header.Fold(folded)

		// If folded, we expand those panels:
		a.Channels.SetHExpand(folded)
		a.Privates.SetHExpand(folded)
	})

	// Create a new Header:
	h, _ := header.NewHeader()
	a.Header = h
}

func (a *Application) Ready(s *ningen.State) error {
	// Acquire the mutex:
	a.busy.Lock()
	defer a.busy.Unlock()

	a.State = s

	// When the websocket closes, the screen must be changed to a busy one. The
	// websocket may close if it's disconnected unexpectedly.
	s.AddHandler(func(*session.Closed) {
		// Is the application already dead?
		if a.Application == nil {
			return
		}

		// Run this asynchronously. This guarantees that the UI thread would
		// never be hardlocked.
		semaphore.Async(window.NowLoading)
	})

	// Store the token:
	keyring.Set(s.Token)

	// Set gateway error functions to our own:
	s.Gateway.ErrorLog = func(err error) {
		log.Errorln(err)
	}

	semaphore.IdleMust(func() {
		// Make the main widgets:
		a.init()
		window.HeaderDisplay(a.Header) // restore header post login.
		window.Resize(variables.WindowWidth, variables.WindowHeight)
	})

	switch s.Ready.Settings.Theme {
	case "dark":
		semaphore.IdleMust(window.PreferDarkTheme, true)
	case "light":
	}

	// Bind the hamburger:
	hamburger.BindToButton(a.Header.Hamburger.Button, hamburger.Opts{
		State: s,
		Settings: func() {
			a.Settings.Show()
		},
		LogOut: func() {
			go a.LogOut()
		},
	})

	// Bind stuff
	a.bindActions()
	a.bindNotifier()

	// Guilds

	g, err := guild.NewGuilds(s)
	if err != nil {
		return errors.Wrap(err, "Failed to make guilds")
	}
	g.OnSelect = func(g *guild.Guild) {
		a.SwitchGuild(g)
		a.SwitchLastChannel(g)
	}
	g.DMButton.OnClick = func() {
		a.SwitchDM()
		a.SwitchLastChannel(nil)
	}

	// Channels and DMs

	c := channel.NewChannels(s, func(ch *channel.Channel) {
		a.SwitchChannel(ch)
		a.FocusMessages()
	})

	p := channel.NewPrivateChannels(s, func(ch *channel.PrivateChannel) {
		a.SwitchChannel(ch)
		a.FocusMessages()
	})

	// Messages

	m, err := message.NewMessages(s, message.Opts{
		InputZeroWidth: a.Settings.General.Behavior.ZeroWidth,
		InputOnTyping:  a.Settings.General.Behavior.OnTyping,
		MessageWidth:   a.Settings.General.Customization.MessageWidth,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to make messages")
	}

	// Binds

	a.Guilds = g
	a.Channels = c
	a.Privates = p
	a.Messages = m

	// Bind OnClick to trigger below callback:
	a.Header.Back.OnClick = func() {
		a.Main.SetFocusChild(a.LeftGrid)
	}

	// Bind the channel menu button:
	a.Header.ChMenuBtn.SetSpawner(func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		guID := a.Messages.GetGuildID()
		chID := a.Messages.GetChannelID()

		if !chID.IsValid() || !guID.IsValid() {
			// guarded, shouldn't happen.
			return nil
		}

		return header.NewChMenuBody(p, s, guID, chID)
	})

	semaphore.IdleMust(func() {
		// Bind to set-focus-child so swiping left works too.
		a.Main.Connect("set-focus-child", func(_ *glib.Object, w *gtk.Widget) {
			if w == nil {
				return
			}

			switch name, _ := w.GetName(); name {
			case "left":
				a.Main.SetVisibleChild(a.LeftGrid)
				a.Header.SetVisibleChild(a.Header.LeftSide)
			case "right":
				a.Main.SetVisibleChild(a.Right)
				a.Header.SetVisibleChild(a.Header.RightSide)
			}
		})

		// Guilds and Channels grid:
		g1, _ := gtk.GridNew()
		g1.Show()
		g1.SetName("left")
		g1.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
		g1.SetRowHomogeneous(true)
		a.LeftGrid = g1

		// Add the guilds and the separator right and left of channels:
		a.setLeftGridCol(a.Guilds, 0)
		a.setLeftGridCol(newSeparator(), 1)
		a.setLeftGridCol(newSeparator(), 3)

		// Set the left grid to the main leaflet:
		a.Main.Add(g1)

		// Make left grid the default view:
		a.Main.SetVisibleChild(g1)

		// Message widget container, which will hold *Messages:
		c, _ := singlebox.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		c.Show()
		c.SetName("right")
		c.SetHExpand(true)
		c.SetVExpand(true)

		// Set the message container to the main container:
		a.Right = c
		a.Main.Add(c)

		// Display the grid and header
		window.Display(a.Main)
		window.Show()

		// Make a Quick switcher
		quickswitcher.Bind(quickswitcher.Spawner{
			State: s,
			OnGuild: func(id discord.GuildID) {
				if g, _ := a.Guilds.FindByID(id); g != nil {
					g.Activate()
				}
			},
			OnChannel: func(ch discord.ChannelID, guild discord.GuildID) {
				a.SwitchToID(ch, guild)
			},
		})
	})

	// Finally, mark plugins as ready:
	a.readyPlugins()

	return nil
}

func (a *Application) LogOut() {
	a.busy.Lock()
	defer a.busy.Unlock()

	// Disable the entire application:
	semaphore.IdleMust(window.Window.SetSensitive, false)
	defer semaphore.IdleMust(window.Window.SetSensitive, true) // restore last

	// First we need to close the session:
	if err := a.State.Close(); err != nil {
		log.Errorln("Failed to close:", err)
	}
	a.State = nil

	// Then we delete the keyrings:
	keyring.Delete()

	// Then we call the login dialog and exit without waiting:
	semaphore.Async(func() {
		l := login.NewLogin(func(s *ningen.State) {
			if err := a.Ready(s); err != nil {
				log.Fatalln("Failed to re-login:", err)
			}
		})

		l.Run()
	})
}

func (a *Application) lastAccess(guild discord.GuildID, ch discord.ChannelID) discord.ChannelID {
	// read
	if !ch.IsValid() {
		return a.LastRead.Access(guild)
	}
	a.LastRead.SetAccess(guild, ch)
	return ch
}

func newSeparator() *gtk.Separator {
	s, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	s.Show()
	return s
}

func (a *Application) setLeftGridCol(w gtk.IWidget, n int) {
	setGridCol(a.LeftGrid, a.leftCols, w, n)
}

func setGridCol(grid *gtk.Grid, gridStore map[int]gtk.IWidget, w gtk.IWidget, n int) {
	if w, ok := gridStore[n]; ok {
		grid.Remove(w)
	}
	gridStore[n] = w
	grid.Attach(w, n, 0, 1, 1)
}
