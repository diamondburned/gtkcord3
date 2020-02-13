package gtkcord

import (
	"html"
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/pbpool"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	FolderSize  = 36
	IconSize    = 52
	IconPadding = 6
)

type Guilds struct {
	*gtk.ListBox

	Friends *gtk.TreeIter // TODO
	Guilds  []*Guild
}

type Guild struct {
	ExtendedWidget
	Guilds *Guilds

	Row *gtk.ListBoxRow

	Folder *GuildFolder

	Style *gtk.StyleContext
	Image *gtk.Image
	// nil if not downloaded
	Pixbuf *Pixbuf

	ID   discord.Snowflake
	Name string

	// nil if Folder
	Channels *Channels
	current  *Channel
}

func newGuildsFromFolders() ([]*Guild, error) {
	s := App.State

	var folders = s.Ready.Settings.GuildFolders
	var rows = make([]*Guild, 0, len(folders))

	for _, f := range folders {
		if len(f.GuildIDs) == 1 && f.Name == "" {
			r, err := newGuildRow(f.GuildIDs[0])
			if err != nil {
				return nil, errors.Wrap(err, "Failed to load guild "+f.GuildIDs[0].String())
			}

			rows = append(rows, r)

		} else {
			e, err := newGuildFolder(s, f)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to create a new folder "+f.Name)
			}

			rows = append(rows, e)
		}
	}

	return rows, nil
}

func newGuildsLegacy() ([]*Guild, error) {
	s := App.State

	gs, err := s.Guilds()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get guilds")
	}

	var pos = s.Ready.Settings.GuildPositions
	var rows = make([]*Guild, 0, len(gs))

	sort.Slice(gs, func(a, b int) bool {
		var found = false
		for _, guild := range pos {
			if found && guild == gs[b].ID {
				return true
			}
			if !found && guild == gs[a].ID {
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

	return rows, nil
}

func newGuilds() (*Guilds, error) {
	var rows []*Guild
	var err error

	if len(App.State.Ready.Settings.GuildPositions) > 0 {
		rows, err = newGuildsFromFolders()
	} else {
		rows, err = newGuildsLegacy()
	}

	if err != nil {
		return nil, errors.Wrap(err, "Failed to get guilds list")
	}

	l, err := gtk.ListBoxNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create list")
	}
	l.SetActivateOnSingleClick(true)

	ctx, err := l.GetStyleContext()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get guild stylecontext")
	}
	ctx.AddClass("guild")

	g := &Guilds{
		ListBox: l,
		Guilds:  rows,
	}

	for _, r := range rows {
		must(func(r *Guild) {
			l.Add(r)
			r.ShowAll()
		}, r)
	}

	l.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		row := rows[r.GetIndex()]
		if row.Folder == nil {
			// Collapse all revealers:
			for _, r := range rows {
				if r.Folder != nil {
					r.Folder.Revealer.SetRevealChild(false)
				}
			}
		} else {
			index := row.Folder.List.GetSelectedRow().GetIndex()
			if index < 0 {
				index = 0
				row.Folder.List.SelectRow(row.Folder.Guilds[0].Row)
			}

			row = row.Folder.Guilds[index]
		}

		App.loadGuild(row)
	})

	return g, nil
}

func newGuildRow(guildID discord.Snowflake) (*Guild, error) {
	g, fetcherr := App.State.Guild(guildID)
	if fetcherr != nil {
		log.Errorln("Failed to get guild ID " + guildID.String() + ", using a placeholder...")
		g = &discord.Guild{
			ID:   guildID,
			Name: "Unavailable",
		}
	}

	r, err := gtk.ListBoxRowNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a list row")
	}
	// Set paddings:
	r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
	r.SetHAlign(gtk.ALIGN_CENTER)
	r.SetVAlign(gtk.ALIGN_CENTER)
	r.SetTooltipMarkup(bold(g.Name))
	r.SetActivatable(true)

	i, err := gtk.ImageNewFromIconName("user-available", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get image-loading icon")
	}

	must(r.Add, i)

	guild := &Guild{
		ExtendedWidget: r,

		Row:   r,
		ID:    g.ID,
		Name:  g.Name,
		Image: i,
	}

	// Check if the guild is unavailable:
	if fetcherr != nil {
		guild.SetUnavailable(true)
	}

	var url = g.IconURL()
	if url == "" {
		// Guild doesn't have an icon, exit:
		return guild, nil
	}

	go guild.UpdateImage(url)
	return guild, nil
}

func (g *Guild) SetUnavailable(unavailable bool) {
	must(g.Row.SetSensitive, !unavailable)
}

func (g *Guild) Current() *Channel {
	if g.current != nil {
		return g.current
	}

	index := -1
	current := g.Channels.ChList.GetSelectedRow()
	if current == nil {
		index = g.Channels.First()
	} else {
		index = current.GetIndex()
	}

	g.current = g.Channels.Channels[index]
	must(g.Channels.ChList.SelectRow, g.current.Row)

	return g.current
}

func (g *Guild) GoTo(ch *Channel) error {
	g.current = ch

	if err := ch.loadMessages(); err != nil {
		return errors.Wrap(err, "Failed to load messages")
	}

	return nil
}

func (g *Guild) UpdateImage(url string) {
	var animated = url[:len(url)-4] == ".gif"

	if !animated {
		p, err := pbpool.DownloadScaled(url+"?size=64", IconSize, IconSize, pbpool.Round)
		if err != nil {
			log.Errorln("Failed to update the pixbuf guild icon:", err)
			return
		}

		g.Pixbuf = &Pixbuf{p, nil}
		g.Pixbuf.Set(g.Image)
	} else {
		p, err := pbpool.DownloadAnimationScaled(url+"?size=64", IconSize, IconSize, pbpool.Round)
		if err != nil {
			log.Errorln("Ffailed to update the pixbuf guild animation:", err)
			return
		}

		g.Pixbuf = &Pixbuf{nil, p}
		g.Pixbuf.Set(g.Image)
	}
}

func escape(str string) string {
	return html.EscapeString(str)
}

func bold(str string) string {
	return "<b>" + escape(str) + "</b>"
}
