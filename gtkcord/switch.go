package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

func (a *Application) SwitchGuild(g *guild.Guild) {
	a.changeCol(columnChange{
		Widget: a.Channels,
		Width:  channel.ChannelsWidth,
		Setter: func(wold, wnew gtk.IWidget) {
			a.LeftGrid.Remove(wold)
			a.LeftGrid.Attach(wnew, 2, 0, 1, 1)
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
			var lastCh *channel.Channel

			var chID = a.lastAccess(g.ID, 0)
			if !chID.Valid() {
				lastCh = a.Channels.First()
			} else {
				lastCh = a.Channels.FindByID(chID)
			}

			if lastCh != nil {
				semaphore.Async(a.Channels.ChList.SelectRow, lastCh.Row)
			}

			a.busy.Lock()
			defer a.busy.Unlock()

			if err := a.Members.LoadGuild(g.ID); err != nil {
				log.Println("Can't load members:", err)
				return
			}
			a.Grid.Attach(a.Members, 2, 0, 1, 1)
		},
	})
}

func (a *Application) SwitchDM() {
	a.changeCol(columnChange{
		Widget: a.Privates,
		Width:  channel.ChannelsWidth,
		Setter: func(wold, wnew gtk.IWidget) {
			a.LeftGrid.Remove(wold)
			a.LeftGrid.Attach(wnew, 2, 0, 1, 1)
		},
		Cleaner: func() {
			cleanup(a.Channels, a.Privates, a.Messages, a.Members)
			semaphore.IdleMust(a.Grid.Remove, a.Members)
		},
		Loader: func() bool {
			a.Privates.LoadChannels(a.State.Ready.PrivateChannels)
			a.Header.UpdateGuild("Private Messages")
			return true
		},
		After: func() {
			c, ok := a.Privates.Channels[a.lastAccess(0, 0).String()]
			if ok {
				semaphore.IdleMust(a.Privates.List.SelectRow, c.ListBoxRow)
			}
		},
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
		Setter: func(wold, wnew gtk.IWidget) {
			a.Grid.Remove(wold)
			a.Grid.Attach(wnew, 1, 0, 1, 1)
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
			a.Header.UpdateChannel(ch.ChannelInfo())
			return true
		},
		After: func() {
			// Grab the message input's focus:
			semaphore.IdleMust(a.Messages.Focus)
		},
	})
}

type columnChange struct {
	Widget  gtkutils.ExtendedWidget
	Width   int
	Setter  func(wold, wnew gtk.IWidget) // thread-safe
	Cleaner func()
	Loader  func() bool
	After   func() // only if succeed
}

func (a *Application) changeCol(c columnChange) {
	// Lock
	a.busy.Lock()
	defer a.busy.Unlock()

	// Clean up channels
	c.Cleaner()

	// Add a spinner here
	var spinner gtkutils.WidgetSizeRequester

	semaphore.IdleMust(func() {
		spinner, _ = animations.NewSizedSpinner(SpinnerSize)
		spinner.SetSizeRequest(c.Width, -1)
		c.Setter(c.Widget, spinner)

		// Blur the grid
		a.Grid.SetSensitive(false)
	})
	defer semaphore.IdleMust(a.Grid.SetSensitive, true)

	if !c.Loader() {
		semaphore.IdleMust(func() {
			sadface, _ := animations.NewSizedSadFace()
			sadface.SetSizeRequest(c.Width, -1)
			c.Setter(spinner, sadface)
		})

		return
	}

	// Replace the spinner with the actual channel:
	semaphore.IdleMust(func() {
		c.Setter(spinner, c.Widget)
		c.Widget.Show()
	})

	go c.After()
}

func cleanup(cleaners ...interface{ Cleanup() }) {
	for _, cleaner := range cleaners {
		cleaner.Cleanup()
	}
}
