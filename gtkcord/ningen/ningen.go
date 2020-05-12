package ningen

import (
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
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

type State struct {
	*ningen.State
	MemberList *MemberListState

	gmu    sync.Mutex
	guilds map[discord.Snowflake]*guildState
}

func Connect(token string, onReady func(s *State)) (*State, error) {
	store := state.NewDefaultStore(&state.DefaultStoreOptions{
		MaxMessages: 50,
	})

	s, err := state.NewWithStore(token, store)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new Discord session")
	}

	// Disable retries:
	s.Retries = 1

	n, err := ningen.FromState(s)
	if err != nil {
		return nil, errors.Wrap(err, "Faield to create ningen")
	}

	state := &State{
		State:      n,
		MemberList: NewMemberListState(n),
	}

	n.AddHandler(func(r *gateway.ReadyEvent) {
		onReady(state)
	})
	n.AddHandler(func(r *gateway.ResumedEvent) {
		onReady(state)
	})
	n.AddHandler(func(ev *gateway.GuildMemberListUpdate) {
		for _, op := range ev.Ops {
			items := append(op.Items, op.Item)

			for _, it := range items {
				if it.Member == nil {
					continue
				}

				s.Store.MemberSet(ev.GuildID, &it.Member.Member)
				s.Store.PresenceSet(ev.GuildID, &it.Member.Presence)

				// If the user is the current user, then we store a copy with no
				// guild. This is useful since popup.Hamburger reads this.
				if it.Member.User.ID == s.Ready.User.ID {
					s.Store.PresenceSet(0, &it.Member.Presence)
				}
			}
		}
	})
	n.AddHandler(func(c *gateway.GuildMembersChunkEvent) {
		state.gmu.Lock()
		defer state.gmu.Unlock()

		gd := state.getGuild(c.GuildID)

		for _, m := range c.Members {
			delete(gd.requestingMembers, m.User.ID)
		}
	})

	if err := state.Open(); err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Discord")
	}

	return state, nil
}
