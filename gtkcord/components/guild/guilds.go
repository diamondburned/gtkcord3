package guild

import (
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Guilds struct {
	*gtk.ListBox
	Guilds []*Guild

	OnSelect func(g *Guild)
}

func NewGuildsFromFolders(folders []gateway.GuildFolder) (*Guilds, error) {
	var rows = make([]*Guild, 0, len(folders))

	for i := 0; i < len(folders); i++ {
		f := folders[i]

		if len(f.GuildIDs) == 1 && f.Name == "" {
			r, err := newGuildRow(f.GuildIDs[0])
			if err != nil {
				return nil, errors.Wrap(err, "Failed to load guild "+f.GuildIDs[0].String())
			}

			rows = append(rows, r)

		} else {
			e, err := newGuildFolder(f)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to create a new folder "+f.Name)
			}

			rows = append(rows, e)
		}
	}

	return newGuilds(rows), nil
}

func NewGuilds(guilds []discord.Guild, positions []discord.Snowflake) (*Guilds, error) {
	var rows = make([]*Guild, 0, len(guilds))

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

	for _, g := range gs {
		r, err := newGuildRow(g.ID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load guild "+g.Name)
		}

		rows = append(rows, r)
	}

	return newGuilds(rows), nil
}

func newGuilds(guilds []*Guild) (g *Guilds) {
	semaphore.IdleMust(func() {
		l, _ := gtk.ListBoxNew()
		l.SetActivateOnSingleClick(true)
		gtkutils.InjectCSSUnsafe(l, "guilds", "")

		g = &Guilds{
			ListBox: l,
			Guilds:  guilds,
		}
	})

	g := &Guilds{
		ListBox: l,
		Guilds:  rows,
	}

	must(func() {
		// Add the button to the first of the list:
		l.Add(dm)

		// Add the rest:
		for i := 0; i < len(rows); i++ {
			l.Add(rows[i])
			rows[i].ShowAll()
		}
	})

	must(l.Connect, "row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		index := r.GetIndex()
		if index < 1 {
			go App.loadGuild(nil)
			return
		}

		index--
		row := rows[index]

		// Unselect all guild folders except the current one:
		for i, r := range rows {
			if i == index {
				continue
			}
			if r.Folder != nil {
				r.Folder.List.SelectRow(nil)
			}
		}

		// We ignore folders, as that'll be handled by its own handler.
		if row.Folder != nil {
			return
		}

		// load the guild, then subscribe to typing events
		go App.loadGuild(row)
	})

	g.find(func(g *Guild) bool {
		g.UpdateImage()
		return false
	})

	return g, nil
}

func (guilds *Guilds) findByID(guildID discord.Snowflake) (*Guild, *GuildFolder) {
	return guilds.find(func(g *Guild) bool {
		return g.ID == guildID
	})
}

func (guilds *Guilds) find(fn func(*Guild) bool) (*Guild, *GuildFolder) {
	for _, guild := range guilds.Guilds {
		if guild.Folder == nil && fn(guild) {
			return guild, nil
		}

		if guild.Folder != nil {
			folder := guild.Folder

			for _, guild := range folder.Guilds {
				if fn(guild) {
					return guild, folder
				}
			}
		}
	}

	return nil, nil
}
