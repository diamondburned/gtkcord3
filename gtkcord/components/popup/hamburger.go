package popup

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/about"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
)

const HamburgerWidth = 240

// Hamburger is the widget that pops up when you click the top left button.
type Hamburger struct {
	gtkutils.ExtendedWidget
	Image   *gtk.Image
	Popover *Popover
}

func SpawnHamburger(relative gtk.IWidget, s *ningen.State) {
	p := NewPopover(relative)
	p.SetPosition(gtk.POS_RIGHT)

	body := NewStatefulPopupBody(s, s.Ready.User.ID, 0)
	body.ParentStyle, _ = p.GetStyleContext()
	wrapHamburger(s, body.UserPopupBody, p.Hide)

	p.SetChildren(body)
	p.Popup()
}

func wrapHamburger(s *ningen.State, body *UserPopupBody, destroy func()) {
	// body MUST starts at 3

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.Show()
	body.Attach(main, 3)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	sep.Show()
	main.Add(sep)

	menu, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	menu.Show()
	gtkutils.Margin(menu, SectionPadding)

	stack, _ := gtk.StackNew()
	stack.AddNamed(menu, "main")
	stack.AddNamed(newStatusPage(s, destroy), "status")
	stack.Show()
	main.Add(stack)

	gtkutils.InjectCSSUnsafe(stack, "", `
		stack { margin: 0; }
	`)

	statusBtn := newModelButton("Status")
	statusBtn.SetProperty("menu-name", "status")
	menu.Add(statusBtn)

	aboutBtn := newButton("About", func() {
		destroy()
		about.Spawn()
	})
	menu.Add(aboutBtn)
}

func newStatusPage(s *ningen.State, destroy func()) gtk.IWidget {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	gtkutils.Margin(box, SectionPadding)

	// Make a back button
	btn, _ := gtk.ModelButtonNew()
	btn.SetLabel("Status")
	btn.SetProperty("inverted", true)
	btn.SetProperty("menu-name", "main")
	btn.Show()
	box.Add(btn)

	statuses := [][3]string{
		{"#43B581", "Online", string(discord.OnlineStatus)},
		{"#FAA61A", "Idle", string(discord.IdleStatus)},
		{"#F04747", "Do Not Disturb", string(discord.DoNotDisturbStatus)},
		{"#747F8D", "Invisible", string(discord.InvisibleStatus)},
	}

	var p, _ = s.Presence(0, s.Ready.User.ID)

	for _, status := range statuses {
		btn := newModelButton(`<span color="` + status[0] + `">●</span> ` + status[1])
		btn.Connect("button-release-event", func() bool {
			destroy()
			return true
		})

		if p != nil && string(p.Status) == status[2] {
			s, _ := btn.GetStyleContext()
			s.SetState(gtk.STATE_FLAG_ACTIVE)
		}

		box.Add(btn)
	}

	return box
}

func newModelButton(markup string) *gtk.ModelButton {
	// Create the button
	btn, _ := gtk.ModelButtonNew()
	btn.SetLabel(markup)

	// Set the label
	c, err := btn.GetChild()
	if err != nil {
		log.Errorln("Failed to get child of ModelButton")
		return btn
	}

	l := &gtk.Label{Widget: *c}
	l.SetUseMarkup(true)
	l.SetHAlign(gtk.ALIGN_START)

	btn.ShowAll()
	return btn
}

func newButton(markup string, callback func()) *gtk.ModelButton {
	btn := newModelButton(markup)
	btn.Connect("button-release-event", func() bool {
		callback()
		return true
	})

	return btn
}
