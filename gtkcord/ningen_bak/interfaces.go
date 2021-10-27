package ningen

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
)

type Presencer interface {
	Presence(guild, user discord.Snowflake) (*discord.Presence, error)
}

var _ Presencer = (*State)(nil)

type GuildRequester interface {
	RequestGuildMembers(gateway.RequestGuildMembersData) error
	GuildSubscribe(gateway.GuildSubscribeData) error
}
