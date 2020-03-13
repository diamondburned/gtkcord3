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
	// <item> <separator> <item> <separator> <item> <separator> <item>
	//  0      1           2      3           4      5           6

	// Application states
	Header   *header.Header
	Guilds   *guild.Guilds
	Privates *channel.PrivateChannels
	Channels *channel.Channels
	Messages *message.Messages

	// GuildID -> ChannelID; if GuildID == 0 then DM
	LastAccess map[discord.Snowflake]discord.Snowflake
	lastAccMut sync.Mutex

	busy sync.Mutex
}

// New is not thread-safe.
func New() (*Application, error) {
	var a = &Application{
		cols:       map[int]gtk.IWidget{},
		LastAccess: map[discord.Snowflake]discord.Snowflake{},
	}

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
	if w, ok := a.cols[n]; ok {
		a.Grid.Remove(w)
	}
	a.cols[n] = w
	a.Grid.Attach(w, n, 0, 1, 1)
}

func (a *Application) Ready(s *ningen.State) error {
	a.State = s

	// Set gateway error functions to our own:
	s.Gateway.ErrorLog = func(err error) {
		log.Debugln("Discord error:", err)
	}

	semaphore.IdleMust(window.Resize, 1200, 850)
	window.Window.Closer = func() {
		s.Close()
	}

	// Set Markdown's highlighting theme
	switch s.Ready.Settings.Theme {
	case "dark":
		md.ChangeStyle("monokai")
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
	g.OnSelect = a.SwitchGuild
	g.DMButton.OnClick = a.SwitchDM

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

	a.Header = h
	a.Guilds = g
	a.Channels = c
	a.Privates = p
	a.Messages = m

	semaphore.IdleMust(func() {
		// Set the Guilds view to the grid
		a.setCol(a.Guilds, 0)

		// Set sizes
		a.Channels.SetSizeRequest(ChannelWidth, -1)

		// Add in separators
		a.setCol(newSeparator(), 1)
		a.setCol(newSeparator(), 3)

		// Display the grid and header
		window.Display(a.Grid)
		window.HeaderDisplay(h)

		// Show everything
		window.ShowAll()
	})

	return nil
}

func (a *Application) SwitchGuild(g *guild.Guild) {
	a.changeCol(a.Channels, 2, channel.ChannelsWidth, func() func() bool {
		a.Channels.Cleanup()
		a.Privates.Cleanup()
		a.Messages.Cleanup()

		return func() bool {
			err := a.Channels.LoadGuild(g.ID)
			if err != nil {
				log.Errorln("Failed to load guild:", err)
				return false
			}

			a.Header.UpdateGuild(g.Name)
			return true
		}
	})

	chID := a.lastAccess(g.ID, 0)
	if !chID.Valid() {
		return
	}

	for _, ch := range a.Channels.Channels {
		if ch.ID != chID {
			continue
		}

		semaphore.Async(a.Channels.ChList.SelectRow, ch.Row)
		a.SwitchChannel(ch)
		return
	}
}

func (a *Application) SwitchDM() {
	a.changeCol(a.Privates, 2, channel.ChannelsWidth, func() func() bool {
		a.Channels.Cleanup()
		a.Privates.Cleanup()
		a.Messages.Cleanup()

		return func() bool {
			a.Privates.LoadChannels(a.State.Ready.PrivateChannels)
			a.Header.UpdateGuild("Private Messages")
			return true
		}
	})

	c, ok := a.Privates.Channels[a.lastAccess(0, 0).String()]
	if ok {
		semaphore.Async(a.Privates.List.SelectRow, c.ListBoxRow)
		a.SwitchChannel(c)
	}
}

type Channel interface {
	GuildID() discord.Snowflake
	ChannelID() discord.Snowflake
	ChannelInfo() (name, topic string)
}

func (a *Application) SwitchChannel(ch Channel) {
	a.changeCol(a.Messages, 4, -1, func() func() bool {
		a.Messages.Cleanup()

		return func() bool {
			err := a.Messages.Load(ch.ChannelID())
			if err != nil {
				log.Errorln("Failed to load messages:", err)
				return false
			}

			a.lastAccess(ch.GuildID(), a.Messages.GetChannelID())
			a.Header.UpdateChannel(ch.ChannelInfo())
			return true
		}
	})

	// Grab the message input's focus:
	semaphore.IdleMust(a.Messages.Focus)
}

func (a *Application) changeCol(
	w gtkutils.ExtendedWidget, n int,
	spinnerWidth int,
	cleanup func() func() bool) {

	// Lock
	a.busy.Lock()
	defer a.busy.Unlock()

	// Clean up channels
	fn := cleanup()

	// Blur the grid
	semaphore.IdleMust(a.Grid.SetSensitive, false)
	defer semaphore.IdleMust(a.Grid.SetSensitive, true)

	// Add a spinner here
	semaphore.IdleMust(func() {
		spinner, _ := animations.NewSizedSpinner(SpinnerSize)
		spinner.SetSizeRequest(spinnerWidth, -1)
		a.setCol(spinner, n)
	})

	if !fn() {
		semaphore.IdleMust(func() {
			sadface, _ := animations.NewSizedSadFace()
			sadface.SetSizeRequest(spinnerWidth, -1)
			a.setCol(sadface, n)
		})

		return
	}

	// Replace the spinner with the actual channel:
	semaphore.IdleMust(func() {
		a.setCol(w, n)
		w.ShowAll()
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
	return s
}
