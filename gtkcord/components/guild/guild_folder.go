package guild

import (
	"html"
	"sync"

	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/icons"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type GuildFolder struct {
	*gtk.ListBoxRow

	Revealer *gtk.Revealer
	List     *gtk.ListBox
	Style    *gtk.StyleContext

	Guilds   []*Guild
	Revealed bool

	classMutex sync.Mutex
	stateClass string
}

func newGuildFolder(
	s *ningen.State, folder gateway.GuildFolder, onSelect func(g *Guild)) (*GuildFolder, error) {

	if folder.Color == 0 {
		folder.Color = 0x7289DA
	}

	var Folder *GuildFolder

	semaphore.IdleMust(func() {
		mainBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		mainBox.SetTooltipMarkup("<b>" + html.EscapeString(folder.Name) + "</b>")

		r, _ := gtk.ListBoxRowNew()
		r.Add(mainBox)
		r.SetHAlign(gtk.ALIGN_CENTER)
		r.SetVAlign(gtk.ALIGN_CENTER)
		r.SetSizeRequest(IconSize+IconPadding*2, -1)
		r.SetSelectable(false)

		style, _ := r.GetStyleContext()
		style.AddClass("guild-folder")

		// Folder icon
		p, _ := icons.PixbufIcon(icons.Folder(folder.Color.Uint32()), FolderSize)
		i, _ := gtk.ImageNewFromPixbuf(p)

		folderEv, _ := gtk.EventBoxNew()
		folderEv.Add(i)
		folderEv.SetEvents(int(gdk.BUTTON_PRESS_MASK))

		// Add the event box image into the main box
		mainBox.Add(folderEv)

		guildList, _ := gtk.ListBoxNew()
		guildList.SetActivateOnSingleClick(true)

		folderRev, _ := gtk.RevealerNew()
		folderRev.Add(guildList)

		// Add the revealer into the main box
		mainBox.Add(folderRev)

		Folder = &GuildFolder{
			ListBoxRow: r,
			Revealer:   folderRev,
			List:       guildList,
			Style:      style,
			Guilds:     make([]*Guild, 0, len(folder.GuildIDs)),
		}

		// On click, toggle revealer
		folderEv.Connect("button_press_event", func() {
			Folder.Revealed = !folderRev.GetRevealChild()
			folderRev.SetRevealChild(Folder.Revealed)
		})

		guildList.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
			row := Folder.Guilds[r.GetIndex()]
			onSelect(row)
		})
	})

	var unread, pinged bool

	for _, id := range folder.GuildIDs {
		r, err := newGuildRow(s, id, nil, Folder)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load guild "+id.String())
		}
		r.Parent = Folder
		Folder.Guilds = append(Folder.Guilds, r)

		switch r.stateClass {
		case "pinged":
			pinged = true
			fallthrough
		case "unread":
			unread = true
		}
	}

	Folder.setUnread(unread, pinged)

	semaphore.IdleMust(func() {
		for _, r := range Folder.Guilds {
			r := r
			r.Row.SetSizeRequest(
				// We need to mult 4 div 3, since if we do full *2, the child
				// channels will be too big and expand the left bar.
				IconSize+IconPadding*4/3,
				IconSize+IconPadding*2,
			)

			Folder.List.Add(r)
		}
	})

	return Folder, nil
}

func (f *GuildFolder) setUnread(unread, pinged bool) {
	// If unread but current folder is pinged, then set pinged to true.
	// if unread && !pinged && f.stateClass == "pinged" {
	// 	pinged = true
	// }

	f.classMutex.Lock()
	defer f.classMutex.Unlock()

	// Check all children guilds
	if !unread || !pinged {
		for _, g := range f.Guilds {
			switch g.stateClass {
			case "pinged":
				pinged = true
				fallthrough
			case "unread":
				unread = true
			}
		}
	}

	switch {
	case pinged:
		f.setClass("pinged")
	case unread:
		f.setClass("unread")
	default:
		f.setClass("")
	}
}

func (f *GuildFolder) setClass(class string) {
	gtkutils.DiffClass(&f.stateClass, class, f.Style)
}
