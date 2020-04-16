package emojis

import (
	"context"
	"runtime"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/gtkcord3/internal/moreatomic"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"golang.org/x/sync/semaphore"
)

// global semaphores are better
var jobSemaphore = semaphore.NewWeighted(int64(runtime.GOMAXPROCS(-1)))

// MainPage contains sections, which has all emojis.
type MainPage struct {
	*gtk.ScrolledWindow
	ListBox  *gtk.ListBox
	Sections []*Section

	picker  *Picker
	current int // used to track the last page
	click   func(string)
}

func newMainPage(p *Picker, click func(string)) MainPage {
	page := MainPage{click: click, picker: p}
	page.ScrolledWindow, _ = gtk.ScrolledWindowNew(nil, nil)
	page.ListBox, _ = gtk.ListBoxNew()

	page.ListBox.SetSelectionMode(gtk.SELECTION_NONE)

	page.ScrolledWindow.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_EXTERNAL)
	page.ScrolledWindow.SetProperty("propagate-natural-height", true)
	page.ScrolledWindow.SetProperty("min-content-height", 400)
	page.ScrolledWindow.SetProperty("max-content-height", 400)
	page.ScrolledWindow.Add(page.ListBox)

	return page
}

func (page *MainPage) init(guildEmojis []ningen.GuildEmojis) {
	page.Sections = make([]*Section, 0, len(guildEmojis))

	// Adding 100 guilds right now, since it's not that expensive.
	for i, group := range guildEmojis {
		s := Section{
			Emojis: guildEmojis[i].Emojis,
		}

		header := newHeader(group.Name, group.IconURL())

		s.Body = newFlowBox()
		s.RevealerRow = newRevealerRow(header, s.Body, page.reveal)
		s.stopped.Set(true)

		// Add the placeholder first.
		page.ListBox.Insert(s, i)
		page.Sections = append(page.Sections, &s)
	}

	// Load the first page.
	page.reveal(0)
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

type Section struct {
	*RevealerRow
	Body *gtk.FlowBox

	shiftHeld bool

	Emojis  []discord.Emoji
	emojis  []*gtk.Image
	loaded  moreatomic.Serial
	stopped moreatomic.Bool
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

			// Allocate a timeout context.
			ctx, cancel := context.WithTimeout(context.TODO(), 7*time.Second)

			// Throttle this job.
			if err := jobSemaphore.Acquire(ctx, 1); err != nil {
				// Timed out, exit.
				cancel()
				return
			}

			// Goroutines will pertain even on tab change. This is intentional.
			go func(i int) {
				// Release the job throttler.
				defer jobSemaphore.Release(1)
				// Complete the used context.
				defer cancel()

				var emoji = s.Emojis[i]
				var img = s.emojis[i]
				var url = md.EmojiURL(emoji.ID.String(), emoji.Animated)

				// Set a custom timeout to avoid clogging up other images.
				if err := cache.SetImageScaledContext(ctx, url, img, EmojiSize, EmojiSize); err != nil {
					log.Errorln("Failed to get emoji:", err)
				}
			}(i)
		}
	}()
}
