package ningen

import (
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/arikawa/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/utils/wsutil"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen"
	"github.com/pkg/errors"
)

func init() {
	gateway.Presence = &gateway.UpdateStatusData{
		Status: discord.OnlineStatus,
	}
	wsutil.WSTimeout = 5 * time.Second
	wsutil.WSDebug = func(v ...interface{}) {
		log.Debugln(v...)
	}
}

type State = ningen.State

func Connect(token string, onReady func(s *State)) (*State, error) {
	store := state.NewDefaultStore(&state.DefaultStoreOptions{
		MaxMessages: 50,
	})

	s, err := state.NewWithStore(token, store)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new Discord session")
	}

	s.AddHandler(func(r *gateway.ReadyEvent) {
		for _, g := range r.Guilds {
			if g.Name == "nixhub" {
				for _, ch := range g.Channels {
					log.Println("Channel nane:", ch.Name)
				}
			}
		}
	})

	s.OnRequest = append(s.OnRequest, func(r httpdriver.Request) error {
		log.Println("Request:", r.GetPath())
		return nil
	})

	// Disable retries:
	s.Retries = 1

	n, err := ningen.FromState(s)
	if err != nil {
		return nil, errors.Wrap(err, "Faield to create ningen")
	}

	n.AddHandler(func(r *gateway.ReadyEvent) {
		onReady(n)
	})
	n.AddHandler(func(r *gateway.ResumedEvent) {
		onReady(n)
	})

	if err := n.Open(); err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Discord")
	}

	return n, nil
}

type Presencer interface {
	Presence(guild, user discord.Snowflake) (*discord.Presence, error)
}

var _ Presencer = (*State)(nil)

type GuildRequester interface {
	RequestGuildMembers(gateway.RequestGuildMembersData) error
	GuildSubscribe(gateway.GuildSubscribeData) error
}

func EmojiString(e *discord.Emoji) string {
	if e == nil {
		return ""
	}

	var emoji = e.Name
	if e.ID.Valid() { // if the emoji is custom:
		emoji = ":" + emoji + ":"
	}

	return emoji
}
