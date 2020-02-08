package gtkcord

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"

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

func (a *Application) newGuilds(s *state.State) (*Guilds, error) {
	log.Println("Version", s.Ready.Version)

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

			r, err := a.newGuildRow(g)
			if err != nil {
				return nil,
					errors.Wrap(err, "Failed to load guild "+g.Name)
			}

			rows = append(rows, r)

		default:
			e, err := a.newGuildFolder(s, f)
			if err != nil {
				return nil,
					errors.Wrap(err, "Failed to create a new folder "+f.Name)
			}

			rows = append(rows, e)
		}
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
				return
			}

			row = row.Folder.Guilds[index]
		}

		a.loadGuild(row)
	})

	must(l.ShowAll)

	g := &Guilds{
		ListBox: l,
		Guilds:  rows,
	}

	return g, nil
}

/* Tree logic
func (gs *Guilds) selector(sl *gtk.TreeSelection) {
	_, iter, ok := sl.GetSelected()
	if !ok {
		return
	}

	path, err := gs.Store.GetPath(iter)
	if err != nil {
		logWrap(err, "Couldn't get path from selected")
		return
	}

	var target *Guild

	for _, g := range gs.Guilds {
		if g := g.Search(path); g != nil {
			target = g
			break
		}
	}

	if target == nil {
		logError(errors.New("What was clicked?"))
		return
	}

	if target.Folder != nil {
		if !target.Folder.Expanded {
			target.Folder.Expanded = true
			gs.CollapseAll()
			gs.ExpandRow(target.Path, true)
		} else {
			target.Folder.Expanded = false
			// target.Pixbuf.SetProperty("class")
			gs.CollapseRow(target.Path)
		}
	}
}
*/

func (a *Application) newGuildRow(guild *discord.Guild) (*Guild, error) {
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

	r.Connect("activate", func(r *gtk.ListBoxRow) bool {
		log.Println("Guild", guild.Name, "pressed")
		return true
	})

	g := &Guild{
		IWidget: r,
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

/* Tree logic

func (g *Guild) Search(path *gtk.TreePath) *Guild {
	if g.Path.Compare(path) == 0 {
		return g
	}

	if g.Folder == nil {
		return nil
	}

	for _, g := range g.Folder.Guilds {
		if g.Path.Compare(path) == 0 {
			return g
		}
	}

	return nil
}

func (g *Guild) UpdateStore() {
	if g.Path == nil {
		path, err := g.Store.GetPath(g.Iter)
		if err != nil {
			logWrap(err, "Failed to get iter path")
		}
		g.Path = path
	}

	switch {
	case g.Pixbuf != nil:
		must(func(g *Guild) {
			g.Store.SetValue(g.Iter, 0, g.Pixbuf)
		}, g)
	case g.Animation != nil:
		must(func(g *Guild) {
			g.Store.SetValue(g.Iter, 0, *g.Animation)
		}, g)
	}
}

*/

func bold(str string) string {
	return "<b>" + html.EscapeString(str) + "</b>"
}
