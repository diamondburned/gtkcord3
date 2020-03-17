package quickswitcher

import (
	"html"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const IconSize = 24

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
	list      []Entry
	sGuilds   []discord.Guild
	sChannels []discord.Channel
	// sFriends  []discord.User

	// TODO: cache guilds and channels into huge slices.
}

type Entry struct {
	// will create if nil
	*gtk.ListBoxRow

	// | <Icon> <Primary> <Secondary>          <Right> |

	// Icon enum
	IconChar rune
	IconURL  string

	PrimaryText   string
	SecondaryText string
	RightText     string

	// enum
	GuildID   discord.Snowflake
	ChannelID discord.Snowflake // could be DM, displayed as one
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
			d.Hide()
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
	}

	list.Connect("row-activated", dialog.onActivate)

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

	return dialog
}

func (d *Dialog) Down() {
	i := d.List.GetSelectedRow().GetIndex()
	i++
	if i >= len(d.list) {
		i = 0
	}
	d.List.SelectRow(d.list[i].ListBoxRow)
}

func (d *Dialog) Up() {
	i := d.List.GetSelectedRow().GetIndex()
	i--
	if i < 0 {
		i = len(d.list) - 1
	}
	d.List.SelectRow(d.list[i].ListBoxRow)
}

func (d *Dialog) onActivate(_ *gtk.ListBox, r *gtk.ListBoxRow) {
	i := r.GetIndex()
	if i < 0 || i >= len(d.list) {
		// wtf?
		return
	}

	switch entry := d.list[i]; {
	case entry.ChannelID.Valid() && d.OnChannel != nil:
		go d.OnChannel(entry.ChannelID, entry.GuildID)
	case entry.GuildID.Valid() && d.OnGuild != nil:
		go d.OnGuild(entry.GuildID)
	}

	// Close the dialog:
	d.Hide()
}

func (d *Dialog) clear() {
	for _, l := range d.list {
		d.List.Remove(l)
	}
	d.list = d.list[:0]
}

func (d *Dialog) onEntryChange(word string) {
	d.clear()

	if word == "" {
		return
	}

	// Search for guilds first:
	g, _ := d.state.Store.Guilds()
	for _, g := range g {
		if contains(g.Name, word) {
			d.addEntry(Entry{
				IconURL:     g.IconURL(),
				PrimaryText: g.Name,
				GuildID:     g.ID,
			})
		}

		c, err := d.state.Store.Channels(g.ID)
		if err != nil {
			continue
		}

		for _, c := range c {
			switch c.Type {
			case discord.DirectMessage: // treat as user
				recip := c.DMRecipients[0]
				entry := Entry{
					PrimaryText:   recip.Username,
					SecondaryText: "#" + recip.Discriminator,
				}

				if !entry.contains(word) {
					continue
				}

				entry.IconURL = recip.AvatarURL()
				entry.ChannelID = c.ID
				d.addEntry(entry)

			default:
				entry := Entry{
					PrimaryText: c.Name,
					RightText:   g.Name,
				}

				if !entry.contains(word) {
					continue
				}

				entry.IconChar = '#'
				entry.ChannelID = c.ID
				entry.GuildID = c.GuildID
				d.addEntry(entry)
			}
		}
	}
}

func (d *Dialog) addEntry(entry Entry) {
	if entry.ListBoxRow == nil {
		entry.generateRow()
	}

	d.List.Insert(entry, -1)
	entry.Show()

	if len(d.list) == 0 {
		d.List.SelectRow(entry.ListBoxRow)
	}

	d.list = append(d.list, entry)
}

func (e *Entry) generateRow() {
	e.ListBoxRow, _ = gtk.ListBoxRowNew()
	e.ListBoxRow.Show()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.Show()
	e.ListBoxRow.Add(b)

	switch {
	case e.IconChar > 0:
		l, _ := gtk.LabelNew(`<span size="larger" weight="bold">` + string(e.IconChar) + `</span>`)
		l.SetUseMarkup(true)
		l.SetSizeRequest(IconSize, IconSize)
		l.SetHAlign(gtk.ALIGN_CENTER)
		l.SetVAlign(gtk.ALIGN_CENTER)
		gtkutils.Margin2(l, 2, 8)

		l.Show()
		b.Add(l)

	default:
		i, _ := gtk.ImageNew()
		i.SetSizeRequest(IconSize, IconSize)
		i.SetHAlign(gtk.ALIGN_CENTER)
		i.SetVAlign(gtk.ALIGN_CENTER)
		gtkutils.Margin2(i, 2, 8)
		gtkutils.ImageSetIcon(i, "user-available-symbolic", IconSize)

		i.Show()
		b.Add(i)

		if e.IconURL != "" {
			go cache.SetImageScaled(e.IconURL, i, IconSize, IconSize, cache.Round)
		}
	}

	// Generate primary text
	p, _ := gtk.LabelNew(`<span weight="bold">` + html.EscapeString(e.PrimaryText) + `</span>`)
	p.SetUseMarkup(true)
	p.SetEllipsize(pango.ELLIPSIZE_END)
	gtkutils.Margin2(p, 2, 4)

	p.Show()
	b.Add(p)

	// Is there a secondary text?
	if e.SecondaryText != "" {
		s, _ := gtk.LabelNew(e.SecondaryText)
		s.SetOpacity(0.8)
		gtkutils.Margin2(s, 2, 4)

		s.Show()
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

		r.Show()
		b.Add(r)
	}
}

func (e Entry) contains(match string) bool {
	return contains(e.PrimaryText+e.SecondaryText+e.RightText, match)
}

// match is assumed to already be lower-cased
func contains(full, match string) bool {
	return strings.Contains(strings.ToLower(full), match)
}
