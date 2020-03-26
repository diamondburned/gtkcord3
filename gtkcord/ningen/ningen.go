package ningen

import (
	"sync"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/pkg/errors"
)

func init() {
	gateway.Presence = &gateway.UpdateStatusData{
		Status: discord.OnlineStatus,
	}
	gateway.WSTimeout = 5 * time.Second
	gateway.WSDebug = func(v ...interface{}) {
		log.Debugln(v...)
	}
}

type State struct {
	*state.State

	mutedMutex    sync.RWMutex
	MutedGuilds   map[discord.Snowflake]*Mute
	MutedChannels map[discord.Snowflake]*Mute

	readMutex sync.RWMutex
	lastAck   api.Ack
	LastRead  map[discord.Snowflake]*gateway.ReadState

	// rs is updated
	callbackMut  sync.Mutex
	OnReadChange []func(s *State, rs *gateway.ReadState, unread bool)
	// OnGuildPosChange func(*gateway.UserSettings)

	gmu    sync.Mutex
	guilds map[discord.Snowflake]*guildState

	MemberList *MemberListState
}

type Mute struct {
	// if true, then muted
	All           bool
	Notifications int // some sort of constant?

	// guild only
	Everyone bool // @everyone
}

func Connect(token string, onReady func(s *State)) (*State, error) {
	store := state.NewDefaultStore(&state.DefaultStoreOptions{
		MaxMessages: 50,
	})

	s, err := state.NewWithStore(token, store)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a new Discord session")
	}

	ningen := ningen(s)

	s.AddHandler(func(r *gateway.ReadyEvent) {
		ningen.UpdateReady(*r)
		onReady(ningen)
	})

	// s.Gateway.OP = make(chan *gateway.OP)
	// go func() {
	// 	for ev := range s.Gateway.OP {
	// 		if ev.EventName == "READY" {
	// 			log.Println(string(ev.Data))
	// 		}
	// 	}
	// }()

	if err := s.Open(); err != nil {
		return nil, errors.Wrap(err, "Failed to connect to Discord")
	}

	return ningen, nil
}

func ningen(s *state.State) *State {
	state := &State{
		State:         s,
		MutedGuilds:   map[discord.Snowflake]*Mute{},
		MutedChannels: map[discord.Snowflake]*Mute{},
		LastRead:      map[discord.Snowflake]*gateway.ReadState{},
		guilds:        map[discord.Snowflake]*guildState{},
	}

	s.AddHandler(func(a *gateway.MessageAckEvent) {
		state.MarkRead(a.ChannelID, a.MessageID)
	})

	s.AddHandler(func(c *gateway.MessageCreateEvent) {
		if c.Author.ID == s.Ready.User.ID {
			return
		}
		var mentions int
		for _, u := range c.Mentions {
			if u.ID == s.Ready.User.ID {
				mentions++
			}
		}

		state.MarkUnread(c.ChannelID, c.ID, mentions)
	})

	s.AddHandler(func(r *gateway.ReadyEvent) {
		state.UpdateReady(*r)
	})

	s.AddHandler(func(r *gateway.UserSettingsUpdateEvent) {
		// state.OnGuildPosChange((*gateway.UserSettings)(r))
	})

	s.AddHandler(func(u *gateway.UserGuildSettingsUpdateEvent) {
		state.updateMuteState([]gateway.UserGuildSettings{
			gateway.UserGuildSettings(*u),
		})
	})

	s.AddHandler(func(c *gateway.GuildMembersChunkEvent) {
		state.gmu.Lock()
		defer state.gmu.Unlock()

		gd := state.getGuild(c.GuildID)

		for _, m := range c.Members {
			delete(gd.requestingMembers, m.User.ID)
		}
	})

	s.AddHandler(func(r *gateway.SessionsReplaceEvent) {
		s.PresenceSet(0, state.JoinSession(r))
	})

	state.MemberList = NewMemberListState()
	s.AddHandler(state.MemberList.handle)

	s.AddHandler(func(ev *gateway.GuildMemberListUpdate) {
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

	// state.UpdateReady(s.Ready)
	// state.UpdateNitroEmojis()
	return state
}

func (s *State) AddReadChange(fn func(s *State, rs *gateway.ReadState, unread bool)) {
	s.callbackMut.Lock()
	defer s.callbackMut.Unlock()

	s.OnReadChange = append(s.OnReadChange, fn)
}

func (s *State) JoinSession(r *gateway.SessionsReplaceEvent) *discord.Presence {
	ses := *r

	var game *discord.Activity
	var status discord.Status
	var activities []discord.Activity

	for i := len(ses) - 1; i >= 0; i-- {
		presence := ses[i]

		if presence.Game != nil {
			game = presence.Game
		}
		if presence.Status != "" {
			status = presence.Status
		}

		activities = append(activities, presence.Activities...)
	}

	if game == nil && len(activities) > 0 {
		game = &activities[len(activities)-1]
	}

	return &discord.Presence{
		User:       s.Ready.User,
		Game:       game,
		Status:     status,
		Activities: activities,
	}
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

func (s *State) FindLastRead(channelID discord.Snowflake) *gateway.ReadState {
	if s.ChannelMuted(channelID) {
		return nil
	}

	s.readMutex.RLock()
	defer s.readMutex.RUnlock()

	if s, ok := s.LastRead[channelID]; ok {
		return s
	}

	return nil
}

func (s *State) MarkUnread(chID, msgID discord.Snowflake, mentions int) {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()
	// log.Debugln(log.Trace(0), "MarkUnread")

	// Check for a ReadState
	st, ok := s.LastRead[chID]
	if !ok {
		st = &gateway.ReadState{
			ChannelID: chID,
		}
		s.LastRead[chID] = st
	}
	// Update ReadState
	// st.LastMessageID = msgID
	st.MentionCount += mentions

	// Update the channel state:
	if ch, err := s.Store.Channel(chID); err == nil {
		ch.LastMessageID = msgID
		s.Store.ChannelSet(ch)
	}

	go func() {
		// Check if this is our message or not:
		if m, err := s.Store.Message(chID, msgID); err == nil {
			if m.Author.ID == s.Ready.User.ID {
				// If it is, don't mark as unread.
				return
			}
		}

		// Announce that there's a read state change
		for _, fn := range s.OnReadChange {
			fn(s, st, true)
		}
	}()
}

func (s *State) MarkRead(chID, msgID discord.Snowflake) {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()
	// log.Debugln(log.Trace(0), "MarkRead")

	// Check for a ReadState
	st, ok := s.LastRead[chID]
	if !ok {
		st = &gateway.ReadState{
			ChannelID: chID,
		}
		s.LastRead[chID] = st
	}

	if st.LastMessageID == msgID {
		return
	}

	// Update ReadState
	st.LastMessageID = msgID
	st.MentionCount = 0

	// Announce that there's a read state change
	for _, fn := range s.OnReadChange {
		fn(s, st, false)
	}

	go func() {
		// Check if this is our message or not:
		if m, err := s.Store.Message(chID, msgID); err == nil {
			if m.Author.ID == s.Ready.User.ID {
				// If it is, don't Ack.
				return
			}
		}

		// Send over Ack.
		if err := s.Ack(chID, msgID, &s.lastAck); err != nil {
			log.Errorln("Failed to ack message:", err)
		}
	}()
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
	s.mutedMutex.RLock()
	defer s.mutedMutex.RUnlock()

	if m, ok := s.MutedChannels[channelID]; ok {
		// Channels don't have an @everyone mute.
		return m.All
	}

	return false
}

func (s *State) GuildMuted(guildID discord.Snowflake, everyone bool) bool {
	s.mutedMutex.RLock()
	defer s.mutedMutex.RUnlock()

	m, ok := s.MutedGuilds[guildID]
	if ok {
		return (!everyone && m.All) || (everyone && m.Everyone)
	}
	return false
}

type GuildEmojis struct {
	Name   string
	Emojis []discord.Emoji
}

func (s *State) SearchEmojis(guildID discord.Snowflake) []GuildEmojis {
	// User doesn't have Nitro, so only non-GIF guild emojis are available:
	if s.Ready.User.Nitro == discord.NoUserNitro {
		// If we don't have a guildID, return nothing.
		if !guildID.Valid() {
			return nil
		}

		g, err := s.Store.Guild(guildID)
		if err != nil {
			log.Errorln("Failed to get guild while searching emojis:", err)
		}

		emojis, err := s.Store.Emojis(guildID)
		if err != nil {
			log.Errorln("Failed to get emojis:", err)
			return nil
		}

		filtered := emojis[:0]

		for _, e := range emojis {
			if e.Animated == false {
				filtered = append(filtered, e)
			}
		}

		return []GuildEmojis{{
			Name:   g.Name,
			Emojis: emojis,
		}}
	}

	// User has Nitro, grab all emojis.
	guilds, err := s.Store.Guilds()
	if err != nil {
		log.Errorln("Failed to get guilds:", err)
		return nil
	}

	var emojis = make([]GuildEmojis, 0, len(guilds))

	for _, g := range guilds {
		if e, err := s.Store.Emojis(g.ID); err == nil {
			emojis = append(emojis, GuildEmojis{
				Name:   g.Name,
				Emojis: e,
			})
		}
	}

	return emojis
}
