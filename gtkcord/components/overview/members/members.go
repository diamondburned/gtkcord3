package members

import (
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

type Container struct {
	*gtk.ListBox
	Rows []gtk.Widgetter

	state *ningen.State

	guildID   discord.GuildID
	channelID discord.ChannelID
}

func New(s *ningen.State) (m *Container) {
	list := gtk.NewListBox()
	list.Show()
	gtkutils.InjectCSS(list, "members", "")

	m = &Container{
		ListBox: list,
		state:   s,
		Rows:    []gtk.Widgetter{},
	}

	gtkutils.OnMap(m, func() func() {
		return s.AddHandler(func(ev *gateway.GuildMemberListUpdate) {
			glib.IdleAdd(func() { m.onSync(ev) })
		})
	})

	list.SetSelectionMode(gtk.SelectionNone)
	list.Connect("row-activated", func(r *gtk.ListBoxRow) {
		i := r.Index()
		w := m.Rows[i]

		rw, ok := w.(*Member)
		if !ok {
			return
		}

		p := popup.NewPopover(r)
		p.SetPosition(gtk.PosBottom)

		body := popup.NewStatefulPopupBody(m.state, rw.ID, m.guildID)
		body.ParentStyle = p.StyleContext()

		p.SetChildren(body)
		p.Popup()
	})

	return
}

func (m *Container) onSync(ev *gateway.GuildMemberListUpdate) {
	if m.guildID != ev.GuildID {
		return
	}

	m.cleanup()
	m.reload()
}

func (m *Container) cleanup() {
	for i, r := range m.Rows {
		m.ListBox.Remove(r)
		m.Rows[i] = nil
	}
	m.Rows = nil
}

// LoadGuild is thread-safe.
func (m *Container) Load(guildID discord.GuildID, chID discord.ChannelID) {
	m.guildID = guildID
	m.channelID = chID
	m.reload()
}

func (m *Container) reload() {
	l, err := m.state.MemberState.GetMemberList(m.guildID, m.channelID)
	if err != nil {
		m.state.MemberState.RequestMemberList(m.guildID, m.channelID, 0)
		return
	}

	guild, err := m.state.Offline().Guild(m.guildID)
	if err != nil {
		return
	}

	l.ViewItems(func(items []gateway.GuildMemberListOpItem) {
		for i, it := range items {
			var item gtk.Widgetter

			switch {
			case it.Group == nil && it.Member == nil:
				item = NewMemberUnavailable()
				log.Errorln("it == nil at index", i)

			case it.Group != nil:
				var name string

				if id, err := discord.ParseSnowflake(it.Group.ID); err == nil {
					r, err := m.state.Cabinet.Role(m.guildID, discord.RoleID(id))
					if err == nil {
						name = r.Name
					}
				}
				if name == "" {
					name = strings.Title(it.Group.ID)
				}

				item = NewSection(name, it.Group.Count)

			case it.Member != nil:
				item = NewMember(it.Member.Member, it.Member.Presence, *guild)
			}

			m.Rows = append(m.Rows, item)
			m.ListBox.Insert(item, -1)
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

func MapRoles(roles []discord.Role) map[discord.RoleID]*discord.Role {
	mapped := make(map[discord.RoleID]*discord.Role, len(roles))
	for i, r := range roles {
		mapped[r.ID] = &roles[i]
	}

	return mapped
}
