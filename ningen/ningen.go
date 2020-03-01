package ningen

import (
	"context"
	"sync"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/pkg/errors"
)

func init() {
	gateway.Presence = &gateway.UpdateStatusData{
		Status: discord.OnlineStatus,
	}
}

type State struct {
	*state.State

	mutedMutex    sync.Mutex
	MutedGuilds   map[discord.Snowflake]*Mute
	MutedChannels map[discord.Snowflake]*Mute

	readMutex sync.Mutex
	lastAck   api.Ack
	LastRead  map[discord.Snowflake]*gateway.ReadState

	OnReadChange     func(rs *gateway.ReadState, ack bool)
	OnGuildPosChange func(*gateway.UserSettings)
}

type Mute struct {
	// if true, then muted
	All           bool
	Notifications int // some sort of constant?

	// guild only
	Everyone bool // @everyone
}

func Connect(token string) (*State, error) {
	s, err := state.New(token)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new Discord session")
	}

	if err := s.Open(); err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Discord")
	}

	return Ningen(s)
}

func Ningen(s *state.State) (*State, error) {
	state := &State{
		State:            s,
		MutedGuilds:      map[discord.Snowflake]*Mute{},
		MutedChannels:    map[discord.Snowflake]*Mute{},
		LastRead:         map[discord.Snowflake]*gateway.ReadState{},
		OnReadChange:     func(*gateway.ReadState, bool) {},
		OnGuildPosChange: func(*gateway.UserSettings) {},
	}

	s.AddHandler(func(a *gateway.MessageAckEvent) {
		state.hookIncomingMessage(a.ChannelID, a.MessageID, true, true)
	})

	s.AddHandler(func(c *gateway.MessageCreateEvent) {
		state.hookIncomingMessage(c.ChannelID, c.ID, true, false)
	})

	s.AddHandler(func(r *gateway.ReadyEvent) {
		state.UpdateReady(*r)
	})

	s.AddHandler(func(r *gateway.UserSettingsUpdateEvent) {
		state.OnGuildPosChange((*gateway.UserSettings)(r))
	})

	s.AddHandler(func(u *gateway.UserGuildSettingsUpdateEvent) {
		state.updateMuteState([]gateway.UserGuildSettings{
			gateway.UserGuildSettings(*u),
		})
	})

	if s.Ready.SessionID == "" {
		s.WaitFor(context.Background(), func(v interface{}) bool {
			_, ok := v.(*gateway.ReadyEvent)
			return ok
		})
	}

	state.UpdateReady(s.Ready)
	return state, nil
}

func (s *State) UpdateReady(r gateway.ReadyEvent) {
	s.updateMuteState(r.UserGuildSettings)
	s.updateReadState(r.ReadState)
}

func (s *State) updateMuteState(ugses []gateway.UserGuildSettings) {
	// TODO: This function doesn't have any callbacks to indicate this update.
	// There should be a better way to know what to call on. This is required
	// for things like updated muting states, mainly UI changes.

	s.mutedMutex.Lock()
	defer s.mutedMutex.Unlock()

	for _, ugs := range ugses {
		mg, ok := s.MutedGuilds[ugs.GuildID]
		if !ok {
			mg = &Mute{}
			s.MutedGuilds[ugs.GuildID] = mg
		}

		mg.All = ugs.Muted
		mg.Everyone = ugs.SupressEveryone
		mg.Notifications = ugs.MessageNotifications

		for _, ch := range ugs.ChannelOverrides {
			mc, ok := s.MutedChannels[ch.ChannelID]
			if !ok {
				mc = &Mute{}
				s.MutedChannels[ch.ChannelID] = mc
			}

			mc.All = ch.Muted
			mc.Notifications = ch.MessageNotifications
		}
	}
}

func (s *State) updateReadState(rs []gateway.ReadState) {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()

	for _, read := range rs {
		s.LastRead[read.ChannelID] = &gateway.ReadState{
			ChannelID:     read.ChannelID,
			LastMessageID: read.LastMessageID,
			MentionCount:  read.MentionCount,
		}
	}
}

// returns *ReadState if updated, marks the message as unread.
func (s *State) hookIncomingMessage(channel, message discord.Snowflake, call, ack bool) bool {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()

	st, ok := s.LastRead[channel]
	if !ok {
		st = &gateway.ReadState{
			ChannelID: channel,
		}
		s.LastRead[channel] = st
	}

	st.LastMessageID = message

	if call {
		s.OnReadChange(st, ack)
	}
	return true
}

func (s *State) FindLastRead(channelID discord.Snowflake) *gateway.ReadState {
	if s.ChannelMuted(channelID) {
		return nil
	}

	s.readMutex.Lock()
	defer s.readMutex.Unlock()

	if s, ok := s.LastRead[channelID]; ok {
		return s
	}

	return nil
}

func (s *State) MarkRead(channelID, messageID discord.Snowflake) {
	// Update ReadState as well as the callback.
	if !s.hookIncomingMessage(channelID, messageID, false, false) {
		return
	}

	// Send over Ack.
	if err := s.Ack(channelID, messageID, &s.lastAck); err != nil {
		log.Errorln("Failed to ack message:", err)
	}
}

func (s *State) CategoryMuted(channelID discord.Snowflake) bool {
	ch, err := s.Store.Channel(channelID)
	if err != nil {
		return false
	}

	if !ch.CategoryID.Valid() {
		return false
	}

	return s.ChannelMuted(ch.CategoryID)
}

func (s *State) ChannelMuted(channelID discord.Snowflake) bool {
	s.mutedMutex.Lock()
	defer s.mutedMutex.Unlock()

	if m, ok := s.MutedChannels[channelID]; ok {
		// Channels don't have an @everyone mute.
		if m.All {
			return true
		}
	}

	return false
}

func (s *State) GuildMuted(guildID discord.Snowflake, everyone bool) bool {
	s.mutedMutex.Lock()
	defer s.mutedMutex.Unlock()

	m, ok := s.MutedGuilds[guildID]
	if ok {
		return (!everyone && m.All) || (everyone && m.Everyone)
	}
	return false
}
