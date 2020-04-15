package emojis

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const EmojiSize = 28

type Picker struct {
	*gtk.Popover
	Main *gtk.Box

	Search *gtk.Entry
	search string

	PageView   *gtk.Stack
	Scroll     *gtk.ScrolledWindow
	MainPage   MainPage   // page 1 "main"
	SearchPage SearchPage // page 2 "search"
}

// MainPage contains sections, which has all emojis.
type MainPage struct {
	*gtk.ListBox
	Sections []*Section

	picker  *Picker
	current int // used to track the last page
	click   func(string)
}

type SearchPage struct {
	// *gtk.FlowBox

	// // all emojis, basically copy of mainpage's emojis
	// emojis  []*gtk.Button
	// visible []int
}

type Section struct {
	*RevealerRow
	Body *gtk.FlowBox

	shiftHeld bool

	Emojis  []discord.Emoji
	emojis  []*gtk.Image
	loaded  moreatomic.Serial
	stopped moreatomic.Bool
}

type Spawner struct {
	// Optional.
	Refocus interface{ GrabFocus() }

	state  *ningen.State
	click  func(string)
	opened *Picker
}

func New(s *ningen.State, click func(string)) *Spawner {
	return &Spawner{
		state: s,
		click: click,
	}
}

func (s *Spawner) Spawn(relative gtk.IWidget, currentGuild discord.Snowflake) *Picker {
	// Destroy the old picker if it's opened:
	if s.opened != nil {
		s.opened.Destroy()
	}

	picker := s.newPicker(relative, currentGuild)
	picker.Connect("destroy", func() {
		// Delete the global reference:
		s.opened = nil

		if s.Refocus != nil {
			s.Refocus.GrabFocus()
		}
	})

	return picker
}

func (s *Spawner) newPicker(r gtk.IWidget, currentGuild discord.Snowflake) *Picker {
	picker := &Picker{}
	picker.PageView, _ = gtk.StackNew()
	picker.Popover, _ = gtk.PopoverNew(r)
	picker.Scroll, _ = gtk.ScrolledWindowNew(nil, nil)
	picker.Search, _ = gtk.EntryNew()
	picker.Main, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	gtkutils.InjectCSSUnsafe(picker.Popover, "emojiview", "")

	picker.Search.Connect("changed", picker.searched)
	picker.Popover.Connect("closed", picker.Destroy)

	picker.PageView.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	picker.PageView.SetTransitionDuration(75)

	picker.Scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_EXTERNAL)
	picker.Scroll.SetProperty("propagate-natural-height", true)
	picker.Scroll.SetProperty("min-content-height", 400)
	picker.Scroll.SetProperty("max-content-height", 400)

	page := MainPage{click: s.click, picker: picker}
	page.ListBox, _ = gtk.ListBoxNew()

	page.SetSelectionMode(gtk.SELECTION_NONE)

	// search := SearchPage{}

	picker.MainPage = page
	// picker.SearchPage = search

	// Make a viewport that doesn't automatically scroll.
	vp := newStaticViewport()
	vp.Add(picker.Main)

	picker.Scroll.Add(vp)
	picker.Popover.Add(picker.Scroll)

	// picker.Main.Add(picker.Search)
	picker.Main.Add(picker.PageView)

	picker.PageView.AddNamed(picker.MainPage, "main")
	// picker.PageView.AddNamed(picker.SearchPage, "search")

	picker.ShowAll()

	// Make all guild pages
	guildEmojis := s.state.SearchEmojis(currentGuild)
	page.Sections = make([]*Section, 0, len(guildEmojis))

	// Adding 100 guilds right now, since it's not that expensive.
	for i, group := range guildEmojis {
		s := Section{
			Emojis: guildEmojis[i].Emojis,
		}

		header := newHeader(group.Name, group.IconURL())

		s.Body, _ = gtk.FlowBoxNew()
		s.Body.Show()
		s.Body.SetHomogeneous(true)
		s.Body.SetSelectionMode(gtk.SELECTION_SINGLE)
		s.Body.SetActivateOnSingleClick(true)
		s.Body.SetMaxChildrenPerLine(10) // from Discord
		s.Body.SetMinChildrenPerLine(10) // from Discord Mobile

		s.RevealerRow = newRevealerRow(header, s.Body, page.reveal)
		s.stopped.Set(true)

		// Add the placeholder first.
		page.Insert(s, i)
		page.Sections = append(page.Sections, &s)
	}

	// Load the first page.
	page.reveal(0)

	s.opened = picker
	return picker
}

func (p *MainPage) reveal(i int) {
	var revealed bool

	for j, section := range p.Sections {
		if j == i {
			revealed = section.Revealer.GetRevealChild()

			// If the current section is not opened, then actually try and
			// uncollapse others, then load it.
			if !revealed {
				continue
			}
		}

		if !section.stopped.Get() {
			section.stopped.Set(true)
			section.Revealer.SetRevealChild(false)
		}
	}

	// Exit, we don't want to re-open the collapsed revealer.
	if revealed {
		return
	}

	section := p.Sections[i]
	section.load(p.click, p.picker.Popover.Hide)
	section.Revealer.SetRevealChild(true)
}

func (s *Section) load(onClick func(string), hide func()) {
	s.stopped.Set(false)

	// Initialize empty images if we haven't yet.
	if s.emojis == nil {
		// Pre-allocate.
		s.emojis = make([]*gtk.Image, len(s.Emojis))

		// Construct all images first:
		for i := range s.Emojis {
			img, _ := gtk.ImageNew()
			img.Show()
			img.SetTooltipText(s.Emojis[i].Name)
			gtkutils.ImageSetIcon(img, "image-missing", EmojiSize)

			s.emojis[i] = img

			// Append the button.
			s.Body.Add(img)
		}

		s.Body.Connect("child-activated", func(f *gtk.FlowBox, c *gtk.FlowBoxChild) {
			onClick(s.Emojis[c.GetIndex()].String())
			// Is shift being held?
			if s.shiftHeld {
				hide()
			}
		})
		s.Body.Connect("button-press-event", func(f *gtk.FlowBox, ev *gdk.Event) bool {
			evk := gdk.EventButtonNewFromEvent(ev)
			const shift = uint(gdk.GDK_SHIFT_MASK)

			// Is shift being held?
			s.shiftHeld = evk.State()&shift != shift

			// Pass all events through.
			return false
		})
	}

	// Render the rest in a goroutine, sequentially.
	go func() {
		for i := s.loaded.Get(); i < len(s.Emojis); i = s.loaded.Incr() {
			// Check if we should stos.
			if s.stopped.Get() {
				return
			}

			var emoji = s.Emojis[i]
			var img = s.emojis[i]
			var url = md.EmojiURL(emoji.ID.String(), emoji.Animated)

			if err := cache.SetImageScaled(url, img, EmojiSize, EmojiSize); err != nil {
				log.Errorln("Failed to get emoji:", err)
			}
		}
	}()
}

func (p *Picker) searched() {
	// p.search, _ = p.Search.GetText()
	// if p.search == "" {
	// 	p.PageView.SetVisibleChild(p.MainPage)
	// 	return
	// }

	// p.PageView.SetVisibleChild(p.SearchPage)

	// p.search = strings.ToLower(p.search)
	// p.SearchPage.search(p.search)
}

// func (s *SearchPage) search(text string) {
// 	// Remove old entries.
// 	for _, v := range s.visible {
// 		s.Remove(s.emojis[v])
// 	}
// 	s.visible = s.visible[:0]

// 	for i, e := range s.emojis {
// 		if n, _ := e.GetName(); strings.Contains(n, text) {
// 			s.Add(e)
// 			s.visible = append(s.visible, i)
// 		}
// 	}
// }
