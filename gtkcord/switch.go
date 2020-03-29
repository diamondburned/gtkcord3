package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

// SwitchToID returns true if it can find the channel.
func (a *Application) SwitchToID(ch, guild discord.Snowflake) bool {
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
		return true
	}

	return false
}

// SwitchLastChannel, nil for DM.
func (a *Application) SwitchLastChannel(g *guild.Guild) {
	if g == nil {
		c, ok := a.Privates.Channels[a.lastAccess(0, 0).String()]
		if ok {
			semaphore.IdleMust(func() {
				a.Privates.List.SelectRow(c.ListBoxRow)
			})
			a.SwitchChannel(c)
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
		semaphore.IdleMust(func() {
			a.Channels.ChList.SelectRow(lastCh.Row)
		})
		a.SwitchChannel(lastCh)
	}
}

func (a *Application) FocusMessages() {
	semaphore.IdleMust(func() {
		// Set the default visible widget to the right container:
		a.Main.SetVisibleChild(a.Right)
		a.Header.SetVisibleChild(a.Header.RightSide)

		// Grab the message input's focus:
		a.Messages.Focus()
	})
}

func (a *Application) SwitchGuild(g *guild.Guild) {
	a.changeCol(columnChange{
		Widget: a.Channels,
		Width:  channel.ChannelsWidth,
		Checker: func() bool {
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
			semaphore.IdleMust(a.Header.UpdateGuild, g.Name)
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
			return a.Messages.GetGuildID().Valid()
		},
		Setter: func(w gtk.IWidget) {
			a.setLeftGridCol(w, 2)
		},
		Before: func() {
			cleanup(a.Channels, a.Privates, a.Messages, a.Header.ChMenuBtn)
		},
		Loader: func() bool {
			a.Privates.LoadChannels(a.State.Ready.PrivateChannels)
			return true
		},
		After: func() {
			semaphore.IdleMust(a.Header.UpdateGuild, "Private Messages")
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
		Checker: func() bool {
			return a.Messages.GetChannelID() != ch.ChannelID()
		},
		Setter: func(w gtk.IWidget) {
			a.Right.Add(w)
		},
		Before: func() {
			a.Messages.Cleanup()
		},
		Loader: func() bool {
			if err := a.Messages.Load(ch.ChannelID()); err != nil {
				log.Errorln("Failed to load messages:", err)
				return false
			}
			return true
		},
		After: func() {
			a.lastAccess(ch.GuildID(), a.Messages.GetChannelID())

			name, _ := ch.ChannelInfo()

			semaphore.IdleMust(func() {
				a.Header.UpdateChannel(name)
				window.SetTitle("#" + name + " - gtkcord")

				// Show the channel menu if we're in a guild:
				if a.Messages.GetGuildID().Valid() {
					a.Header.ChMenuBtn.SetRevealChild(true)
				}

				// Always scroll to bottom:
				a.Messages.ScrollToBottom()
			})
		},
	})
}

type columnChange struct {
	Widget  gtkutils.ExtendedWidget
	Width   int
	Checker func() bool            // true == switch
	Setter  func(wnew gtk.IWidget) // thread-safe
	Before  func()
	Loader  func() bool
	After   func() // only if succeed
}

func (a *Application) changeCol(c columnChange) {
	// Check if busy, prevents a deadlock in the main thread:
	if a.busy.IsBusy() {
		return
	}

	// Lock and blur in the same thread so less race conditions occur:
	semaphore.IdleMust(func() {
		// We disable input first so no events can queue up:
		a.LeftGrid.SetSensitive(false)

		// Then we acquire the lock:
		a.busy.Lock()
	})

	// We defer in a similar way, but we do the opposite:
	defer semaphore.IdleMust(func() {
		// Unlock first, so incoming events don't lock:
		a.busy.Unlock()

		// Then allow events:
		a.LeftGrid.SetSensitive(true)
	})

	if !c.Checker() {
		c.After()
		return
	}

	// Clean up channels
	c.Before()

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
		log.Println("Running setter.")
		c.Setter(c.Widget)
		log.Println("Setter ran.")
		c.Widget.Show()
	})

	c.After()
}

func cleanup(cleaners ...interface{ Cleanup() }) {
	for _, cleaner := range cleaners {
		cleaner.Cleanup()
	}
}
