package quickswitcher

import (
	"html"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const IconSize = 24

// Any more and Gtk becomes super slow.
const MaxEntries = 80

type Spawner struct {
	State *ningen.State

	OnGuild   func(discord.Snowflake)
	OnChannel func(_, _ discord.Snowflake)
	OnFriend  func(discord.Snowflake)
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
	OnGuild   func(guildid discord.Snowflake)
	OnChannel func(channel, guild discord.Snowflake)
	OnFriend  func(userid discord.Snowflake)

	// reusable slices
	list []Entry
	done chan struct{}

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
	GuildID   discord.Snowflake
	ChannelID discord.Snowflake // could be DM, visible as one
	FriendID  discord.Snowflake // only called if needed a new channel
}

func NewDialog(s *ningen.State) *Dialog {
	d, _ := gtk.DialogNew()
	d.SetModal(true)
	d.SetTransientFor(window.Window)
	d.SetDefaultSize(500, 400)

	gtkutils.InjectCSSUnsafe(d, "quickswitcher", "")

	d.Connect("response", func(_ *gtk.Dialog, resp gtk.ResponseType) {
		if resp == gtk.RESPONSE_DELETE_EVENT {
			d.Destroy()
		}
	})

	// Header

	header, _ := gtk.HeaderBarNew()
	header.Show()
	header.SetShowCloseButton(true)

	entry, _ := gtk.EntryNew()
	entry.Show()
	entry.GrabFocus()
	entry.SetPlaceholderText("Search anything")
	entry.SetSizeRequest(400, -1)

	// Custom Title allows Entry to be centered.
	header.SetCustomTitle(entry)

	d.SetTitlebar(header)

	// Body

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Show()
	sw.SetSizeRequest(400, -1)
	sw.SetHAlign(gtk.ALIGN_CENTER)
	sw.SetVExpand(true)

	list, _ := gtk.ListBoxNew()
	list.Show()
	list.SetVExpand(true)
	sw.Add(list)

	c, _ := d.GetContentArea()
	d.Remove(c)
	d.Add(sw)

	dialog := &Dialog{
		Dialog: d,
		Entry:  entry,
		List:   list,
		state:  s,
		done:   make(chan struct{}),
	}

	list.Connect("row-activated", dialog.onActivate)
	list.SetSelectionMode(gtk.SELECTION_SINGLE)

	entry.Connect("key-press-event", func(e *gtk.Entry, ev *gdk.Event) bool {
		evKey := gdk.EventKeyNewFromEvent(ev)
		if evKey.Type() != gdk.EVENT_KEY_PRESS {
			return false
		}
		if len(dialog.list) == 0 {
			return false
		}

		switch evKey.KeyVal() {
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
		list.GetSelectedRow().Activate()
	})
	entry.Connect("changed", func() {
		t, err := entry.GetText()
		if err != nil {
			log.Errorln("Failed to get text from quickswitcher:", err)
			return
		}

		dialog.onEntryChange(strings.ToLower(t))
	})

	// Populate in the background.
	go func() {
		dialog.populateEntries()
		close(dialog.done) // special behavior, all <-done unblocks instantly
	}()

	return dialog
}

func (d *Dialog) Down() {
	i := d.List.GetSelectedRow().GetIndex()
	i++
	if i >= len(d.visible) {
		i = 0
	}
	d.List.SelectRow(d.rows[d.visible[i]].ListBoxRow)
}

func (d *Dialog) Up() {
	i := d.List.GetSelectedRow().GetIndex()
	i--
	if i < 0 {
		i = len(d.visible) - 1
	}
	d.List.SelectRow(d.rows[d.visible[i]].ListBoxRow)
}

func (d *Dialog) onActivate(_ *gtk.ListBox, r *gtk.ListBoxRow) {
	i := r.GetIndex()
	if i < 0 || i >= len(d.visible) {
		// wtf?
		return
	}

	// Close the dialog:
	d.Destroy()

	switch entry := d.list[d.visible[i]]; {
	case entry.ChannelID.Valid():
		d.OnChannel(entry.ChannelID, entry.GuildID)
	case entry.GuildID.Valid():
		d.OnGuild(entry.GuildID)
	}
}

func (d *Dialog) onEntryChange(word string) {
	// Wait for population to be done.
	<-d.done

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

func (d *Dialog) populateEntries() {
	// Search for guilds first:
	guilds, _ := d.state.Store.Guilds()

	// Pre-grow the slice.
	d.list = make([]Entry, 0, len(guilds))
	for _, g := range guilds {
		d.list = append(d.list, Entry{
			IconURL:     g.IconURL(),
			PrimaryText: g.Name,
			GuildID:     g.ID,
		})
	}

	for _, g := range guilds {
		// If somehow the guild is broken.
		if g.Name == "" || !g.ID.Valid() {
			continue
		}

		c, err := d.state.Store.Channels(g.ID)
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
		d.list = append(d.list, channels...)
	}

	dm, _ := d.state.PrivateChannels()

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
	d.list = append(d.list, dmEntries...)

	// Form long strings:
	for i, l := range d.list {
		d.list[i].longString = strings.ToLower(l.PrimaryText + l.SecondaryText + l.RightText)
	}

	// Pre-allocate (arbitrarily) half the length of list for visible:
	d.visible = make([]int, 0, len(d.list)/2)
	d.rows = make(map[int]*Row, len(d.list)/2)
}

func generateRow(i int, e *Entry) *Row {
	r := Row{index: i}
	r.ListBoxRow, _ = gtk.ListBoxRowNew()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	r.ListBoxRow.Add(b)

	switch {
	case e.IconChar > 0:
		l, _ := gtk.LabelNew(`<span size="larger" weight="bold">` + string(e.IconChar) + `</span>`)
		l.SetUseMarkup(true)
		l.SetSizeRequest(IconSize, IconSize)
		l.SetHAlign(gtk.ALIGN_CENTER)
		l.SetVAlign(gtk.ALIGN_CENTER)
		gtkutils.Margin2(l, 2, 8)

		b.Add(l)

	default:
		i, _ := gtk.ImageNew()
		i.SetSizeRequest(IconSize, IconSize)
		i.SetHAlign(gtk.ALIGN_CENTER)
		i.SetVAlign(gtk.ALIGN_CENTER)
		gtkutils.Margin2(i, 2, 8)
		gtkutils.ImageSetIcon(i, "user-available-symbolic", IconSize)

		b.Add(i)

		if e.IconURL != "" {
			go cache.SetImageScaled(e.IconURL+"?size=32", i, IconSize, IconSize, cache.Round)
		}
	}

	// Generate primary text
	p, _ := gtk.LabelNew(`<span weight="bold">` + html.EscapeString(e.PrimaryText) + `</span>`)
	p.SetUseMarkup(true)
	p.SetEllipsize(pango.ELLIPSIZE_END)
	gtkutils.Margin2(p, 2, 4)

	b.Add(p)

	// Is there a secondary text?
	if e.SecondaryText != "" {
		s, _ := gtk.LabelNew(e.SecondaryText)
		s.SetOpacity(0.8)
		gtkutils.Margin2(s, 2, 4)

		b.Add(s)
	}

	// Is there a right text?
	if e.RightText != "" {
		r, _ := gtk.LabelNew(
			`<span weight="bold" size="smaller">` + html.EscapeString(e.RightText) + `</span>`)
		r.SetUseMarkup(true)
		r.SetHExpand(true)
		r.SetHAlign(gtk.ALIGN_END)
		gtkutils.Margin2(r, 2, 4)

		b.Add(r)
	}

	r.ShowAll()
	return &r
}
