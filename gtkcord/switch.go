package gtkcord

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/channel"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/internal/log"
)

// SwitchToID returns true if it can find the channel.
func (a *Application) SwitchToID(chID discord.ChannelID, guildID discord.GuildID) bool {
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
			a.Channels.ChList.SelectRow(channel.Row)
			return true
		}

	default:
		a.Guilds.Select(a.Guilds.DMButton.ListBoxRow)

		// Switch the channels away to the private ones:
		a.SwitchDM()

		// Find the destination channel:
		if channel := a.Privates.FindByID(chID); channel != nil {
			a.Privates.List.SelectRow(channel.ListBoxRow)
			return true
		}
	}

	return false
}

func (a *Application) FocusMessages() {
	if a.Main.Folded() {
		// unreveal the flap if we're folded
		a.Main.SetRevealFlap(false)
	}

	// Grab the message input's focus:
	a.Messages.Focus()
}

// leftIsDM returns whether or not the current view shows the direct messages.
func (a *Application) leftIsDM() bool {
	if wg := a.leftCols[channelsColumn]; wg != nil {
		_, ok := wg.(*channel.PrivateChannels)
		return ok
	}
	return false
}

func (a *Application) leftIsGuild() bool {
	if wg := a.leftCols[channelsColumn]; wg != nil {
		_, ok := wg.(*channel.Channels)
		return ok
	}
	return false
}

// GuildID gets the application's guild ID. It enforces the internal application
// state to be the same.
func (a *Application) GuildID() discord.GuildID {
	if gID := a.Messages.GuildID(); gID.IsValid() && gID != a.Channels.GuildID {
		log.Panicf("mismatch mesasge guild (%d) and channels guild (%d)", gID, a.Channels.GuildID)
	}
	if !a.leftIsGuild() {
		return 0
	}
	return a.Channels.GuildID
}

// ChannelID gets the application's channel ID. The same enforcement applies.
func (a *Application) ChannelID() discord.ChannelID {
	chID := a.Messages.ChannelID()
	if !chID.IsValid() {
		return 0
	}
	return chID
}

// prepChannelSwitch does a cleanup on multiple components in preparation for
// channel switching.
func (a *Application) prepChannelSwitch() {
	type cleaner interface{ Cleanup() }

	cleaners := []cleaner{
		a.Channels,
		a.Privates,
		a.Messages,
		a.Header,
		a.Header.ChMenuBtn,
	}

	for _, cleaner := range cleaners {
		cleaner.Cleanup()
	}
}

func (a *Application) SwitchGuild(g *guild.Guild) {
	if a.leftIsGuild() && a.GuildID() == g.ID {
		return
	}

	a.prepChannelSwitch()
	a.Channels.LoadGuild(g.ID)

	a.setLeftGridCol(a.Channels, channelsColumn)
	a.Header.UpdateGuild(g.Name)
}

func (a *Application) SwitchDM() {
	if a.leftIsDM() {
		return
	}

	a.prepChannelSwitch()
	a.Privates.Load()

	a.setLeftGridCol(a.Privates, channelsColumn)
	a.Header.UpdateGuild("Private Messages")
}

type ChannelContainer interface {
	GuildID() discord.GuildID
	ChannelID() discord.ChannelID
	ChannelInfo() (name, topic string)
}

func (a *Application) SwitchChannel(ch ChannelContainer) {
	if a.ChannelID() == ch.ChannelID() {
		return
	}

	a.Messages.Cleanup()
	a.Messages.Load(ch.ChannelID())

	a.Right.SetChild(a.Messages)

	name, _ := ch.ChannelInfo()
	if ch.GuildID().IsValid() {
		name = "#" + name
	}

	a.Header.UpdateChannel(name)
	window.SetTitle(name + " - gtkcord")

	// Show the channel menu if we're in a guild:
	if ch.GuildID().IsValid() {
		a.Header.ChMenuBtn.SetRevealChild(true)
	}
}
