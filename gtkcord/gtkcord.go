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
	"github.com/diamondburned/gtkcord3/gtkcord/components/members"
	"github.com/diamondburned/gtkcord3/gtkcord/components/message"
	"github.com/diamondburned/gtkcord3/gtkcord/components/quickswitcher"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
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

const (
	SpinnerSize  = 56
	ChannelWidth = 240
)

type Application struct {
	State *ningen.State

	// Main Grid
	Grid *gtk.Grid
	cols map[int]gtk.IWidget
	// <grid>        <item>       <item>
	//  0            1             2
	// | Left Grid   | Messages   | Members |

	// Left Grid
	LeftGrid *gtk.Grid
	LeftRev  *gtk.Revealer
	leftCols map[int]gtk.IWidget
	// <item>     <item>
	// | Guilds   | Channels

	// <item> <separator> <item> <separator> <item> <separator> <item>
	//   0      1           2      3           4      5           6
	// | Guilds           | Channels         | Messages         | Members

	// Application states
	Header   *header.Header
	Guilds   *guild.Guilds
	Privates *channel.PrivateChannels
	Channels *channel.Channels
	Messages *message.Messages
	Members  *members.Revealer

	// GuildID -> ChannelID; if GuildID == 0 then DM
	LastAccess map[discord.Snowflake]discord.Snowflake
	lastAccMut sync.Mutex

	busy sync.Mutex
}

// New is not thread-safe.
func New() (*Application, error) {
	var a = &Application{
		cols:       map[int]gtk.IWidget{},
		leftCols:   map[int]gtk.IWidget{},
		LastAccess: map[discord.Snowflake]discord.Snowflake{},
	}

	// Pre-make the grid but don't use it:
	g, err := gtk.GridNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create grid")
	}
	g.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
	g.SetRowHomogeneous(true)
	g.Show()
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

func (a *Application) Ready(s *ningen.State) error {
	a.State = s

	// Set gateway error functions to our own:
	s.Gateway.ErrorLog = func(err error) {
		log.Errorln(err)
	}

	semaphore.IdleMust(window.Resize, 1200, 850)
	window.Window.Closer = func() {
		if err := s.Close(); err != nil {
			log.Fatalln("Failed to close:", err)
		}
	}

	// Set Markdown's highlighting theme
	switch s.Ready.Settings.Theme {
	case "dark":
		md.ChangeStyle("monokai")
		semaphore.IdleMust(window.PreferDarkTheme, true)
	case "light":
		md.ChangeStyle("monokailight")
	}

	// Create a new Header:
	h, err := header.NewHeader(s)
	if err != nil {
		return errors.Wrap(err, "Failed to create header")
	}

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

	c := channel.NewChannels(s)
	c.OnSelect = func(ch *channel.Channel) {
		a.SwitchChannel(ch)
	}

	p := channel.NewPrivateChannels(s)
	p.OnSelect = func(ch *channel.PrivateChannel) {
		a.SwitchChannel(ch)
	}

	m, err := message.NewMessages(s)
	if err != nil {
		return errors.Wrap(err, "Failed to make messages")
	}

	memberc := members.New(s)
	var mrevealer *members.Revealer

	semaphore.IdleMust(func() {
		// TODO: settings
		revealed := true

		mrevealer = members.NewRevealer(memberc)
		mrevealer.SetRevealChild(revealed)
		mrevealer.BindController(h.Controller)
	})

	a.Header = h
	a.Guilds = g
	a.Channels = c
	a.Privates = p
	a.Messages = m
	a.Members = mrevealer

	// jank shit
	// h.Hamburger.GuildID = &a.Channels.GuildID

	semaphore.IdleMust(func() {
		// Set sizes
		a.Channels.SetSizeRequest(ChannelWidth, -1)

		// Guilds and Channels grid:
		g1, _ := gtk.GridNew()
		g1.Show()
		g1.SetOrientation(gtk.ORIENTATION_HORIZONTAL)
		g1.SetRowHomogeneous(true)
		a.LeftGrid = g1

		// Guilds and Channels revealer:
		r1, _ := gtk.RevealerNew()
		r1.SetTransitionDuration(50)
		r1.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
		r1.Show()
		r1.Add(g1)

		// Bind the revealer:
		h.Hamburger.OnClick = func() {
			revealed := !r1.GetRevealChild()
			r1.SetRevealChild(revealed)
			h.Hamburger.Button.SetActive(revealed)

			if revealed {
				h.GuildBox.Show()
			} else {
				h.GuildBox.Hide()
			}
		}

		// Force display the left panel, which toggles it to true:
		h.Hamburger.OnClick()

		// Add the guilds and the separator right and left of channels:
		a.setLeftGridCol(a.Guilds, 0)
		a.setLeftGridCol(newSeparator(), 1)
		a.setLeftGridCol(newSeparator(), 3)

		// Set the left grid to the main grid:
		a.setGridCol(r1, 0)

		// Message widget placeholder
		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		b.Show()
		b.SetHExpand(true)
		b.SetVExpand(true)

		// Set the message placeholder to the main grid:
		a.setGridCol(b, 1)

		// Display the grid and header
		window.Display(a.Grid)
		window.HeaderDisplay(h)
		window.Show()

		// Make a Quick switcher
		quickswitcher.Bind(quickswitcher.Spawner{
			State: s,
			OnGuild: func(id discord.Snowflake) {
				if g, _ := a.Guilds.FindByID(id); g != nil {
					a.SwitchGuild(g)
				}
			},
			OnChannel: func(ch, guild discord.Snowflake) {
				var channel Channel
				if g, _ := a.Guilds.FindByID(guild); g != nil {
					a.SwitchGuild(g)
					channel = a.Channels.FindByID(ch)
				} else {
					a.SwitchDM()
					channel = a.Privates.FindByID(ch)
				}

				if channel != nil {
					a.SwitchChannel(channel)
				}
			},
		})
	})

	return nil
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

func (a *Application) setGridCol(w gtk.IWidget, n int) {
	setGridCol(a.Grid, a.cols, w, n)
}
func (a *Application) setLeftGridCol(w gtk.IWidget, n int) {
	setGridCol(a.LeftGrid, a.leftCols, w, n)
}

func newSeparator() *gtk.Separator {
	s, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	s.Show()
	return s
}

func setGridCol(grid *gtk.Grid, gridStore map[int]gtk.IWidget, w gtk.IWidget, n int) {
	if w, ok := gridStore[n]; ok {
		grid.Remove(w)
	}
	gridStore[n] = w
	grid.Attach(w, n, 0, 1, 1)
}
