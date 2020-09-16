package moreatomic

import (
	"sync/atomic"

	"github.com/diamondburned/arikawa/discord"
)

type Snowflake uint64

func (s *Snowflake) Get() discord.Snowflake {
	return discord.Snowflake(atomic.LoadUint64((*uint64)(s)))
}

func (s *Snowflake) Set(id discord.Snowflake) {
	atomic.StoreUint64((*uint64)(s), uint64(id))
}

type ChannelID Snowflake

func (c *ChannelID) Get() discord.ChannelID {
	return discord.ChannelID(atomic.LoadUint64((*uint64)(c)))
}

func (c *ChannelID) Set(id discord.ChannelID) {
	atomic.StoreUint64((*uint64)(c), uint64(id))
}

type GuildID Snowflake

func (g *GuildID) Get() discord.GuildID {
	return discord.GuildID(atomic.LoadUint64((*uint64)(g)))
}

func (g *GuildID) Set(id discord.GuildID) {
	atomic.StoreUint64((*uint64)(g), uint64(id))
}
