package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

// func (a *Application)

func (a *Application) SwitchGuild(g *guild.Guild) {
	a.changeCol(columnChange{
		Widget: a.Channels,
		Width:  channel.ChannelsWidth,
		Checker: func() bool {
			// We just check if the guild ID matches that in Messages. It
			// shouldn't.
			return a.Messages.GuildID != g.ID
		},
		Setter: func(w gtk.IWidget) {
			a.setLeftGridCol(w, 2)
		},
		Cleaner: func() {
			cleanup(a.Channels, a.Privates, a.Messages, a.Members)
		},
		Loader: func() bool {
			if err := a.Channels.LoadGuild(g.ID); err != nil {
				log.Errorln("Failed to load guild:", err)
				return false
			}

			a.Header.UpdateGuild(g.Name)
			return true
		},
		After: func() {
			if err := a.Members.LoadGuild(g.ID); err != nil {
				log.Println("Can't load members:", err)
				return
			}
			semaphore.IdleMust(a.Right.Add, a.Members)
		},
	})
}

// SwitchLastChannel, nil for DM.
func (a *Application) SwitchLastChannel(g *guild.Guild) {
	if g == nil {
		c, ok := a.Privates.Channels[a.lastAccess(0, 0).String()]
		if ok {
			semaphore.IdleMust(a.Privates.List.SelectRow, c.ListBoxRow)
		}

		return
	}

	var lastCh *channel.Channel

	var chID = a.lastAccess(g.ID, 0)
	if !chID.Valid() {
		lastCh = a.Channels.First()
	} else {
		lastCh = a.Channels.FindByID(chID)
	}

	if lastCh != nil {
		semaphore.IdleMust(a.Channels.ChList.SelectRow, lastCh.Row)
	}
}

func (a *Application) SwitchDM() {
	a.changeCol(columnChange{
		Widget: a.Privates,
		Width:  channel.ChannelsWidth,
		Checker: func() bool {
			// If the guildID is valid, that means the channel does have a
			// guild, so we're not in DMs.
			return a.Messages.GuildID.Valid()
		},
		Setter: func(w gtk.IWidget) {
			a.setLeftGridCol(w, 2)
		},
		Cleaner: func() {
			cleanup(a.Channels, a.Privates, a.Messages, a.Members)
			semaphore.IdleMust(a.Right.Clear)
		},
		Loader: func() bool {
			a.Privates.LoadChannels(a.State.Ready.PrivateChannels)
			a.Header.UpdateGuild("Private Messages")
			return true
		},
		After: func() {},
	})
}

type Channel interface {
	GuildID() discord.Snowflake
	ChannelID() discord.Snowflake
	ChannelInfo() (name, topic string)
}

func (a *Application) SwitchChannel(ch Channel) {
	a.changeCol(columnChange{
		Widget: a.Messages,
		Width:  -1,
		Checker: func() bool {
			return a.Messages.ChannelID != ch.ChannelID()
		},
		Setter: func(w gtk.IWidget) {
			a.Middle.Add(w)
		},
		Cleaner: func() {
			a.Messages.Cleanup()
		},
		Loader: func() bool {
			if err := a.Messages.Load(ch.ChannelID()); err != nil {
				log.Errorln("Failed to load messages:", err)
				return false
			}

			a.lastAccess(ch.GuildID(), a.Messages.GetChannelID())

			name, _ := ch.ChannelInfo()
			a.Header.UpdateChannel(name)
			return true
		},
		After: func() {
			semaphore.IdleMust(func() {
				// Set the default visible widget to messages:
				a.Main.SetVisibleChild(a.Middle)

				// Grab the message input's focus:
				a.Messages.Focus()
			})
		},
	})
}

type columnChange struct {
	Widget  gtkutils.ExtendedWidget
	Width   int
	Checker func() bool            // true == switch
	Setter  func(wnew gtk.IWidget) // thread-safe
	Cleaner func()
	Loader  func() bool
	After   func() // only if succeed
}

func (a *Application) changeCol(c columnChange) {
	// Lock
	a.busy.Lock()
	defer a.busy.Unlock()

	if !c.Checker() {
		return
	}

	// Blur the entire left grid, which includes guilds and channels.
	semaphore.IdleMust(a.LeftGrid.SetSensitive, false)
	defer semaphore.IdleMust(a.LeftGrid.SetSensitive, true)

	// Clean up channels
	c.Cleaner()

	// We're not adding a spinner anymore. The message view now loads so fast a
	// spinner is practically useless and is more likely to induce epilepsy than
	// looking cool.

	// semaphore.IdleMust(func() {
	// 	spinner, _ := animations.NewSizedSpinner(SpinnerSize)
	// 	spinner.SetSizeRequest(c.Width, -1)
	// 	c.Setter(spinner)
	// })

	if !c.Loader() {
		semaphore.IdleMust(func() {
			sadface, _ := animations.NewSizedSadFace()
			sadface.SetSizeRequest(c.Width, -1)
			c.Setter(sadface)
		})

		return
	}

	// Replace the spinner with the actual channel:
	semaphore.IdleMust(func() {
		c.Setter(c.Widget)
		c.Widget.Show()
	})

	c.After()
}

func cleanup(cleaners ...interface{ Cleanup() }) {
	for _, cleaner := range cleaners {
		cleaner.Cleanup()
	}
}
