package gtkcord

import (
	"math/rand"
	"net/http"
	"time"

	_ "embed"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/greet"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/hamburger"
	"github.com/diamondburned/gtkcord3/gtkcord/components/header"
	"github.com/diamondburned/gtkcord3/gtkcord/components/login"
	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/components/quickswitcher"
	"github.com/diamondburned/gtkcord3/gtkcord/components/singlebox"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils/gdbus"
	"github.com/diamondburned/ningen/v2"

	"github.com/diamondburned/gtkcord3/internal/keyring"
	"github.com/diamondburned/gtkcord3/internal/log"
)

//go:embed style.css
var styleCSS string

func init() {
	window.ApplicationCSS = styleCSS
}

var HTTPClient = http.Client{
	Timeout: 10 * time.Second,
}

func discordSettings() {
	discord.DefaultEmbedColor = 0x808080
	api.UserAgent = "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/79.0.3945.130 " +
		"Safari/537.36"

	gateway.DefaultIdentity = gateway.IdentifyProperties{
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

	Notifier   *gdbus.Notifier
	MPRIS      *gdbus.MPRISWatcher
	mprisState *mprisState

	Plugins []*Plugin

	State *ningen.State

	// Preferences window, hidden by default
	Settings *Settings

	// Main Grid, left is always LeftGrid - *gtk.Grid
	Main       *handy.Flap // LeftGrid -- Right
	RightWhole *gtk.Box
	Right      *singlebox.Box // Stack of Messages or full screen server details TODO
	// <grid>        <item>       <item>
	// | Left        | Right    |
	// | Left Grid   | Messages |

	// Left Grid
	LeftWhole *gtk.Box
	LeftGrid  *gtk.Grid
	leftCols  [maxLeftGridColumn]gtk.Widgetter
	// <item>     <item>
	// | Guilds   | Channels

	// Application states
	Header   *header.Header
	Guilds   *guild.Guilds
	Privates *channel.PrivateChannels
	Channels *channel.Channels
	Messages *message.Messages
}

// New is not thread-safe.
func New(app *gtk.Application) *Application {
	plugins, err := loadPlugins()
	if err != nil {
		log.Fatalln("Failed to load plugins:", err)
	}

	return &Application{
		Application: app,
		Plugins:     plugins,
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
	conn := a.DBusConnection()
	a.Notifier = gdbus.NewNotifier(conn)
	a.MPRIS = gdbus.NewMPRISWatcher(conn) // notify.go
	a.mprisState = newMPRISState()

	// Activate the window singleton:
	if err := window.WithApplication(a.Application); err != nil {
		log.Fatalln("Failed to initialize the window:", err)
	}
	a.Window = window.Window

	// Set the window specs:
	window.SetTitle("gtkcord")

	// Create the preferences/settings window, which applies settings as a side
	// effect:
	a.Settings = a.makeSettings()
}

func (a *Application) init() {
	// Pre-make the leaflet but don't use it:
	l := handy.NewFlap()
	l.SetRevealDuration(150)
	l.SetTransitionType(handy.FlapTransitionTypeOver)
	l.SetFoldPolicy(handy.FlapFoldPolicyAuto)
	l.SetSwipeToClose(true)
	l.SetSwipeToOpen(true)
	l.SetModal(true)
	l.Container.Show()
	gtkutils.InjectCSS(l, "main-leaflet", "")
	gtkutils.InjectCSS(l, "main-fold", "")
	a.Main = l

	l.Connect("notify::folded", func() {
		// If we're not ready:
		if a.State == nil {
			return
		}

		folded := l.Folded()
		a.Header.Fold(folded)
		// If folded, we expand those panels:
		a.Channels.SetHExpand(folded)
		a.Privates.SetHExpand(folded)
	})

	// Create a new Header:
	a.Header = header.NewHeader()
}

func (a *Application) displayMain() {
	window.Display(&a.Main.Container)
}

func (a *Application) Ready(s *ningen.State) error {
	a.State = s

	// When the websocket closes, the screen must be changed to a busy one. The
	// websocket may close if it's disconnected unexpectedly.
	s.Gateway.AfterClose = func(error) {
		// Is the application already dead?
		if a.Application == nil {
			return
		}

		// Run this asynchronously. This guarantees that the UI thread would
		// never be hardlocked.
		glib.IdleAdd(func() { window.NowLoading() })
	}

	// Show the main screen once everything is resumed. See above NowLoading
	// call.
	s.AddHandler(func(c *ningen.Connected) {
		glib.IdleAdd(func() { a.displayMain() })
	})

	// Store the token:
	go func() {
		keyring.Set(s.Token)
	}()

	// Set gateway error functions to our own:
	s.Gateway.ErrorLog = func(err error) {
		log.Errorln(err)
	}

	// Make the main widgets:
	a.init()
	window.SetHeader(nil)

	if ready := s.Ready(); ready.UserSettings != nil {
		switch ready.UserSettings.Theme {
		case "dark":
			window.PreferDarkTheme(true)
		case "light":
			window.PreferDarkTheme(false)
		}
	}

	// Bind the hamburger:
	hamburger.BindToButton(a.Header.Hamburger.Button, hamburger.Opts{
		State:    s,
		Settings: a.Settings.Show,
		LogOut:   a.LogOut,
	})

	// Bind stuff
	a.bindActions()
	a.bindNotifier()

	// Guilds

	a.Guilds = guild.NewGuilds(s)
	a.Guilds.OnSelect = a.SwitchGuild
	a.Guilds.DMButton.OnClick = a.SwitchDM

	a.Channels = channel.NewChannels(s, func(ch *channel.Channel) {
		a.SwitchChannel(ch)
		a.FocusMessages()
	})

	a.Privates = channel.NewPrivateChannels(s, func(ch *channel.PrivateChannel) {
		a.SwitchChannel(ch)
		a.FocusMessages()
	})

	a.Messages = message.NewMessages(s, message.Opts{
		InputZeroWidth: a.Settings.General.Behavior.ZeroWidth,
		InputOnTyping:  a.Settings.General.Behavior.OnTyping,
		MessageWidth:   a.Settings.General.Customization.MessageWidth,
	})

	greeter := greet.NewGreeter()
	greeter.SetSurface(logo.Surface(greet.IconSize, 2))
	a.Messages.SetPlaceholder(greeter)

	// Bind OnClick to trigger below callback:
	a.Header.Back.OnClick = func() {
		a.Main.SetRevealFlap(true)
	}

	// Bind the channel menu button:
	a.Header.ChMenuBtn.SetSpawner(func(p *gtk.Popover) gtk.Widgetter {
		guID := a.Messages.GuildID()
		chID := a.Messages.ChannelID()

		if !chID.IsValid() || !guID.IsValid() {
			// guarded, shouldn't happen.
			return nil
		}

		return header.NewChMenuBody(p, s, guID, chID)
	})

	// // Bind to set-focus-child so swiping left works too.
	// a.Main.Connect("set-focus-child", func(w gtk.Widgetter) {
	// 	if w == nil {
	// 		return
	// 	}
	// 	switch gtk.BaseWidget(w).Name() {
	// 	case "left":
	// 		a.Main.SetVisibleChild(a.LeftGrid)
	// 		a.Header.Body.SetVisibleChild(a.Header.LeftSide)
	// 	case "right":
	// 		a.Main.SetVisibleChild(a.Right)
	// 		a.Header.Body.SetVisibleChild(a.Header.RightSide)
	// 	}
	// })

	// Set widths:
	a.Channels.SetSizeRequest(ChannelWidth, -1)
	a.Privates.SetSizeRequest(ChannelWidth, -1)

	// Guilds and Channels grid:
	a.LeftGrid = gtk.NewGrid()
	a.LeftGrid.SetHAlign(gtk.AlignStart)
	a.LeftGrid.SetName("left")
	a.LeftGrid.SetOrientation(gtk.OrientationHorizontal)
	a.LeftGrid.SetRowHomogeneous(true)
	a.LeftGrid.Show()

	a.LeftWhole = gtk.NewBox(gtk.OrientationVertical, 0)
	a.LeftWhole.SetHExpand(false)
	a.LeftWhole.PackStart(a.Header.Left, false, false, 0)
	a.LeftWhole.PackStart(a.LeftGrid, true, true, 0)
	gtkutils.InjectCSS(a.LeftWhole, "left-whole", "")
	a.LeftWhole.Show()

	// Add the guilds and the separator right and left of channels:
	a.setLeftGridCol(a.Guilds, guildsColumn)
	a.setLeftGridCol(newSeparator(), separatorColumn1)
	a.setLeftGridCol(newSeparator(), separatorColumn2)

	// Set the left grid to the main leaflet:
	a.Main.SetFlap(a.LeftWhole)

	// Message widget container, which will hold *Messages:
	a.Right = singlebox.NewBox(gtk.OrientationHorizontal, 0)
	a.Right.SetName("right")
	a.Right.SetHExpand(true)
	a.Right.SetVExpand(true)
	a.Right.Show()

	a.RightWhole = gtk.NewBox(gtk.OrientationVertical, 0)
	a.RightWhole.SetHExpand(true)
	a.RightWhole.PackStart(a.Header.Right, false, false, 0)
	a.RightWhole.PackStart(a.Right, true, true, 0)
	gtkutils.InjectCSS(a.RightWhole, "right-whole", "")
	a.RightWhole.Show()

	// Set the message container to the main container:
	a.Main.SetContent(a.RightWhole)

	// Display the grid and header
	a.displayMain()
	window.Show()

	// Make a Quick switcher
	quickswitcher.Bind(quickswitcher.Spawner{
		State: s,
		OnGuild: func(id discord.GuildID) {
			if g, _ := a.Guilds.FindByID(id); g != nil {
				g.Activate()
			}
		},
		OnChannel: func(chID discord.ChannelID, gID discord.GuildID) {
			a.SwitchToID(chID, gID)
		},
	})

	// Finally, mark plugins as ready:
	a.readyPlugins()

	return nil
}

func (a *Application) LogOut() {
	// Disable the entire application:
	window.Window.SetSensitive(false)

	state := a.State
	a.State = nil

	go func() {
		// First we need to close the session:
		if err := state.CloseGracefully(); err != nil {
			log.Errorln("failed to close gracefully:", err)
		}

		// Then we delete the keyrings:
		keyring.Delete()

		// Then we call the login dialog and exit without waiting:
		glib.IdleAdd(func() {
			window.Window.SetSensitive(true)
			a.ShowLogin("")
		})
	}()
}

// ShowLogin shows the login screen.
func (a *Application) ShowLogin(lastToken string) {
	l := login.NewLogin(func(s *ningen.State) {
		if err := a.Ready(s); err != nil {
			log.Fatalln("failed to login:", err)
		}
		a.Connect("shutdown", func() {
			log.Infoln("app shutting down")
			s.Close()
		})
	})
	l.LastToken = lastToken
	l.Run()
}

func newSeparator() *gtk.Separator {
	s := gtk.NewSeparator(gtk.OrientationVertical)
	s.Show()
	return s
}

type leftGridColPosition uint8

const (
	guildsColumn leftGridColPosition = iota
	separatorColumn1
	channelsColumn
	separatorColumn2
	maxLeftGridColumn
)

func (a *Application) setLeftGridCol(w gtk.Widgetter, n leftGridColPosition) {
	if w := a.leftCols[n]; w != nil {
		a.LeftGrid.Remove(w)
	}
	a.leftCols[n] = w
	a.LeftGrid.Attach(w, int(n), 0, 1, 1)
}
