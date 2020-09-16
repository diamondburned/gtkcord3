package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

// SwitchToID returns true if it can find the channel.
func (a *Application) SwitchToID(chID discord.ChannelID, guildID discord.GuildID) bool {
	var row *gtk.ListBoxRow

	guild, folder := a.Guilds.FindByID(guildID)

	switch {
	case folder != nil && guild != nil:
		folder.List.SelectRow(guild.ListBoxRow)
		fallthrough
	case guild != nil:
		a.Guilds.Select(guild.ListBoxRow)

		// Switch the channels view to the guild:
		a.SwitchGuild(guild)

		// Find the destination channel:
		if channel := a.Channels.FindByID((chID)); channel != nil {
			row = channel.Row
		}

	default:
		a.Guilds.Select(a.Guilds.DMButton.ListBoxRow)

		// Switch the channels away to the private ones:
		a.SwitchDM()

		// Find the destination channel:
		if channel := a.Privates.FindByID(chID); channel != nil {
			row = channel.ListBoxRow
		}
	}

	if row != nil {
		row.Activate()
		return true
	}

	return false
}

// SwitchLastChannel, nil for DM.
func (a *Application) SwitchLastChannel(g *guild.Guild) {
	if g == nil {
		c, ok := a.Privates.Channels[a.lastAccess(0, 0).String()]
		if ok {
			a.Privates.List.SelectRow(c.ListBoxRow)
			a.SwitchChannel(c)
		}

		return
	}

	var lastCh *channel.Channel

	var chID = a.lastAccess(g.ID, 0)
	if !chID.IsValid() {
		lastCh = a.Channels.First()
	} else {
		lastCh = a.Channels.FindByID(chID)
	}

	if lastCh != nil {
		a.Channels.ChList.SelectRow(lastCh.Row)
		a.SwitchChannel(lastCh)
	}
}

func (a *Application) FocusMessages() {
	// Set the default visible widget to the right container:
	a.Main.SetVisibleChild(a.Right)
	a.Header.SetVisibleChild(a.Header.RightSide)

	// Grab the message input's focus:
	a.Messages.Focus()
}

// leftIsDM returns whether or not the current view shows the direct messages.
func (a *Application) leftIsDM() bool {
	if wg, ok := a.leftCols[2]; ok {
		_, ok = wg.(*channel.PrivateChannel)
		if ok {
			return true
		}
	}
	return false
}

func (a *Application) SwitchGuild(g *guild.Guild) {
	a.changeCol(columnChange{
		Widget: a.Channels,
		Width:  channel.ChannelsWidth,
		Checker: func() bool {
			// Second column should be a DM if we're not in a guild.
			if a.leftIsDM() {
				return true
			}

			// We just check if the guild ID matches that in Messages. It
			// shouldn't.
			return a.Channels.GuildID != g.ID || a.Messages.GetGuildID() != g.ID
		},
		Setter: func(w gtk.IWidget) {
			a.setLeftGridCol(w, 2)
		},
		Before: func() {
			cleanup(a.Channels, a.Privates, a.Messages, a.Header.ChMenuBtn)
		},
		Loader: func() bool {
			if err := a.Channels.LoadGuild(g.ID); err != nil {
				log.Errorln("Failed to load guild:", err)
				return false
			}

			return true
		},
		After: func() {
			a.Header.UpdateGuild(g.Name)
		},
	})
}

func (a *Application) SwitchDM() {
	a.changeCol(columnChange{
		Widget: a.Privates,
		Width:  channel.ChannelsWidth,
		Checker: func() bool {
			// If the guildID is valid, that means the channel does have a
			// guild, so we're not in DMs.
			return a.Messages.GetGuildID().IsValid()
		},
		Setter: func(w gtk.IWidget) {
			a.setLeftGridCol(w, 2)
		},
		Before: func() {
			cleanup(a.Channels, a.Privates, a.Messages, a.Header.ChMenuBtn)
		},
		Loader: func() bool {
			if err := a.Privates.LoadChannels(); err != nil {
				log.Errorln("Failed to load Privates")
				return false
			}
			return true
		},
		After: func() {
			a.Header.UpdateGuild("Private Messages")
		},
	})
}

type Channel interface {
	GuildID() discord.GuildID
	ChannelID() discord.ChannelID
	ChannelInfo() (name, topic string)
}

func (a *Application) SwitchChannel(ch Channel) {
	a.changeCol(columnChange{
		Widget: a.Messages,
		Width:  -1,
		Checker: func() bool {
			// If left side is currently DM, then we must switch.
			return a.leftIsDM() || a.Messages.GetChannelID() != ch.ChannelID()
		},
		Setter: func(w gtk.IWidget) {
			a.Right.Add(w)
		},
		Before: func() {
			a.Messages.Cleanup()
		},
		LoaderAsync: func(c *columnChange) {
			a.Messages.Load(ch.ChannelID(), func(err error) {
				if err != nil {
					log.Errorln("Failed to load messages:", err)
					c.Fail()
				} else {
					c.Pass()
				}
			})
		},
		After: func() {
			a.lastAccess(ch.GuildID(), a.Messages.GetChannelID())

			name, _ := ch.ChannelInfo()

			a.Header.UpdateChannel(name)
			window.SetTitle("#" + name + " - gtkcord")

			// Show the channel menu if we're in a guild:
			if a.Messages.GetGuildID().IsValid() {
				a.Header.ChMenuBtn.SetRevealChild(true)
			}
		},
	})
}

// EVERYTHING IS THREAD-SAFE AND WILL BLOCK UI!!!!
type columnChange struct {
	Widget      gtkutils.ExtendedWidget
	Width       int
	Checker     func() bool // true == switch
	Setter      func(wnew gtk.IWidget)
	Before      func()
	Loader      func() bool
	LoaderAsync func(c *columnChange)
	After       func() // only if succeed

	// non-nil if LoaderAsync != nil
	unblur interface{ SetSensitive(bool) }
}

func (c *columnChange) Fail() {
	sadface, _ := animations.NewSizedSadFace()
	sadface.SetSizeRequest(c.Width, -1)
	c.Setter(sadface)
	c.final()
}

func (c *columnChange) Pass() {
	c.Setter(c.Widget)
	c.Widget.Show()
	c.After()
	c.final()
}

func (c *columnChange) final() {
	// Just in case. This will only do something if LoaderAsync is ran.
	if c.unblur != nil {
		c.unblur.SetSensitive(true)
	}
}

func (a *Application) changeCol(c columnChange) {
	// Check if busy, prevents a deadlock in the main thread:
	if a.busy.IsBusy() {
		return
	}

	a.busy.Lock()
	defer a.busy.Unlock()

	if !c.Checker() {
		c.After()
		return
	}

	// Clean up channels
	c.Before()

	// See if it's possible to load asynchronously:
	if c.LoaderAsync != nil {
		// Deactivate the entire window.
		c.unblur = a.Main
		c.unblur.SetSensitive(false)

		go c.LoaderAsync(&c)
		return
	}

	if !c.Loader() {
		c.Fail()
	} else {
		c.Pass()
	}
}

func cleanup(cleaners ...interface{ Cleanup() }) {
	for _, cleaner := range cleaners {
		cleaner.Cleanup()
	}
}
