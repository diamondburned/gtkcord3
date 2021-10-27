package emojis

import (
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/ningen/v2"
)

// Size is the emoji size.
const Size = 28

type Picker struct {
	*gtk.Popover
	Main *gtk.Box

	Search *handy.SearchBar
	search *gtk.Entry

	PageView   *gtk.Stack
	MainPage   MainPage   // page 1 "main"
	SearchPage SearchPage // page 2 "search"
	Error      *gtk.Label
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

func (s *Spawner) Spawn(relative gtk.Widgetter, currentGuild discord.GuildID) *Picker {
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

func (s *Spawner) newPicker(r gtk.Widgetter, currentGuild discord.GuildID) *Picker {
	picker := &Picker{}
	picker.PageView = gtk.NewStack()
	picker.Popover = gtk.NewPopover(r)
	picker.Main = gtk.NewBox(gtk.OrientationVertical, 0)

	picker.search = gtk.NewEntry()
	picker.Search = handy.NewSearchBar()

	gtkutils.InjectCSS(picker.Popover, "emojiview", "")

	picker.search.Connect("changed", picker.entryChanged)
	picker.Popover.Connect("closed", picker.Destroy)

	picker.PageView.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	picker.PageView.SetTransitionDuration(75)

	picker.MainPage = newMainPage(picker, s.click)
	picker.SearchPage = newSearchPage(picker)
	picker.Error = gtk.NewLabel("")

	picker.Main.Add(picker.Search)
	picker.Main.Add(picker.PageView)
	picker.Main.Add(picker.Error)
	picker.Popover.Add(picker.Main)

	picker.Search.SetSearchMode(true)
	picker.Search.Add(picker.search)

	picker.PageView.AddNamed(picker.MainPage, "main")
	picker.PageView.AddNamed(picker.SearchPage, "search")

	// Go back to main page.
	picker.PageView.SetVisibleChildName("main")
	picker.ShowAll()

	// Make all guild pages
	e, err := s.state.EmojiState.Get(currentGuild)
	if err != nil {
		log.Errorln("Failed to get emojis:", err)

		// Show the error visually.
		picker.Error.SetMarkup(`<span color="red">` + err.Error() + "</span>")
		picker.PageView.SetVisibleChild(picker.Error)

		// Early return.
		return picker
	}

	picker.MainPage.init(e)

	s.opened = picker
	return picker
}

func (p *Picker) entryChanged(e *gtk.Entry) {
	text := e.Text()
	if text == "" {
		p.PageView.SetVisibleChild(p.MainPage)
		return
	}

	p.PageView.SetVisibleChild(p.SearchPage)
	p.SearchPage.search(strings.ToLower(text))
}
