package members

import (
	"sort"
	"strings"
	"sync"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const Width = 280

type Container struct {
	*gtk.ScrolledWindow
	// Main  *gtk.Box
	// Roles []*RoleSection

	List *gtk.ListBox
	Rows []gtkutils.ExtendedWidget

	GuildID discord.Snowflake

	mutex sync.Mutex
	state *ningen.State
}

// thread-safe
func New(s *ningen.State) (m *Container) {
	semaphore.IdleMust(func() {
		sw, _ := gtk.ScrolledWindowNew(nil, nil)
		sw.Show()
		sw.SetSizeRequest(Width, -1)
		gtkutils.InjectCSSUnsafe(sw, "membercontainer", "")

		list, _ := gtk.ListBoxNew()
		list.Show()
		gtkutils.InjectCSSUnsafe(list, "members", "")
		sw.Add(list)

		m = &Container{
			ScrolledWindow: sw,
			List:           list,
			state:          s,

			Rows: []gtkutils.ExtendedWidget{},
		}
		s.MemberList.OnOP = m.handle
		s.MemberList.OnSync = m.handleSync

		list.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
			i := r.GetIndex()
			w := m.Rows[i]

			rw, ok := w.(*Member)
			if !ok {
				return
			}

			p := popup.NewPopover(r)
			p.SetPosition(gtk.POS_LEFT)

			body := popup.NewStatefulPopupBody(m.state, rw.ID, m.GuildID)
			body.ParentStyle, _ = p.GetStyleContext()
			body.AddUnhandler(func() {
				l.SelectRow(nil)
			})

			p.SetChildren(body)
			p.Popup()
		})
	})
	return
}

func (m *Container) handleSync(ml *ningen.MemberList, guildID discord.Snowflake) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.GuildID != guildID {
		return
	}

	guild, err := m.state.Store.Guild(guildID)
	if err != nil {
		log.Errorln("Failed to get guild:", err)
		return
	}

	log.Println("handleSync called")

	m.cleanup()
	m.reset(ml, *guild)
}

func (m *Container) handle(
	ml *ningen.MemberList, guildID discord.Snowflake, op gateway.GuildMemberListOp) {

	if m.GuildID != guildID {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
}

// Cleanup is thread-safe.
func (m *Container) Cleanup() {
	m.mutex.Lock()
	m.cleanup()
	m.mutex.Unlock()
}

func (m *Container) cleanup() {
	semaphore.IdleMust(func() {
		for i, r := range m.Rows {
			m.List.Remove(r)
			m.Rows[i] = nil
		}
		m.Rows = m.Rows[:0]
	})
}

// LoadGuild is thread-safe.
func (m *Container) LoadGuild(guildID discord.Snowflake) error {
	guild, err := m.state.Guild(guildID)
	if err != nil {
		return errors.Wrap(err, "Failed to get guild")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.GuildID = guild.ID

	// Borrow MemberList's mutex
	ml := m.state.MemberList.GetMemberList(guild.ID)
	if ml == nil {
		return nil
	}
	unlock := ml.Acquire()
	defer unlock()

	m.reset(ml, *guild)
	return nil
}

func (m *Container) reset(ml *ningen.MemberList, guild discord.Guild) {
	semaphore.IdleMust(func() {
		for i, it := range ml.Items {
			var item gtkutils.ExtendedWidget

			switch {
			case it == nil, it.Group == nil && it.Member == nil:
				item = NewMemberUnavailable()
				log.Errorln("it == nil at index", i)

			case it.Group != nil:
				var name string

				if id, err := discord.ParseSnowflake(it.Group.ID); err == nil {
					r, err := m.state.Store.Role(m.GuildID, id)
					if err == nil {
						name = r.Name
					}
				}
				if name == "" {
					name = strings.Title(it.Group.ID)
				}

				item = NewSection(name, it.Group.Count)

			case it.Member != nil:
				item = NewMember(it.Member.Member, it.Member.Presence, guild)
			}

			m.Rows = append(m.Rows, item)
			m.List.Insert(item, -1)
		}
	})
}

// func (m *Container) getHoistRoles() ([]discord.Role, error) {
// 	r, err := m.state.Store.Roles(m.GuildID)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "Failed to get roles")
// 	}
// 	return FilterHoistRoles(r), nil
// }

// func (m *Container) memberSelect(s *RoleSection, r *gtk.ListBoxRow) {
// 	if member := s.getFromRow(r); member != nil {
// 		p := popup.NewPopover(r)

// 		body := popup.NewStatefulPopupBody(m.state, member.ID, m.GuildID)
// 		body.ParentStyle, _ = p.GetStyleContext()
// 		body.AddUnhandler(func() {
// 			s.Members.SelectRow(nil)
// 		})

// 		p.SetChildren(body)
// 		p.Show()
// 	}
// }

// // sumMembers ignores ID == 0
// func sumMembers(sections []*RoleSection) (sum int) {
// 	for _, s := range sections {
// 		if s.ID > 0 {
// 			sum += len(s.members)
// 		}
// 	}
// 	return
// }

func GetTopRole(roles []discord.Role) *discord.Role {
	var pos, ind = -1, -1

	for i, r := range roles {
		if r.Position < pos || pos == -1 {
			ind = i
			pos = r.Position
		}
	}

	if ind < 0 {
		return nil
	}
	return &roles[ind]
}

func GetTopRoleID(
	ids []discord.Snowflake, roles map[discord.Snowflake]*discord.Role) *discord.Role {

	var pos, ind = -1, -1

	for i, id := range ids {
		if r := roles[id]; r.Position < pos || pos == -1 {
			ind = i
			pos = r.Position
		}
	}

	if ind < 0 {
		return nil
	}

	return roles[ids[ind]]
}

func FilterHoistRoles(roles []discord.Role) []discord.Role {
	filtered := roles[:0]

	for i, r := range roles {
		if r.Hoist {
			filtered = append(filtered, roles[i])
		}
	}

	return filtered
}

func SortRoles(roles []discord.Role) {
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position > roles[j].Position
	})
}

func MapRoles(roles []discord.Role) map[discord.Snowflake]*discord.Role {
	var mapped = make(map[discord.Snowflake]*discord.Role, len(roles))
	for i, r := range roles {
		mapped[r.ID] = &roles[i]
	}

	return mapped
}
