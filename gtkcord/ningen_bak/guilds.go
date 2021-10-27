package ningen

import (
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gtkcord3/internal/log"
)

type guildState struct {
	subscribedMax     int // 0, 99, 199, 299, etc
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

	/* TODO: INSPECT ME */ go func() {
		err := n.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
			GuildID:   []discord.Snowflake{guildID},
			Query:     prefix,
			Presences: true,
			Limit:     25,
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

	gd.requestingMembers[memID] = struct{}{}

	/* TODO: INSPECT ME */ go func() {
		err := n.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
			GuildID:   []discord.Snowflake{guildID},
			UserIDs:   []discord.Snowflake{memID},
			Presences: true,
		})

		// relock
		n.gmu.Lock()
		defer n.gmu.Unlock()

		if err != nil {
			log.Errorln("Failed to request guild members:", err)
		}

		delete(gd.requestingMembers, memID)
	}()
}

func (n *State) Subscribe(guildID discord.Snowflake, chID discord.Snowflake, chunk int) {
	n.gmu.Lock()
	defer n.gmu.Unlock()

	// Round chunk to its maximum (ceiling-ed):
	chunk /= 100
	chunk++

	chunks := make([][2]int, 0, chunk)

	for i := 0; i < chunk; i++ {
		chunks = append(chunks, [2]int{
			(i * 100),      // start: 100
			(i * 100) + 99, // end:   199
		})
	}

	gd := n.getGuild(guildID)
	if gd.subscribedMax >= chunk {
		return
	}

	// temp unlock
	n.gmu.Unlock()

	// subscribe
	err := n.Gateway.GuildSubscribe(gateway.GuildSubscribeData{
		GuildID:    guildID,
		Typing:     true,
		Activities: true,
		Channels: map[discord.Snowflake][][2]int{
			chID: chunks,
		},
	})

	// relock
	n.gmu.Lock()

	if err != nil {
		log.Errorln("Failed to subscribe:", err)
		return
	}

	gd.subscribedMax = chunk
}
