package gtkcord

import (
	"fmt"
	"html"
	"io/ioutil"
	"sort"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	FolderSize  = 36
	IconSize    = 56
	IconPadding = 6
)

type Guilds struct {
	// *gtk.TreeView
	// Store *gtk.TreeStore

	*gtk.ListBox

	Friends *gtk.TreeIter // TODO
	Guilds  []*Guild
}

type Guild struct {
	gtk.IWidget
	Row *gtk.ListBoxRow

	/* Tree logic

	*gtk.TreeIter
	Folder *GuildFolder // can be non-nil

	Parent *gtk.TreeIter
	Iter   *gtk.TreeIter
	Path   *gtk.TreePath
	Store  *gtk.TreeStore

	*/

	Folder *GuildFolder

	Style *gtk.StyleContext
	Image *gtk.Image
	// nil if not downloaded
	Pixbuf    *gdk.Pixbuf
	Animation *gdk.PixbufAnimation

	ID   discord.Snowflake
	Name string

	// nil if Folder
	Channels *Channels
}

func newGuildsFromFolders(s *state.State) ([]*Guild, error) {
	var folders = s.Ready.Settings.GuildFolders
	var rows = make([]*Guild, 0, len(folders))

	for _, f := range folders {
		switch len(f.GuildIDs) {
		case 0: // ???
			continue
		case 1:
			g, err := s.Guild(f.GuildIDs[0])
			if err != nil {
				return nil,
					errors.Wrap(err, "Failed to get guild in folder "+f.Name)
			}

			r, err := newGuildRow(*g)
			if err != nil {
				return nil,
					errors.Wrap(err, "Failed to load guild "+g.Name)
			}

			rows = append(rows, r)

		default:
			e, err := newGuildFolder(s, f)
			if err != nil {
				return nil,
					errors.Wrap(err, "Failed to create a new folder "+f.Name)
			}

			rows = append(rows, e)
		}
	}

	return rows, nil
}

func newGuildsLegacy(s *state.State) ([]*Guild, error) {
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
		r, err := newGuildRow(g)
		if err != nil {
			return nil,
				errors.Wrap(err, "Failed to load guild "+g.Name)
		}

		rows = append(rows, r)
	}

	return rows, nil
}

func newGuilds(s *state.State, callback func(*Guild)) (*Guilds, error) {
	var rows []*Guild
	var err error

	if len(s.Ready.Settings.GuildPositions) > 0 {
		rows, err = newGuildsFromFolders(s)
	} else {
		rows, err = newGuildsLegacy(s)
	}

	if err != nil {
		return nil, errors.Wrap(err, "Failed to get guilds list")
	}

	l, err := gtk.ListBoxNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create list")
	}
	l.SetActivateOnSingleClick(true)

	for _, r := range rows {
		must(l.Add, r.IWidget)
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

		callback(row)
	})

	must(l.ShowAll)

	g := &Guilds{
		ListBox: l,
		Guilds:  rows,
	}

	return g, nil
}

func newGuildRow(guild discord.Guild) (*Guild, error) {
	r, err := gtk.ListBoxRowNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create a list row")
	}
	// Set paddings:
	r.SetSizeRequest(IconSize+IconPadding*2, IconSize+IconPadding*2)
	r.SetHAlign(gtk.ALIGN_CENTER)
	r.SetVAlign(gtk.ALIGN_CENTER)
	r.SetTooltipMarkup(bold(guild.Name))
	r.SetActivatable(true)

	i, err := gtk.ImageNewFromIconName("image-loading", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get image-loading icon")
	}
	// i.SetTooltipText(guild.Name)

	must(r.Add, i)

	g := &Guild{
		IWidget: r,
		Row:     r,
		ID:      guild.ID,
		Name:    guild.Name,
		Image:   i,
		// Iter:       store.Append(parent),
		// Store:      store,
		// Parent:     parent,
	}

	var url = guild.IconURL()
	if url == "" {
		// Guild doesn't have an icon, exit:
		return g, nil
	}

	go g.UpdateImage(url)
	return g, nil
}

func (g *Guild) UpdateImage(url string) {
	var animated = url[:len(url)-4] == ".gif"

	r, err := HTTPClient.Get(url + "?size=64")
	if err != nil {
		logWrap(err, "Failed to GET URL "+url)
		return
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		logError(fmt.Errorf("Bad status code %d for %s", r.StatusCode, url))
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logWrap(err, "Failed to download image")
		return
	}

	if !animated {
		p, err := NewPixbuf(b, PbSize(IconSize, IconSize))
		if err != nil {
			logWrap(err, "Failed to get the pixbuf guild icon")
			return
		}

		g.Pixbuf = p
		g.updateImage()
	} else {
		p, err := NewAnimator(b, PbSize(IconSize, IconSize))
		if err != nil {
			logWrap(err, "Failed to get the pixbuf guild animation")
		}

		g.Animation = p
		g.updateImage()
	}
}

func (g *Guild) updateImage() {
	switch {
	case g.Pixbuf != nil:
		must(func(g *Guild) {
			g.Image.SetFromPixbuf(g.Pixbuf)
		}, g)
	case g.Animation != nil:
		must(func(g *Guild) {
			g.Image.SetFromAnimation(g.Animation)
		}, g)
	}
}

func escape(str string) string {
	return html.EscapeString(str)
}

func bold(str string) string {
	return "<b>" + escape(str) + "</b>"
}
