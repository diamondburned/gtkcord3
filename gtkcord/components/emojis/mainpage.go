package emojis

import (
	"context"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/cache"
	"github.com/diamondburned/ningen/v2/md"
	"github.com/diamondburned/ningen/v2/states/emoji"
)

// MainPage contains sections, which has all emojis.
type MainPage struct {
	*gtk.ScrolledWindow
	Main     *gtk.Box
	Sections []*Section

	vadj *gtk.Adjustment
	hadj *gtk.Adjustment

	picker  *Picker
	current int // used to track the last page
	click   func(string)
}

func newMainPage(p *Picker, click func(string)) MainPage {
	page := MainPage{click: click, picker: p}
	page.ScrolledWindow = gtk.NewScrolledWindow(nil, nil)
	page.Main = gtk.NewBox(gtk.OrientationVertical, 0)

	page.ScrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyExternal)
	page.ScrolledWindow.SetPropagateNaturalHeight(true)
	page.ScrolledWindow.SetMinContentHeight(400)
	page.ScrolledWindow.SetMaxContentHeight(400)
	page.ScrolledWindow.Add(page.Main)

	page.vadj = page.ScrolledWindow.VAdjustment()
	page.hadj = page.ScrolledWindow.HAdjustment()

	return page
}

func (p *MainPage) init(guildEmojis []emoji.Guild) {
	p.Sections = make([]*Section, 0, len(guildEmojis))

	// Adding 100 guilds right now, since it's not that expensive.
	for i, group := range guildEmojis {
		i := i

		s := newSection(group)
		s.hide = p.picker.Popover.Popdown
		s.clicked = p.click
		s.ConnectRevealChild(func(bool) { p.unrevealOthers(i) })

		// Bind the revealer to the scrolled window so that expands can focus.
		s.Revealer.SetFocusHAdjustment(p.hadj)
		s.Revealer.SetFocusVAdjustment(p.vadj)

		// Add the placeholder first.
		p.Main.Add(s)
		p.Sections = append(p.Sections, s)
	}

	p.ShowAll()
}

func (p *MainPage) unrevealOthers(ix int) {
	for i, section := range p.Sections {
		if i != ix {
			section.Revealer.SetRevealChild(false)
		}
	}
}

type Section struct {
	*RevealerBox
	Button *gtk.ToggleButton
	Body   *gtk.FlowBox

	Emojis []discord.Emoji
	emojis []*gtk.Image

	clicked func(string)
	hide    func()

	ctx    context.Context
	cancel context.CancelFunc

	lastLoaded int
	shiftHeld  bool
}

type sectionLoadState struct {
	ctx    context.Context
	loaded int
}

func newSection(group emoji.Guild) *Section {
	s := Section{
		Emojis: group.Emojis,
	}

	s.Button = newHeaderButton(group.Name, group.IconURL())
	s.Body = newFlowBox()
	s.RevealerBox = newRevealerBox(s.Button, s.Body)
	s.RevealerBox.ConnectUnmap(func() {
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
	})

	s.RevealerBox.Revealer.Connect("notify::reveal-child", func() {
		if s.Revealer.RevealChild() {
			if s.cancel == nil {
				s.ctx, s.cancel = context.WithCancel(context.Background())
				s.load()
			}
		} else {
			if s.cancel != nil {
				s.cancel()
				s.cancel = nil
			}
		}
	})

	return &s
}

// init initializes empty images if we haven't yet.
func (s *Section) init() {
	if s.emojis != nil {
		return
	}

	// Pre-allocate.
	s.emojis = make([]*gtk.Image, len(s.Emojis))

	// Construct all images first:
	for i := range s.Emojis {
		img := gtk.NewImage()
		img.SetTooltipText(s.Emojis[i].Name)

		s.emojis[i] = img

		// Append the button.
		s.Body.Add(img)
	}

	s.Body.ShowAll()

	s.Body.Connect("child-activated", func(c *gtk.FlowBoxChild) {
		s.clicked(s.Emojis[c.Index()].String())
		if !s.shiftHeld {
			s.hide()
		}
	})

	// Intercept a button click instead. It's better than listening to
	// keypresses.
	s.Body.Connect("button-press-event", func(f *gtk.FlowBox, ev *gdk.Event) bool {
		evk := ev.AsButton()
		const shift = gdk.ShiftMask

		// Is shift being held?
		s.shiftHeld = evk.State()&shift == shift

		// Pass all events through.
		return false
	})
}

func (s *Section) load() {
	s.init()

	ctx := s.ctx
	lastLoaded := s.lastLoaded

	// Render the rest in a goroutine, sequentially.
	/* TODO: INSPECT ME */
	go func() {
		for lastLoaded < len(s.Emojis) {
			select {
			case <-ctx.Done():
				break
			default:
				// ok
			}

			emoji := s.Emojis[lastLoaded]

			img := s.emojis[lastLoaded]
			url := md.EmojiURL(emoji.ID.String(), emoji.Animated)

			cache.SetImageURLScaledContext(ctx, img, url, Size, Size)

			lastLoaded++
		}

		glib.IdleAdd(func() {
			if lastLoaded > s.lastLoaded {
				s.lastLoaded = lastLoaded
			}
		})
	}()
}
