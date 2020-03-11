package popup

import (
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
	if !s.User.Valid() {
		return
	}

	u, err := s.state.User(s.User)
	if err != nil {
		log.Errorln("Failed to get user:", err)
		return
	}

	p, err := s.state.Presence(s.Guild, u.ID)
	if err != nil {
		p, err = s.state.Presence(0, u.ID)
	}

	if err == nil {
		s.UserPopupBody.UpdateStatus(p.Status)
		s.UpdateActivity(p.Game)
	}

	if !s.Guild.Valid() {
		s.UserPopupBody.Update(*u)
		semaphore.IdleMust(s.UserPopupBody.Box.ShowAll)
		return
	}

	s.injectHandlers()

	// fetch above presence if error not nil
	if err != nil {
		s.state.RequestMember(s.Guild, u.ID)
	}

	m, err := s.state.Store.Member(s.Guild, u.ID)
	if err != nil {
		s.UserPopupBody.Update(*u)
		semaphore.IdleMust(s.UserPopupBody.Box.ShowAll)
		return
	}

	s.UserPopupBody.UpdateMember(*m)
	semaphore.IdleMust(s.UserPopupBody.Box.ShowAll)

	r, err := NewUserPopupRoles(s.state, s.Guild, m.RoleIDs)
	if err != nil {
		log.Errorln("Failed to get roles:", err)
		return
	}

	semaphore.IdleMust(func() {
		s.UserPopupBody.Box.Add(r)
		s.UserPopupBody.Box.ShowAll()
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

				s.UserPopupBody.UpdateMemberPart(g.Nick, g.User)
				s.UserPopupBody.UpdateStatus(g.Status)
				s.UpdateActivity(g.Game)

				// TODO: roles
			}),
		)
	}

	// only add this event if the user is yourself
	if s.User == s.state.Ready.User.ID {
		s.unhandlers = append(s.unhandlers,
			s.state.AddHandler(func(g *gateway.SessionsReplaceEvent) {
				presence := s.state.JoinSession(g)
				s.UserPopupBody.UpdateStatus(presence.Status)
				s.UpdateActivity(presence.Game)
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

func (s *StatefulPopupBody) Destroy() {
	for _, h := range s.unhandlers {
		h()
	}
	s.UserPopupBody.Destroy()
	s.UserPopupBody = nil
}

// thread-safe
func (s *StatefulPopupBody) setParentClass(class string) {
	gtkutils.DiffClass(&s.parentClass, class, s.ParentStyle)
}
