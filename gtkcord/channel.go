package gtkcord

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/gotk3/gotk3/gtk"
)

type Channels struct {
	*gtk.ListBox

	// Headers
	Header *gtk.Box
	Name   *gtk.Label
	Banner *gtk.Image // nil

	// Channel list
	Channels []*Channel

	GuildID discord.Snowflake
}

type Channel struct {
}

/*
func NewChannels(s *state.State, guildID discord.Guild) (*Channels, error) {
	discordChannels, err := s.Channels(guildID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get channels")
	}
}
*/
