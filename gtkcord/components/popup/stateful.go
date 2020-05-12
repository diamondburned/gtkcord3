package popup

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/handler"
	"github.com/gotk3/gotk3/gtk"
)

type StatefulPopupBody struct {
	*UserPopupBody
	state *ningen.State
	stop  bool

	ParentStyle *gtk.StyleContext
	parentClass string

	User  discord.Snowflake
	Guild discord.Snowflake

	Prefetch *discord.User

	stateHandlers interface {
		handler.AddHandler
		handler.Unbinder
	}
}

func NewStatefulPopupBody(s *ningen.State, user, guild discord.Snowflake) *StatefulPopupBody {
	return statefulPopupUser(&StatefulPopupBody{
		state: s,
		User:  user,
		Guild: guild,
	})
}

func NewStatefulPopupUser(s *ningen.State, user discord.User, guild discord.Snowflake) *StatefulPopupBody {
	return statefulPopupUser(&StatefulPopupBody{
		state:    s,
		User:     user.ID,
		Guild:    guild,
		Prefetch: &user,
	})
}

func statefulPopupUser(body *StatefulPopupBody) *StatefulPopupBody {
	body.stateHandlers = handler.NewRepository(body.state)
	body.UserPopupBody = NewUserPopupBody()
	go body.initialize()
	return body
}

// must be thread-safe, function is running in a goroutine
func (s *StatefulPopupBody) initialize() {
	if !s.User.Valid() {
		return
	}

	if s.Prefetch == nil {
		u, err := s.state.User(s.User)
		if err != nil {
			log.Errorln("Failed to get user:", err)
			return
		}

		s.Prefetch = u
	}

	// Update user first:
	s.idlemust(func() {
		s.UserPopupBody.Update(*s.Prefetch)
		s.UserPopupBody.Grid.ShowAll()
	})

	p, err := s.state.Presence(s.Guild, s.Prefetch.ID)
	if err != nil {
		p, err = s.state.Presence(0, s.Prefetch.ID)
	}

	if err == nil {
		s.idlemust(func() {
			s.UserPopupBody.UpdateStatus(p.Status)
			s.UpdateActivity(p.Game)
		})
	}

	s.injectHandlers()

	// fetch above presence if error not nil
	if err != nil {
		s.state.RequestMember(s.Guild, s.Prefetch.ID)
	}

	// Permit fetching member through the API.
	m, err := s.state.Member(s.Guild, s.Prefetch.ID)
	if err != nil {
		// If no member:
		return
	}

	s.idlemust(func() {
		s.UserPopupBody.UpdateMember(*m)
		s.UserPopupBody.Grid.ShowAll()
	})

	r, err := NewUserPopupRoles(s.state, s.Guild, m.RoleIDs)
	if err != nil {
		log.Errorln("Failed to get roles:", err)
		return
	}

	s.idlemust(func() {
		s.UserPopupBody.Attach(r, 2)
		s.UserPopupBody.Grid.ShowAll()
	})
}

func (s *StatefulPopupBody) injectHandlers() {
	if s.Guild.Valid() {
		s.stateHandlers.AddHandler(func(g *gateway.PresenceUpdateEvent) {
			if s.Guild.Valid() && g.User.ID == s.User {
				s.idlemust(func() {
					s.UserPopupBody.UpdateMemberPart(g.Nick, g.User)
					s.UserPopupBody.UpdateStatus(g.Status)
					s.UpdateActivity(g.Game)
				})
			}
			// TODO: roles
		})
		s.stateHandlers.AddHandler(func(g *gateway.GuildMembersChunkEvent) {
			if g.GuildID != s.Guild {
				return
			}
			for _, m := range g.Members {
				if m.User.ID == s.User {
					s.idlemust(func() {
						s.UserPopupBody.UpdateMember(m)
					})
				}
			}
		})
	}

	// only add this event if the user is yourself
	if s.User == s.state.Ready.User.ID {
		s.stateHandlers.AddHandler(func(g *gateway.SessionsReplaceEvent) {
			p, err := s.state.Presence(s.Guild, s.User)
			if err != nil {
				return
			}

			s.idlemust(func() {
				s.UserPopupBody.UpdateStatus(p.Status)
				s.UpdateActivity(p.Game)
			})
		})
	}
}

func (s *StatefulPopupBody) UpdateActivity(a *discord.Activity) {
	// Update this first
	s.UserPopupBody.UpdateActivity(a)

	// Try and update the parent
	if s.ParentStyle == nil {
		return
	}

	if a == nil {
		s.setParentClass("")
		return
	}

	switch a.Type {
	case discord.GameActivity:
		s.setParentClass("game")
	case discord.ListeningActivity:
		s.setParentClass("spotify")
	case discord.StreamingActivity:
		s.setParentClass("twitch")
	default:
		s.setParentClass("")
	}
}

func (s *StatefulPopupBody) Destroy() {
	s.stateHandlers.Unbind()
	s.UserPopupBody.Destroy()
}

func (s *StatefulPopupBody) idlemust(fn func()) {
	semaphore.IdleMust(func() {
		if !s.stop {
			fn()
		}
	})
}

// thread-safe
func (s *StatefulPopupBody) setParentClass(class string) {
	gtkutils.DiffClassUnsafe(&s.parentClass, class, s.ParentStyle)
}
