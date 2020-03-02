package gtkcord

import (
	"html"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type GuildFolder struct {
	Revealer *gtk.Revealer
	List     *gtk.ListBox
	Guilds   []*Guild
	Revealed bool
}

func newGuildFolder(folder gateway.GuildFolder) (*Guild, error) {
	if folder.Color == 0 {
		folder.Color = 0x7289DA
	}

	mainBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create main box")
	}
	mainBox.SetTooltipMarkup("<b>" + html.EscapeString(folder.Name) + "</b>")

	r, _ := gtk.ListBoxRowNew()
	r.Add(mainBox)
	r.SetHAlign(gtk.ALIGN_CENTER)
	r.SetVAlign(gtk.ALIGN_CENTER)
	r.SetSizeRequest(IconSize+IconPadding*2, -1)

	style, _ := r.GetStyleContext()
	style.AddClass("guild-folder")

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

	// Add the revealer into the main box
	mainBox.Add(folderRev)

	f := &Guild{
		// Iter:  store.Append(nil),
		// Store: store,
		ExtendedWidget: r,
		Row:            r,
		Style:          style,
		Folder: &GuildFolder{
			Revealer: folderRev,
			List:     guildList,
			Guilds:   make([]*Guild, 0, len(folder.GuildIDs)),
		},

		ID:   folder.ID,
		Name: folder.Name,
	}

	// On click, toggle revealer
	folderEv.Connect("button_press_event", func() {
		f.Folder.Revealed = !folderRev.GetRevealChild()
		folderRev.SetRevealChild(f.Folder.Revealed)
	})

	guildList.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		row := f.Folder.Guilds[r.GetIndex()]
		go App.loadGuild(row)
	})

	for _, id := range folder.GuildIDs {
		r, err := newGuildRow(id)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load guild "+id.String())
		}
		r.Parent = f
		f.Folder.Guilds = append(f.Folder.Guilds, r)
	}

	must(func(f *Guild) {
		for _, r := range f.Folder.Guilds {
			r := r
			r.Row.SetSizeRequest(
				// We need to mult 4 div 3, since if we do full *2, the child
				// channels will be too big and expand the left bar.
				IconSize+IconPadding*4/3,
				IconSize+IconPadding*2,
			)

			f.Folder.List.Add(r)
		}
	}, f)

	return f, nil
}
