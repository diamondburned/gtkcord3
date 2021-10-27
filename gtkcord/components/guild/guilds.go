package guild

import (
	"sort"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/ningen/v2"
	"github.com/diamondburned/ningen/v2/states/read"
)

type Guilds struct {
	gtk.Widgetter

	ListBox  *gtk.ListBox
	DMButton *DMButton

	Guilds   []gtk.Widgetter
	Current  *Guild
	OnSelect func(g *Guild)

	state *ningen.State
}

func NewGuilds(s *ningen.State) *Guilds {
	settings := s.Ready().UserSettings

	if len(settings.GuildFolders) > 0 {
		return newGuildsFromFolders(s, settings.GuildFolders)
	} else {
		return newGuildsLegacy(s, settings.GuildPositions)
	}
}

func newGuildsFromFolders(s *ningen.State, folders []gateway.GuildFolder) *Guilds {
	rows := make([]gtk.Widgetter, 0, len(folders))
	g := &Guilds{}

	for i := 0; i < len(folders); i++ {
		f := folders[i]

		if len(f.GuildIDs) == 1 {
			r := newGuildRow(s, f.GuildIDs[0], nil)
			rows = append(rows, r)
		} else {
			e := newGuildFolder(s, f, g.onFolderSelect)
			rows = append(rows, e)
		}
	}

	g.Guilds = rows
	initGuilds(g, s)
	return g
}

func newGuildsLegacy(s *ningen.State, positions []discord.GuildID) *Guilds {
	// TODO: retry mechanism.
	guilds, _ := s.Guilds()

	rows := make([]gtk.Widgetter, 0, len(guilds))

	sort.Slice(guilds, func(a, b int) bool {
		var found = false
		for _, guild := range positions {
			if found && guild == guilds[b].ID {
				return true
			}
			if !found && guild == guilds[a].ID {
				found = true
			}
		}

		return false
	})

	for _, g := range guilds {
		r := newGuildRow(s, g.ID, nil)
		rows = append(rows, r)
	}

	g := &Guilds{
		Guilds: rows,
	}
	initGuilds(g, s)
	return g
}

func initGuilds(g *Guilds, s *ningen.State) {
	g.state = s

	g.ListBox = gtk.NewListBox()
	g.ListBox.SetActivateOnSingleClick(true)
	gtkutils.InjectCSS(g.ListBox, "guilds", "")
	g.ListBox.Show()

	gw := gtk.NewScrolledWindow(nil, nil)
	gw.SetPolicy(gtk.PolicyNever, gtk.PolicyExternal) // external means hidden scroll
	gw.Add(g.ListBox)
	gw.Show()
	g.Widgetter = gw

	// Add the button to the second of the list:
	g.DMButton = NewPMButton(s)
	g.ListBox.Insert(g.DMButton, -1)

	// Add the rest:
	for _, guild := range g.Guilds {
		g.ListBox.Insert(guild, -1)
	}

	g.ListBox.ShowAll()
	g.ListBox.Connect("row-activated", g.rowActivated)
	s.ReadState.OnUpdate(func(rs *read.UpdateEvent) {
		glib.IdleAdd(func() { g.TraverseReadState(rs) })
	})
}

func (g *Guilds) rowActivated(l *gtk.ListBox, r *gtk.ListBoxRow) {
	guild, dm := g.preSelect(r)
	switch {
	case guild != nil:
		g.onSelect(guild)
	case dm:
		g.DMButton.OnClick()
	}
}

func (g *Guilds) Select(r *gtk.ListBoxRow) {
	g.ListBox.SelectRow(r)
	g.preSelect(r)
}

func (g *Guilds) preSelect(r *gtk.ListBoxRow) (guild *Guild, dm bool) {
	var index = -1
	if r != nil {
		index = r.Index()
	}

	switch {
	case index < 1:
		g.unselectAll(-1)
		g.DMButton.Unread.SetActive(true)
		return nil, true
	default:
		g.DMButton.inactive() // manual work
		index--
	}

	row := g.Guilds[index]

	// Unselect all guild folders except the current one:
	g.unselectAll(index)

	switch r := row.(type) {
	case *Guild:
		r.Unread.SetActive(true)
		return r, false
	}
	return nil, false
}

func (g *Guilds) unselectAll(except int) {
	// Unselect all guild folders except the current one:
	for i, r := range g.Guilds {
		if i == except {
			continue
		}

		switch r := r.(type) {
		case *Guild:
			r.Unread.SetActive(false)
		case *GuildFolder:
			// TODO: r.Unread.SetSuppress(true)
			r.unselectAll(-1) // will never be the current folder.
			r.List.SelectRow(nil)
		}
	}
}

func (guilds *Guilds) onFolderSelect(g *Guild) {
	guilds.ListBox.SelectRow(nil)
	guilds.onSelect(g)
}

func (guilds *Guilds) onSelect(g *Guild) {
	if guilds.OnSelect == nil {
		return
	}

	guilds.Current = g
	guilds.OnSelect(g)
}

func (guilds *Guilds) FindByID(guildID discord.GuildID) (*Guild, *GuildFolder) {
	return guilds.Find(func(g *Guild) bool {
		return g.ID == guildID
	})
}

func (guilds *Guilds) Find(fn func(*Guild) bool) (*Guild, *GuildFolder) {
	for _, v := range guilds.Guilds {
		switch v := v.(type) {
		case *Guild:
			if fn(v) {
				return v, nil
			}
		case *GuildFolder:
			for _, guild := range v.Guilds {
				if fn(guild) {
					return guild, v
				}
			}
		}
	}

	return nil, nil
}

func (guilds *Guilds) TraverseReadState(rs *read.UpdateEvent) {
	ch, err := guilds.state.Offline().Channel(rs.ChannelID)
	if err != nil {
		return
	}
	if !ch.GuildID.IsValid() {
		// DM:
		guilds.DMButton.setUnread(rs.Unread)
		return
	}

	guild, _ := guilds.FindByID(ch.GuildID)
	if guild == nil {
		return
	}

	pinged := rs.MentionCount > 0
	unread := rs.Unread

	if guilds.state.MutedState.Channel(ch.ID) {
		unread = false
	}

	// TODO: confirm that nothing breaks with this running prematurely.
	if guild.muted {
		return
	}

	if !unread {
		delete(guild.unreadChs, ch.ID)
	} else {
		guild.unreadChs[ch.ID] = pinged
	}

	if !unread || !pinged {
		for _, chPinged := range guild.unreadChs {
			unread = true
			if pinged {
				break
			}
			if !pinged && chPinged {
				pinged = true
			}
		}
	}

	guild.setUnread(unread, pinged)
}
