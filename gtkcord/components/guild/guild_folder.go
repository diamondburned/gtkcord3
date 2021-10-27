package guild

import (
	"fmt"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/ningen/v2"
)

type GuildFolder struct {
	// Row that belongs to the parent list.
	*RevealerRow

	Icon *GuildFolderIcon
	Name string

	// Child list.
	List *gtk.ListBox

	Guilds   []*Guild
	Revealed bool

	stateClass string
}

func newGuildFolder(
	s *ningen.State, folder gateway.GuildFolder, onSelect func(g *Guild)) *GuildFolder {

	if folder.Color == 0 {
		folder.Color = 0x7289DA
	}

	guildList := gtk.NewListBox()
	guildList.SetActivateOnSingleClick(true)

	f := &GuildFolder{
		List:   guildList,
		Name:   folder.Name,
		Guilds: make([]*Guild, 0, len(folder.GuildIDs)),
	}

	// Bind the child list independent of the parent list.
	guildList.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		i := r.Index()
		f.unselectAll(i)

		row := f.Guilds[i]
		row.Unread.SetActive(true)
		onSelect(row)
	})

	// Take care of the icon part after we've created our guilds.
	f.Icon = newGuildFolderIcon()

	// On click, toggle revealer.
	f.RevealerRow = newRevealerRow(f.Icon, guildList, func(reveal bool) {
		// Expand/collapse the icon
		f.Icon.setReveal(reveal)
	})

	for _, id := range folder.GuildIDs {
		r := newGuildRow(s, id, f)
		r.Parent = f

		f.Guilds = append(f.Guilds, r)
		f.List.Add(r)
	}

	f.Icon.load(f.Guilds)

	gtkutils.InjectCSS(f.RevealerRow, "guild-folder", "")
	// Folder.Style, _ = rev.GetStyleContext()
	// Folder.Style.AddClass("guild-folder")

	// Show name on hover.
	BindNameDirect(f.RevealerRow.Button, f.RevealerRow.Strip, &f.Name)

	// Color time.
	color := fmt.Sprintf("#%06X", folder.Color.Uint32())

	// Color the folder icon.
	gtkutils.InjectCSS(f.Icon.Folder, "", `* { color: `+color+`; }`)
	// Color the collapsed folder background.
	gtkutils.AddCSS(f.Icon.Style, `
		*.collapsed {
			/* We have to use mix because alpha breaks with border-radius */
			background-color: mix(@theme_bg_color, `+color+`, 0.4);
		}
	`)

	// Add some room:
	f.RevealerRow.ListBoxRow.SetSizeRequest(IconSize+IconPadding*3, IconSize+IconPadding/2)
	gtkutils.Margin2(f.RevealerRow, IconPadding/2, 0)

	return f
}

func (f *GuildFolder) unselectAll(except int) {
	for i, r := range f.Guilds {
		if i == except {
			continue
		}
		r.Unread.SetActive(false)
	}
}

func (f *GuildFolder) setUnread(unread, pinged bool) {
	// If unread but current folder is pinged, then set pinged to true.
	// if unread && !pinged && f.stateClass == "pinged" {
	// 	pinged = true
	// }

	// Check all children guilds
	if !unread || !pinged {
		for _, g := range f.Guilds {
			switch _unread, _pinged := g.Unread.State(); {
			case _pinged:
				pinged = true
				fallthrough
			case _unread:
				unread = true
			}
		}
	}

	switch {
	case pinged:
		f.Strip.SetPinged()
	case unread:
		f.Strip.SetUnread()
	default:
		f.Strip.SetRead()
	}
}

type GuildFolderIcon struct {
	// Main stack, switches between "guilds" and "folder"
	*gtk.Stack
	Style *gtk.StyleContext

	Guilds *gtk.Grid            // contains 4 images always.
	Images [4]*roundimage.Image // first 4 of folder.Guilds

	folder []*Guild
	Folder *gtk.Image
}

func newGuildFolderIcon() *GuildFolderIcon {
	i := &GuildFolderIcon{}

	i.Stack = gtk.NewStack()
	i.Stack.SetTransitionType(gtk.StackTransitionTypeSlideUp) // unsure
	i.Stack.SetSizeRequest(IconSize, IconSize)

	i.Style = i.Stack.StyleContext()
	i.Style.AddClass("collapsed") // used for coloring

	i.Folder = gtk.NewImageFromIconName("folder-symbolic", FolderSize)
	i.Folder.SetPixelSize(FolderSize)

	return i
}

func (i *GuildFolderIcon) load(guilds []*Guild) {
	if i.Guilds != nil {
		panic("GuildFolderIcon.load called twice")
	}

	i.Guilds = gtk.NewGrid()
	i.Guilds.SetHAlign(gtk.AlignCenter)
	i.Guilds.SetVAlign(gtk.AlignCenter)
	i.Guilds.SetRowSpacing(4) // calculated from Discord
	i.Guilds.SetRowHomogeneous(true)
	i.Guilds.SetColumnSpacing(4)
	i.Guilds.SetColumnHomogeneous(true)

	// Make dummy images.
	for ix := range i.Images {
		img := roundimage.NewImage(0)
		img.SetSizeRequest(16, 16)

		i.Images[ix] = img
	}

	// Set the dummy images in a grid.
	// [0] [1]
	// [2] [3]
	i.Guilds.Attach(i.Images[0], 0, 0, 1, 1)
	i.Guilds.Attach(i.Images[1], 1, 0, 1, 1)
	i.Guilds.Attach(i.Images[2], 0, 1, 1, 1)
	i.Guilds.Attach(i.Images[3], 1, 1, 1, 1)

	// Asynchronously fetch the icons.
	for ix := 0; ix < len(guilds) && ix < 4; ix++ {
		if guilds[ix].IconURL == "" {
			i.Images[ix].SetInitials(guilds[ix].Name)
			continue
		}

		url := guilds[ix].IconURL + "?size=64"
		cache.SetImageURLScaled(i.Images[ix], url, 16, 16)
	}

	// Add things together.
	i.Stack.AddNamed(i.Guilds, "guilds")
	i.Stack.AddNamed(i.Folder, "folder")
}

// called with revealer
func (i *GuildFolderIcon) setReveal(reveal bool) {
	if reveal {
		// show the folder.
		i.Stack.SetVisibleChildName("folder")
		i.Style.RemoveClass("collapsed")
	} else {
		// show the guilds
		i.Stack.SetVisibleChildName("guilds")
		i.Style.AddClass("collapsed")
	}
}

type RevealerRow struct {
	*gtk.ListBoxRow
	Strip    *UnreadStrip
	Button   *gtk.Button
	Revealer *gtk.Revealer
}

func newRevealerRow(button, reveal gtk.Widgetter, click func(reveal bool)) *RevealerRow {
	r := gtk.NewRevealer()
	r.SetTransitionType(gtk.RevealerTransitionTypeSlideDown)
	r.SetRevealChild(false)
	r.Add(reveal)
	r.Show()

	btn := gtk.NewButton()
	btn.SetHAlign(gtk.AlignCenter)
	btn.SetVAlign(gtk.AlignCenter)
	btn.SetRelief(gtk.ReliefNone)
	btn.Add(button)
	btn.Show()

	// Wrap both the widget child and the revealer
	b := gtk.NewBox(gtk.OrientationVertical, 0)
	b.Add(btn)
	b.Add(r)
	b.Show()

	// Wrap the stack inside the unread strip overlay.
	strip := NewUnreadStrip(b)

	row := gtk.NewListBoxRow()
	row.Show()
	row.SetHAlign(gtk.AlignCenter)
	row.SetVAlign(gtk.AlignCenter)
	row.SetSelectable(false)
	row.Add(strip)

	btn.Connect("clicked", func() {
		reveal := !r.RevealChild()
		r.SetRevealChild(reveal)
		click(reveal)
		strip.SetSuppress(reveal)
	})

	return &RevealerRow{row, strip, btn, r}
}
