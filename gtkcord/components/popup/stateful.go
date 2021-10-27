package popup

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
	"github.com/diamondburned/ningen/v2/handlerrepo"
)

type StatefulPopupBody struct {
	*UserPopupBody
	state *ningen.State
	stop  bool

	ParentStyle *gtk.StyleContext
	parentClass string

	UserID  discord.UserID
	GuildID discord.GuildID

	Prefetch *discord.User

	stateHandlers interface {
		handlerrepo.AddHandler
		handlerrepo.Unbinder
	}

	bound bool
}

func NewStatefulPopupBody(s *ningen.State, user discord.UserID, guild discord.GuildID) *StatefulPopupBody {
	return statefulPopupUser(&StatefulPopupBody{
		state:   s,
		UserID:  user,
		GuildID: guild,
	})
}

func NewStatefulPopupUser(s *ningen.State, user discord.User, guild discord.GuildID) *StatefulPopupBody {
	return statefulPopupUser(&StatefulPopupBody{
		state:    s,
		UserID:   user.ID,
		GuildID:  guild,
		Prefetch: &user,
	})
}

func statefulPopupUser(body *StatefulPopupBody) *StatefulPopupBody {
	body.stateHandlers = handlerrepo.NewRepository(body.state)
	body.UserPopupBody = NewUserPopupBody()
	body.initialize()
	return body
}

// must be thread-safe, function is running in a goroutine
func (s *StatefulPopupBody) initialize() {
	if !s.UserID.IsValid() {
		return
	}

	if s.Prefetch == nil && !s.GuildID.IsValid() {
		go func() {
			u, err := s.state.User(s.UserID)
			if err != nil {
				log.Errorln("Failed to get user:", err)
				return
			}

			glib.IdleAdd(func() {
				s.Prefetch = u
				s.initialize()
			})
		}()
		return
	}

	s.ConnectMap(func() { s.injectHandlers() })

	// Updating the user is optional, since the member state can have people
	// too. This is only needed for DM channels.
	if s.Prefetch != nil {
		s.UserPopupBody.Update(*s.Prefetch)
	}

	s.asyncUpdatePresence(false)

	if s.GuildID.IsValid() {
		s.asyncUpdateMember(false)
		s.UserPopupBody.Attach(NewUserPopupRoles(s.state, s.GuildID, s.UserID), 2)
	}

	s.UserPopupBody.Grid.ShowAll()
}

func (s *StatefulPopupBody) injectHandlers() {
	if s.bound {
		return
	}
	s.bound = true
	var handlers []func()

	if s.GuildID.IsValid() {
		handlers = append(handlers,
			s.stateHandlers.AddHandler(func(g *gateway.PresenceUpdateEvent) {
				if s.GuildID.IsValid() && g.User.ID == s.UserID {
					s.asyncUpdatePresence(true)
				}
			}),
			s.stateHandlers.AddHandler(func(m *gateway.GuildMemberUpdateEvent) {
				if m.GuildID == s.GuildID && m.User.ID == s.UserID {
					s.asyncUpdateMember(true)
				}
			}),
			s.stateHandlers.AddHandler(func(g *gateway.GuildMembersChunkEvent) {
				if g.GuildID == s.GuildID {
					// Not too expensive, hopefully.
					s.asyncUpdateMember(true)
				}
			}),
		)
	}

	// only add this event if the user is yourself
	me, _ := s.state.Me()
	if s.UserID == me.ID {
		handlers = append(handlers,
			s.stateHandlers.AddHandler(func(g *gateway.SessionsReplaceEvent) {
				s.asyncUpdatePresence(true)
			}),
		)
	}

	s.ConnectUnmap(func() {
		s.bound = false
		for _, f := range handlers {
			f()
		}
	})
}

func (s *StatefulPopupBody) asyncUpdatePresence(thread bool) {
	p, _ := s.state.Presence(s.GuildID, s.UserID)
	if p == nil {
		return
	}

	var activity *discord.Activity
	if len(p.Activities) > 0 {
		activity = &p.Activities[0]
	}

	f := func() {
		s.UserPopupBody.UpdateStatus(p.Status)
		s.UpdateActivity(activity)
	}

	if thread {
		glib.IdleAdd(f)
	} else {
		f()
	}
}

func (s *StatefulPopupBody) asyncUpdateMember(thread bool) {
	var member *discord.Member
	var err error

	if !thread {
		member, err = s.state.Offline().Member(s.GuildID, s.UserID)
	} else {
		member, err = s.state.Member(s.GuildID, s.UserID)
	}

	if err != nil {
		if !thread {
			// Force an API query if not possible.
			go s.asyncUpdateMember(true)
		}
		return
	}

	f := func() {
		if s.Prefetch == nil {
			s.Prefetch = &member.User
		}

		s.UserPopupBody.UpdateMember(*member)
		s.UserPopupBody.Grid.ShowAll()
	}

	if !thread {
		f()
	} else {
		glib.IdleAdd(f)
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

// thread-safe
func (s *StatefulPopupBody) setParentClass(class string) {
	gtkutils.DiffClass(&s.parentClass, class, s.ParentStyle)
}
