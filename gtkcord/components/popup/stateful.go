package popup

import (
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

type StatefulPopupBody struct {
	*UserPopupBody
	state *ningen.State

	ParentStyle *gtk.StyleContext
	parentClass string

	User  discord.Snowflake
	Guild discord.Snowflake

	unhandlers []func()
	mutex      sync.Mutex
}

func NewStatefulPopupBody(s *ningen.State, user, guild discord.Snowflake) *StatefulPopupBody {
	b := NewUserPopupBody()

	body := &StatefulPopupBody{
		UserPopupBody: b,

		state: s,
		User:  user,
		Guild: guild,
	}

	b.Connect("destroy", func() {
		log.Infoln("Destroying stateful popup body")
		body.Destroy()
		log.Infoln("Destroyed")
	})

	go body.initialize()
	return body
}

// must be thread-safe, function is running in a goroutine
func (s *StatefulPopupBody) initialize() {
	if !s.User.Valid() {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	u, err := s.state.User(s.User)
	if err != nil {
		log.Errorln("Failed to get user:", err)
		return
	}

	p, err := s.state.Store.Presence(s.Guild, u.ID)
	if err != nil {
		p, err = s.state.Store.Presence(0, u.ID)
	}

	if err == nil {
		semaphore.IdleMust(func() {
			s.UserPopupBody.UpdateStatus(p.Status)
			s.UpdateActivity(p.Game)
		})
	}

	s.injectHandlers()

	if !s.Guild.Valid() {
		semaphore.IdleMust(func() {
			s.UserPopupBody.Update(*u)
			s.UserPopupBody.Grid.ShowAll()
		})
		return
	}

	// fetch above presence if error not nil
	if err != nil {
		s.state.RequestMember(s.Guild, u.ID)
		return
	}

	m, err := s.state.Store.Member(s.Guild, u.ID)
	if err != nil {
		semaphore.IdleMust(func() {
			s.UserPopupBody.Update(*u)
			s.UserPopupBody.Grid.ShowAll()
		})
		return
	}

	semaphore.IdleMust(func() {
		s.UserPopupBody.UpdateMember(*m)
		s.UserPopupBody.Grid.ShowAll()
	})

	r, err := NewUserPopupRoles(s.state, s.Guild, m.RoleIDs)
	if err != nil {
		log.Errorln("Failed to get roles:", err)
		return
	}

	semaphore.IdleMust(func() {
		s.UserPopupBody.Attach(r, 2)
		s.UserPopupBody.Grid.ShowAll()
	})
}

func (s *StatefulPopupBody) injectHandlers() {
	if s.Guild.Valid() {
		s.unhandlers = append(s.unhandlers,
			s.state.AddHandler(func(g *gateway.PresenceUpdateEvent) {
				// Since PresenceUpdate is
				if !s.Guild.Valid() || g.User.ID != s.User {
					return
				}

				s.mutex.Lock()
				defer s.mutex.Unlock()

				if s.UserPopupBody == nil {
					return
				}

				semaphore.IdleMust(func() {
					s.UserPopupBody.UpdateMemberPart(g.Nick, g.User)
					s.UserPopupBody.UpdateStatus(g.Status)
					s.UpdateActivity(g.Game)
				})

				// TODO: roles
			}),
			s.state.AddHandler(func(g *gateway.GuildMembersChunkEvent) {
				if g.GuildID != s.Guild {
					return
				}
				for _, m := range g.Members {
					if m.User.ID != s.User {
						continue
					}

					s.mutex.Lock()
					defer s.mutex.Unlock()

					if s.UserPopupBody == nil {
						return
					}

					semaphore.IdleMust(func() {
						s.UserPopupBody.UpdateMember(m)
					})

					return
				}
			}),
		)
	}

	// only add this event if the user is yourself
	if s.User == s.state.Ready.User.ID {
		s.unhandlers = append(s.unhandlers,
			s.state.AddHandler(func(g *gateway.SessionsReplaceEvent) {
				s.mutex.Lock()
				defer s.mutex.Unlock()

				if s.UserPopupBody == nil {
					return
				}

				presence := s.state.JoinSession(g)

				semaphore.IdleMust(func() {
					s.UserPopupBody.UpdateStatus(presence.Status)
					s.UpdateActivity(presence.Game)
				})
			}),
		)
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

func (s *StatefulPopupBody) AddUnhandler(u func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.unhandlers = append(s.unhandlers, u)
}

func (s *StatefulPopupBody) Destroy() {
	// s.mutex.Lock()
	// defer s.mutex.Unlock()

	for _, h := range s.unhandlers {
		h()
	}
	s.UserPopupBody.Destroy()
	s.UserPopupBody = nil
}

// thread-safe
func (s *StatefulPopupBody) setParentClass(class string) {
	gtkutils.DiffClassUnsafe(&s.parentClass, class, s.ParentStyle)
}
