package quickswitcher

import (
	"html"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/roundimage"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/ningen/v2"
)

const IconSize = 24

// Any more and Gtk becomes super slow.
const MaxEntries = 80

type Spawner struct {
	State     *ningen.State
	OnGuild   func(discord.GuildID)
	OnChannel func(discord.ChannelID, discord.GuildID)
	OnFriend  func(discord.UserID)
}

func (s Spawner) Spawn() {
	d := NewDialog(s.State)
	d.OnGuild = s.OnGuild
	d.OnChannel = s.OnChannel
	d.OnFriend = s.OnFriend
	d.Show()
	d.Entry.GrabFocus()

	d.Run()
}

type Dialog struct {
	*gtk.Dialog
	Entry *gtk.Entry   // in header
	List  *gtk.ListBox // in dialog

	state *ningen.State

	// callback functions
	OnGuild   func(discord.GuildID)
	OnChannel func(discord.ChannelID, discord.GuildID)
	OnFriend  func(discord.UserID)

	// reusable slices
	list []Entry

	visible []int // key to rows
	rows    map[int]*Row

	// TODO: cache guilds and channels into huge slices.
}

type Row struct {
	*gtk.ListBoxRow
	index int
}

type Entry struct {
	// | <Icon> <Primary> <Secondary>          <Right> |

	// Icon enum
	IconChar rune
	IconURL  string

	PrimaryText   string
	SecondaryText string
	RightText     string
	longString    string

	// enum
	GuildID   discord.GuildID
	ChannelID discord.ChannelID // could be DM, visible as one
	FriendID  discord.UserID    // only called if needed a new channel
}

func NewDialog(s *ningen.State) *Dialog {
	d := gtk.NewDialog()
	d.SetModal(true)
	d.SetTransientFor(&window.Window.Window)
	d.SetDefaultSize(500, 400)

	gtkutils.InjectCSS(d, "quickswitcher", "")

	d.Connect("response", func(_ *gtk.Dialog, resp gtk.ResponseType) {
		if resp == gtk.ResponseDeleteEvent {
			d.Destroy()
		}
	})

	// Header

	header := gtk.NewHeaderBar()
	header.Show()
	header.SetShowCloseButton(true)

	entry := gtk.NewEntry()
	entry.Show()
	entry.GrabFocus()
	entry.SetPlaceholderText("Search anything")
	entry.SetSizeRequest(400, -1)

	// Custom Title allows Entry to be centered.
	header.SetCustomTitle(entry)

	d.SetTitlebar(header)

	// Body

	sw := gtk.NewScrolledWindow(nil, nil)
	sw.Show()
	sw.SetSizeRequest(400, -1)
	sw.SetHAlign(gtk.AlignCenter)
	sw.SetVExpand(true)

	list := gtk.NewListBox()
	list.Show()
	list.SetVExpand(true)
	sw.Add(list)

	d.Remove(d.ContentArea())
	d.Add(sw)

	dialog := &Dialog{
		Dialog: d,
		Entry:  entry,
		List:   list,
		state:  s,
	}

	list.Connect("row-activated", dialog.onActivate)
	list.SetSelectionMode(gtk.SelectionSingle)

	entry.Connect("key-press-event", func(e *gtk.Entry, ev *gdk.Event) bool {
		if ev.AsType() != gdk.KeyPressType {
			return false
		}
		if len(dialog.list) == 0 {
			return false
		}

		switch ev.AsKey().Keyval() {
		case gdk.KEY_Up:
			dialog.Up()
			return true
		case gdk.KEY_Down:
			dialog.Down()
			return true
		}

		return false
	})
	entry.Connect("activate", func() {
		list.SelectedRow().Activate()
	})
	entry.Connect("changed", func() {
		t := entry.Text()
		dialog.onEntryChange(strings.ToLower(t))
	})

	dialog.populateEntries()

	return dialog
}

func (d *Dialog) Down() {
	if len(d.visible) == 0 {
		return
	}
	i := d.List.SelectedRow().Index()
	i++
	if i >= len(d.visible) {
		i = 0
	}
	d.List.SelectRow(d.rows[d.visible[i]].ListBoxRow)
}

func (d *Dialog) Up() {
	if len(d.visible) == 0 {
		return
	}
	i := d.List.SelectedRow().Index()
	i--
	if i < 0 {
		i = len(d.visible) - 1
	}
	d.List.SelectRow(d.rows[d.visible[i]].ListBoxRow)
}

func (d *Dialog) onActivate(_ *gtk.ListBox, r *gtk.ListBoxRow) {
	i := r.Index()
	if i < 0 || i >= len(d.visible) {
		// wtf?
		return
	}

	// Close the dialog:
	d.Destroy()

	switch entry := d.list[d.visible[i]]; {
	case entry.ChannelID.IsValid():
		d.OnChannel(entry.ChannelID, entry.GuildID)
	case entry.GuildID.IsValid():
		d.OnGuild(entry.GuildID)
	}
}

func (d *Dialog) onEntryChange(word string) {
	if d.list == nil {
		return
	}

	// Remove old entries:
	d.List.UnselectAll() // unselect first.
	for _, l := range d.visible {
		d.List.Remove(d.rows[l])
	}
	d.visible = d.visible[:0]

	if word == "" {
		return
	}

	for i := range d.list {
		if !strings.Contains(d.list[i].longString, word) {
			continue
		}

		row, ok := d.rows[i]
		if !ok {
			row = generateRow(i, &d.list[i])
			d.rows[i] = row
		}

		d.List.Insert(row, -1)

		if len(d.visible) == 0 {
			d.List.SelectRow(row.ListBoxRow)
		}
		d.visible = append(d.visible, i)

		if len(d.visible) >= MaxEntries {
			// Stop searching. Any finer requires a better search string.
			return
		}
	}
}

func populateEntries(s *ningen.State) []Entry {
	s = s.Offline()

	// Search for guilds first:
	guilds, _ := s.Cabinet.Guilds()

	// Pre-grow the slice.
	list := make([]Entry, 0, len(guilds))
	for _, g := range guilds {
		list = append(list, Entry{
			IconURL:     g.IconURL(),
			IconChar:    '*', // for guild apparently
			PrimaryText: g.Name,
			GuildID:     g.ID,
		})
	}

	for _, g := range guilds {
		// If somehow the guild is broken.
		if g.Name == "" || !g.ID.IsValid() {
			continue
		}

		c, err := s.Cabinet.Channels(g.ID)
		if err != nil {
			continue
		}

		var channels = make([]Entry, 0, len(c))
		for _, c := range c {
			// Allow only text channels.
			if c.Type != discord.GuildText {
				continue
			}
			channels = append(channels, Entry{
				PrimaryText: c.Name,
				RightText:   g.Name,
				IconChar:    '#',
				ChannelID:   c.ID,
				GuildID:     c.GuildID,
			})
		}

		// Batch append:
		list = append(list, channels...)
	}

	dm, _ := s.PrivateChannels()

	// Prepare another slice to grow to:
	dmEntries := make([]Entry, 0, len(dm))

	for _, c := range dm {
		switch c.Type {
		case discord.DirectMessage:
			recip := c.DMRecipients[0]
			dmEntries = append(dmEntries, Entry{
				PrimaryText:   recip.Username,
				SecondaryText: "#" + recip.Discriminator,
				IconURL:       recip.AvatarURL(),
				IconChar:      '@',
				ChannelID:     c.ID,
			})

		default:
			dmEntries = append(dmEntries, Entry{
				PrimaryText: c.Name,
				IconChar:    '#',
				ChannelID:   c.ID,
			})
		}
	}

	// Batch append:
	list = append(list, dmEntries...)

	// Form long strings:
	for i, l := range list {
		list[i].longString = string(l.IconChar) +
			strings.ToLower(l.PrimaryText+l.SecondaryText+l.RightText)
	}

	return list
}

func (d *Dialog) populateEntries() {
	go func() {
		list := populateEntries(d.state)
		glib.IdleAdd(func() {
			d.list = list

			// Pre-allocate (arbitrarily) half the length of list for visible:
			d.visible = make([]int, 0, len(d.list)/2)
			d.rows = make(map[int]*Row, len(d.list)/2)
		})
	}()
}

func generateRow(i int, e *Entry) *Row {
	r := Row{index: i}
	r.ListBoxRow = gtk.NewListBoxRow()

	b := gtk.NewBox(gtk.OrientationHorizontal, 0)
	r.ListBoxRow.Add(b)

	switch {
	case e.IconURL != "":
		i := roundimage.NewImage(0)
		i.SetSizeRequest(IconSize, IconSize)
		i.SetFromIconName("user-available-symbolic", 0)
		i.SetPixelSize(IconSize)
		i.SetHAlign(gtk.AlignCenter)
		i.SetVAlign(gtk.AlignCenter)
		gtkutils.Margin2(i, 2, 8)

		b.Add(i)

		if e.IconURL != "" {
			cache.SetImageURLScaled(i, e.IconURL+"?size=32", IconSize, IconSize)
		}

	default:
		l := gtk.NewLabel(`<span size="larger" weight="bold">` + string(e.IconChar) + `</span>`)
		l.SetUseMarkup(true)
		l.SetSizeRequest(IconSize, IconSize)
		l.SetHAlign(gtk.AlignCenter)
		l.SetVAlign(gtk.AlignCenter)
		gtkutils.Margin2(l, 2, 8)

		b.Add(l)
	}

	// Generate primary text
	p := gtk.NewLabel(`<span weight="bold">` + html.EscapeString(e.PrimaryText) + `</span>`)
	p.SetUseMarkup(true)
	p.SetEllipsize(pango.EllipsizeEnd)
	gtkutils.Margin2(p, 2, 4)

	b.Add(p)

	// Is there a secondary text?
	if e.SecondaryText != "" {
		s := gtk.NewLabel(e.SecondaryText)
		s.SetOpacity(0.8)
		gtkutils.Margin2(s, 2, 4)

		b.Add(s)
	}

	// Is there a right text?
	if e.RightText != "" {
		r := gtk.NewLabel(
			`<span weight="bold" size="smaller">` + html.EscapeString(e.RightText) + `</span>`)
		r.SetUseMarkup(true)
		r.SetHExpand(true)
		r.SetHAlign(gtk.AlignEnd)
		gtkutils.Margin2(r, 2, 4)

		b.Add(r)
	}

	r.ShowAll()
	return &r
}
