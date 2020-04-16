package emojis

import (
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const EmojiSize = 28

type Picker struct {
	*gtk.Popover
	Main *gtk.Box

	Search *handy.SearchBar
	search *gtk.Entry

	PageView   *gtk.Stack
	MainPage   MainPage   // page 1 "main"
	SearchPage SearchPage // page 2 "search"
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
	picker.Main, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	picker.search, _ = gtk.EntryNew()
	picker.Search = handy.SearchBarNew()

	gtkutils.InjectCSSUnsafe(picker.Popover, "emojiview", "")

	picker.search.Connect("changed", picker.entryChanged)
	picker.Popover.Connect("closed", picker.Destroy)

	picker.PageView.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	picker.PageView.SetTransitionDuration(75)

	picker.MainPage = newMainPage(picker, s.click)
	picker.SearchPage = newSearchPage(picker)

	picker.Main.Add(picker.Search)
	picker.Main.Add(picker.PageView)

	picker.Popover.Add(picker.Main)
	picker.Popover.Connect("key-press-event", func(p *gtk.Popover, ev *gdk.Event) bool {
		return picker.Search.HandleEvent(ev)
	})
	picker.Search.SetSearchMode(true)
	picker.Search.Add(picker.search)

	picker.PageView.AddNamed(picker.MainPage, "main")
	picker.PageView.AddNamed(picker.SearchPage, "search")

	// Go back to main page.
	picker.PageView.SetVisibleChildName("main")
	picker.ShowAll()

	// Make all guild pages
	guildEmojis := s.state.SearchEmojis(currentGuild)
	picker.MainPage.init(guildEmojis)

	s.opened = picker
	return picker
}

func (p *Picker) entryChanged(e *gtk.Entry) {
	text, _ := e.GetText()
	if text == "" {
		p.PageView.SetVisibleChild(p.MainPage)
		return
	}

	p.PageView.SetVisibleChild(p.SearchPage)
	p.SearchPage.search(strings.ToLower(text))
}
