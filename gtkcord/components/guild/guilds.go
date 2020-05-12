package guild

import (
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Guilds struct {
	gtkutils.ExtendedWidget

	ListBox  *gtk.ListBox
	DMButton *DMButton

	Guilds   []gtkutils.ExtendedWidget
	Current  *Guild
	OnSelect func(g *Guild)

	state *ningen.State
}

func NewGuilds(s *ningen.State) (g *Guilds, err error) {
	semaphore.IdleMust(func() {
		if len(s.Ready.Settings.GuildFolders) > 0 {
			g, err = newGuildsFromFolders(s, s.Ready.Settings.GuildFolders)
		} else {
			g, err = newGuildsLegacy(s, s.Ready.Settings.GuildPositions)
		}
	})
	return
}

func newGuildsFromFolders(s *ningen.State, folders []gateway.GuildFolder) (*Guilds, error) {
	var rows = make([]gtkutils.ExtendedWidget, 0, len(folders))
	var g = &Guilds{}

	for i := 0; i < len(folders); i++ {
		f := folders[i]

		if len(f.GuildIDs) == 1 {
			r, err := newGuildRow(s, f.GuildIDs[0], nil, nil)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to load guild "+f.GuildIDs[0].String())
			}

			rows = append(rows, r)

		} else {
			e, err := newGuildFolder(s, f, g.onFolderSelect)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to create a new folder "+f.Name)
			}

			rows = append(rows, e)
		}
	}

	g.Guilds = rows
	initGuilds(g, s)
	return g, nil
}

func newGuildsLegacy(s *ningen.State, positions []discord.Snowflake) (*Guilds, error) {
	guilds, err := s.Guilds()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get guilds")
	}

	var rows = make([]gtkutils.ExtendedWidget, 0, len(guilds))

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
		r, err := newGuildRow(s, g.ID, &g, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load guild "+g.Name)
		}

		rows = append(rows, r)
	}

	g := &Guilds{
		Guilds: rows,
	}
	initGuilds(g, s)
	return g, nil
}

func initGuilds(g *Guilds, s *ningen.State) {
	g.state = s

	dm := NewPMButton(s)

	gw, _ := gtk.ScrolledWindowNew(nil, nil)
	gw.Show()
	gw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_EXTERNAL) // external means hidden scroll
	g.ExtendedWidget = gw

	l, _ := gtk.ListBoxNew()
	l.Show()
	l.SetActivateOnSingleClick(true)
	gtkutils.InjectCSSUnsafe(l, "guilds", "")

	gw.Add(l)
	g.ListBox = l

	// Add the button to the second of the list:
	g.DMButton = dm
	l.Insert(dm, -1)

	// Add the rest:
	for _, g := range g.Guilds {
		l.Insert(g, -1)
		g.ShowAll()
	}
	l.Connect("row-activated", g.rowActivated)
	s.Read.OnChange(g.TraverseReadState)
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
		index = r.GetIndex()
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

	var row = g.Guilds[index]

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

func (guilds *Guilds) FindByID(guildID discord.Snowflake) (*Guild, *GuildFolder) {
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

func (guilds *Guilds) TraverseReadState(chrs gateway.ReadState, unread bool) {
	ch, err := guilds.state.Store.Channel(chrs.ChannelID)
	if err != nil {
		return
	}
	if !ch.GuildID.Valid() {
		// DM:
		guilds.DMButton.setUnread(unread)
		return
	}

	guild, _ := guilds.FindByID(ch.GuildID)
	if guild == nil {
		return
	}

	pinged := chrs.MentionCount > 0

	if guilds.state.Muted.Channel(ch.ID) {
		unread = false
	}

	guild.busy.Lock()
	defer guild.busy.Unlock()

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

	// IdleMust so the mutex is affected.
	semaphore.IdleMust(guild.setUnread, unread, pinged)
}
