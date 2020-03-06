package ningen

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/log"
)

type guildState struct {
	subscribed        bool
	requestingMembers map[discord.Snowflake]struct{}

	lastRequested time.Time
}

func (n *State) getGuild(id discord.Snowflake) *guildState {
	if n.guilds == nil {
		n.guilds = map[discord.Snowflake]*guildState{}
	}

	gd, ok := n.guilds[id]
	if !ok {
		gd = &guildState{
			requestingMembers: map[discord.Snowflake]struct{}{},
		}
		n.guilds[id] = gd
	}
	return gd
}

func (n *State) SearchMember(guildID discord.Snowflake, prefix string) {
	n.gmu.Lock()
	defer n.gmu.Unlock()

	gd := n.getGuild(guildID)

	if time.Now().Before(gd.lastRequested) {
		return
	}

	gd.lastRequested = time.Now().Add(time.Second)

	go func() {
		err := n.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
			GuildID:   []discord.Snowflake{guildID},
			Query:     prefix,
			Presences: true,
		})

		if err != nil {
			log.Errorln("Failed to request guild members for completion:", err)
		}
	}()
}

func (n *State) RequestMember(guildID, memID discord.Snowflake) {
	n.gmu.Lock()
	defer n.gmu.Unlock()

	gd := n.getGuild(guildID)
	if _, ok := gd.requestingMembers[memID]; ok {
		return
	}

	// temp unlock
	n.gmu.Unlock()

	err := n.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
		GuildID:   []discord.Snowflake{guildID},
		UserIDs:   []discord.Snowflake{memID},
		Presences: true,
	})

	// relock
	n.gmu.Lock()

	if err != nil {
		log.Errorln("Failed to request guild members:", err)
	}

	gd.requestingMembers[memID] = struct{}{}
	return
}

func (n *State) Subscribe(guildID discord.Snowflake) {
	n.gmu.Lock()
	defer n.gmu.Unlock()

	gd := n.getGuild(guildID)
	if gd.subscribed {
		return
	}

	// temp unlock
	n.gmu.Unlock()

	// subscribe
	err := n.Gateway.GuildSubscribe(gateway.GuildSubscribeData{
		GuildID:    guildID,
		Typing:     true,
		Activities: true,
	})

	// relock
	n.gmu.Lock()

	if err != nil {
		log.Errorln("Failed to subscribe:", err)
		return
	}

	gd.subscribed = true
}
