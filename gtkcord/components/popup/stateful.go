package popup

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
)

type StatefulPopupBody struct {
	*UserPopupBody
	state *ningen.State

	User  discord.Snowflake
	Guild discord.Snowflake

	unhandlers []func()
}

func NewStatefulPopupBody(s *ningen.State, user, guild discord.Snowflake) *StatefulPopupBody {
	b := NewUserPopupBody()

	body := &StatefulPopupBody{
		UserPopupBody: b,

		state: s,
		User:  user,
		Guild: guild,
	}
	go body.initialize()

	b.Connect("destroy", func() {
		log.Infoln("Destroying stateful popup body")
		body.Destroy()
	})

	return body
}

// must be thread-safe, function is running in a goroutine
func (s *StatefulPopupBody) initialize() {
	defer s.injectHandlers()

	u, err := s.state.User(s.User)
	if err != nil {
		log.Errorln("Failed to get user:", err)
		return
	}

	p, err := s.state.Presence(s.Guild, u.ID)
	if err == nil {
		s.UserPopupBody.UpdateStatus(p.Status)
		s.UserPopupBody.UpdateActivity(p.Game)
	}

	if !s.Guild.Valid() {
		s.UserPopupBody.Update(*u)
		return
	}

	// fetch above presence if error not nil
	if err != nil {
		s.state.RequestMember(s.Guild, u.ID)
		return
	}

	m, err := s.state.Member(s.Guild, u.ID)
	if err != nil {
		s.UserPopupBody.Update(*u)
		return
	}

	s.UserPopupBody.UpdateMember(*m)

	r, err := NewUserPopupRoles(s.state, s.Guild, m.RoleIDs)
	if err != nil {
		log.Errorln("Failed to get roles:", err)
		return
	}

	semaphore.IdleMust(s.UserPopupBody.Box.Add, r)
	semaphore.IdleMust(s.UserPopupBody.Box.ShowAll)
}

func (s *StatefulPopupBody) injectHandlers() {
	if s.Guild.Valid() {
		s.unhandlers = append(s.unhandlers,
			s.state.AddHandler(func(g *gateway.PresenceUpdateEvent) {
				// Since PresenceUpdate is
				if !s.Guild.Valid() || g.User.ID != s.User {
					return
				}

				s.UserPopupBody.UpdateMemberPart(g.Nick, g.User)
				s.UserPopupBody.UpdateActivity(g.Game)
				s.UserPopupBody.UpdateStatus(g.Status)

				// TODO: roles
			}),
		)
	}

	// only add this event if the user is yourself
	if s.User == s.state.Ready.User.ID {
		s.unhandlers = append(s.unhandlers,
			s.state.AddHandler(func(g *gateway.SessionsReplaceEvent) {
				if len(*g) == 0 {
					s.UserPopupBody.UpdateActivity(nil)
					return
				}

				presence := (*g)[0]
				s.UserPopupBody.UpdateActivity(presence.Game)
				s.UserPopupBody.UpdateStatus(presence.Status)
			}),
		)
	}
}

func (s *StatefulPopupBody) Destroy() {
	for _, h := range s.unhandlers {
		h()
	}
	s.UserPopupBody.Destroy()
	s.UserPopupBody = nil
}
