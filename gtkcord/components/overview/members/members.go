package members

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/states/member"
	"github.com/gotk3/gotk3/gtk"
)

type Container struct {
	*gtk.ListBox
	Rows []gtkutils.ExtendedWidget

	GuildID discord.Snowflake

	state *ningen.State
}

// thread-unsafe
func New(s *ningen.State) (m *Container) {
	list, _ := gtk.ListBoxNew()
	list.Show()
	gtkutils.InjectCSSUnsafe(list, "members", "")

	m = &Container{
		ListBox: list,
		state:   s,
		Rows:    []gtkutils.ExtendedWidget{},
	}
	// TODO
	// s.MemberList.OnOP = m.handle
	s.Members.OnSync(m.handleSync)

	// unreference these things
	list.Connect("destroy", m.cleanup)

	list.SetSelectionMode(gtk.SELECTION_NONE)
	list.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		i := r.GetIndex()
		w := m.Rows[i]

		rw, ok := w.(*Member)
		if !ok {
			return
		}

		p := popup.NewPopover(r)
		p.SetPosition(gtk.POS_BOTTOM)

		body := popup.NewStatefulPopupBody(m.state, rw.ID, m.GuildID)
		body.ParentStyle, _ = p.GetStyleContext()

		p.SetChildren(body)
		p.Popup()
	})

	return
}

func (m *Container) handleSync(id string, ml *member.List, guildID discord.Snowflake) {
	semaphore.Async(func() {
		if m.GuildID != guildID {
			return
		}
		m.cleanup()
		m.reset(ml, guildID)
	})
}

// TODO
// func (m *Container) handleUnsafe(
// 	ml *ningen.MemberList, guildID discord.Snowflake, op gateway.GuildMemberListOp) {

// 	if m.GuildID != guildID {
// 		return
// 	}

// 	m.mutex.Lock()
// 	defer m.mutex.Unlock()
// }

func (m *Container) cleanup() {
	for _, r := range m.Rows {
		m.ListBox.Remove(r)
	}
	m.Rows = nil
}

// LoadGuild is thread-unsafe.
func (m *Container) LoadChannel(guildID, channelID discord.Snowflake) {
	m.GuildID = guildID

	// Borrow MemberList's mutex
	err := m.state.Members.GetMemberList(guildID, channelID, func(l *member.List) {
		m.reset(l, guildID)
	})
	// Request a member list if none.
	if err != nil {
		m.state.Members.RequestMemberList(guildID, channelID, 0)
		return
	}
}

func (m *Container) reset(ml *member.List, guildID discord.Snowflake) {
	g, err := m.state.Store.Guild(guildID)
	if err != nil {
		log.Errorln("Failed to get guild:", err)
		return
	}

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
			item = NewMember(it.Member.Member, it.Member.Presence, *g)
		}

		m.Rows = append(m.Rows, item)
		m.ListBox.Insert(item, i)
	}
}
