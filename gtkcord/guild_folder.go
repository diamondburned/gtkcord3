package gtkcord

import (
	"html"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/state"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type GuildFolder struct {
	Revealer *gtk.Revealer
	List     *gtk.ListBox
	Guilds   []*Guild
}

func newGuildFolder(s *state.State, folder gateway.GuildFolder) (*Guild, error) {
	if folder.Color == 0 {
		folder.Color = 0x7289DA
	}

	mainEv, err := gtk.EventBoxNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create main event box")
	}

	mainBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create main box")
	}
	mainBox.SetTooltipMarkup("<b>" + html.EscapeString(folder.Name) + "</b>")

	// Add the main box into main event box
	mainEv.Add(mainBox)

	// Folder icon
	p, err := icons.PixbufIcon(icons.Folder(folder.Color.Uint32()), FolderSize)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create the folder pixbuf")
	}

	i, err := gtk.ImageNewFromPixbuf(p)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create folder image")
	}

	folderEv, err := gtk.EventBoxNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create image event box")
	}
	folderEv.Add(i)
	folderEv.SetEvents(int(gdk.BUTTON_PRESS_MASK))

	// Add the event box image into the main box
	mainBox.Add(folderEv)

	guildList, err := gtk.ListBoxNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create list")
	}
	guildList.SetActivateOnSingleClick(true)

	folderRev, err := gtk.RevealerNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create revealer")
	}
	folderRev.Add(guildList)

	// On click, toggle revealer
	folderEv.Connect("button_press_event", func() {
		folderRev.SetRevealChild(!folderRev.GetRevealChild())
	})

	// Add the revealer into the main box
	mainBox.Add(folderRev)

	f := &Guild{
		// Iter:  store.Append(nil),
		// Store: store,
		IWidget: mainEv,
		Folder: &GuildFolder{
			Revealer: folderRev,
			List:     guildList,
			Guilds:   make([]*Guild, 0, len(folder.GuildIDs)),
		},

		ID:   folder.ID,
		Name: folder.Name,
	}

	for _, id := range folder.GuildIDs {
		g, err := s.Guild(id)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get guild ID"+id.String())
		}

		r, err := newGuildRow(*g)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load guild "+g.Name)
		}

		f.Folder.Guilds = append(f.Folder.Guilds, r)
		guildList.Add(r.IWidget)
	}

	return f, nil
}
