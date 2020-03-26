package gtkcord

import (
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/header"
	"github.com/diamondburned/gtkcord3/gtkcord/components/login"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/components/quickswitcher"
	"github.com/diamondburned/gtkcord3/gtkcord/components/singlebox"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/keyring"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
	"github.com/sasha-s/go-deadlock"
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

const (
	SpinnerSize  = 56
	ChannelWidth = 240
)

type Application struct {
	*gtk.Application
	Window *window.Container

	State *ningen.State

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

	// GuildID -> ChannelID; if GuildID == 0 then DM
	LastAccess map[discord.Snowflake]discord.Snowflake
	lastAccMut sync.Mutex

	busy deadlock.Mutex
	done chan int // exit code
}

// New is not thread-safe.
func New() (*Application, error) {
	a, err := gtk.ApplicationNew("org.diamondburned.gtkcord", glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new *gtk.Application")
	}

	app := &Application{
		Application: a,
	}
	a.Connect("activate", app.activate)

	return app, nil
}

func (a *Application) Start() {
	a.done = make(chan int)

	go func() {
		runtime.LockOSThread()
		a.done <- a.Application.Run(nil)
	}()
}

func (a *Application) Wait() {
	sig := <-a.done

	// Close session on exit:
	if a.State != nil {
		a.State.Close()
	}

	if sig != 0 {
		os.Exit(sig)
	}
}

func (a *Application) activate() {
	// Activate the window singleton:
	if err := window.WithApplication(a.Application); err != nil {
		log.Fatalln("Failed to initialize the window:", err)
	}
	a.Window = window.Window

	a.leftCols = map[int]gtk.IWidget{}
	a.LastAccess = map[discord.Snowflake]discord.Snowflake{}

	// Pre-make some widgets but don't use them:
	a.init()

	// Use the spinner instead of the Leaflet:
	s, _ := animations.NewSpinner(75)

	// Use a custom header instead of the actual Header:
	h, _ := gtk.HeaderBarNew()
	h.SetTitle("Connecting to Discord.")
	h.SetShowCloseButton(true)

	window.Display(s)
	window.HeaderDisplay(h)
	window.Resize(1200, 900)
	window.SetTitle("gtkcord")
	window.ShowAll()
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

	// Create a new Header:
	h, _ := header.NewHeader()
	h.Hamburger.LogOut = a.LogOut // bind
	a.Header = h
}

func (a *Application) Ready(s *ningen.State) error {
	a.State = s

	// Store the token:
	keyring.Set(s.Token)

	// Set gateway error functions to our own:
	s.Gateway.ErrorLog = func(err error) {
		log.Errorln(err)
	}

	semaphore.IdleMust(window.HeaderDisplay, a.Header) // restore header post login.
	semaphore.IdleMust(window.Resize, 1200, 900)

	// Set Markdown's highlighting theme
	switch s.Ready.Settings.Theme {
	case "dark":
		md.ChangeStyle("monokai")
		semaphore.IdleMust(window.PreferDarkTheme, true)
	case "light":
		md.ChangeStyle("monokailight")
	}

	// Have Hamburger use the state:
	a.Header.Hamburger.UseState(s)

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

	c := channel.NewChannels(s)
	c.OnSelect = func(ch *channel.Channel) {
		a.SwitchChannel(ch)
		a.FocusMessages()
	}

	p := channel.NewPrivateChannels(s)
	p.OnSelect = func(ch *channel.PrivateChannel) {
		a.SwitchChannel(ch)
		a.FocusMessages()
	}

	// Messages

	m, err := message.NewMessages(s)
	if err != nil {
		return errors.Wrap(err, "Failed to make messages")
	}

	// Binds

	a.Guilds = g
	a.Channels = c
	a.Privates = p
	a.Messages = m

	// Expand the channel list on fold:
	a.Header.OnFold = func(folded bool) {
		// If folded, we expand those panels:
		a.Channels.SetHExpand(folded)
		a.Privates.SetHExpand(folded)
	}

	// Bind OnClick to trigger below callback:
	a.Header.Back.OnClick = func() {
		a.Main.SetFocusChild(a.LeftGrid)
	}

	// Bind the channel menu button:
	a.Header.ChMenuBtn.SetSpawner(func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		guID := a.Messages.GetGuildID()
		chID := a.Messages.GetChannelID()

		if !chID.Valid() || !guID.Valid() {
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

		// Set widths:
		a.Channels.SetSizeRequest(ChannelWidth, -1)
		a.Privates.SetSizeRequest(ChannelWidth, -1)

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
			OnGuild: func(id discord.Snowflake) {
				if g, _ := a.Guilds.FindByID(id); g != nil {
					semaphore.IdleMust(g.Row.Activate)
				}
			},
			OnChannel: func(ch, guild discord.Snowflake) {
				var row *gtk.ListBoxRow
				if g, _ := a.Guilds.FindByID(guild); g != nil {
					a.SwitchGuild(g)
					if channel := a.Channels.FindByID(ch); channel != nil {
						row = channel.Row
					}
				} else {
					a.SwitchDM()
					if channel := a.Privates.FindByID(ch); channel != nil {
						row = channel.ListBoxRow
					}
				}

				if row != nil {
					semaphore.IdleMust(row.Activate)
				}
			},
		})
	})

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

	// Then we reinitialize some widgets:
	a.init()

	// Then we call the login dialog and exit without waiting:
	go semaphore.IdleMust(func() {
		l := login.NewLogin(func(s *ningen.State) {
			if err := a.Ready(s); err != nil {
				log.Fatalln("Failed to re-login:", err)
			}
		})

		l.Run()
	})
}

func (a *Application) lastAccess(guild, ch discord.Snowflake) discord.Snowflake {
	a.lastAccMut.Lock()
	defer a.lastAccMut.Unlock()

	if !ch.Valid() {
		if id, ok := a.LastAccess[guild]; ok {
			return id
		}
		return 0
	}

	a.LastAccess[guild] = ch
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
