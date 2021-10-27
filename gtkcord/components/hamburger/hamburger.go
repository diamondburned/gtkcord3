package hamburger

import (
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/components/about"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/ningen/v2"

	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

type Opts struct {
	State    *ningen.State
	LogOut   func()
	Settings func()
}

type Popover struct {
	*popup.Popover
	opts Opts
}

func BindToButton(button *gtk.MenuButton, opts Opts) {
	popup.NewDynamicPopover(button, callHamburger(opts))
}

func callHamburger(opts Opts) popup.PopoverCreator {
	me, _ := opts.State.Me()
	meID := me.ID

	return func(p *gtk.Popover) gtk.Widgetter {
		body := popup.NewStatefulPopupBody(opts.State, meID, 0)
		body.ParentStyle = p.StyleContext()
		wrapHamburger(body.UserPopupBody, p.Hide, opts)

		return body
	}
}

func wrapHamburger(body *popup.UserPopupBody, destroy func(), opts Opts) {
	// body MUST starts at 3

	main := gtk.NewBox(gtk.OrientationVertical, 0)
	main.Show()
	body.Attach(main, 3)

	sep := gtk.NewSeparator(gtk.OrientationHorizontal)
	sep.Show()
	main.Add(sep)

	menu := gtk.NewBox(gtk.OrientationVertical, 0)
	menu.Show()
	gtkutils.Margin(menu, popup.SectionPadding)

	stack := gtk.NewStack()
	stack.AddNamed(menu, "main")
	stack.AddNamed(newStatusPage(opts.State, destroy), "status")
	stack.SetTransitionDuration(150)
	stack.SetTransitionType(gtk.StackTransitionTypeSlideRight)
	stack.Show()
	main.Add(stack)

	gtkutils.InjectCSS(stack, "", `
		stack { margin: 0; }
	`)

	statusBtn := newModelButton("Status")
	statusBtn.SetObjectProperty("menu-name", "status")
	menu.Add(statusBtn)

	propBtn := newButton("Properties", func() {
		destroy()
		opts.Settings()
	})
	menu.Add(propBtn)

	logoutBtn := newButton("Log Out", func() {
		destroy()
		opts.LogOut()
	})
	menu.Add(logoutBtn)

	aboutBtn := newButton("About", func() {
		destroy()
		about.Spawn()
	})
	menu.Add(aboutBtn)

	quitBtn := newButton("Quit", func() {
		destroy()
		window.Destroy()
	})
	menu.Add(quitBtn)
}

func newStatusPage(s *ningen.State, destroy func()) gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Show()
	gtkutils.Margin(box, popup.SectionPadding)

	// Make a back button
	btn := gtk.NewModelButton()
	btn.SetLabel("Status")
	btn.SetObjectProperty("inverted", true)
	btn.SetObjectProperty("menu-name", "main")
	btn.Show()
	box.Add(btn)

	statuses := [][3]string{
		{"#43B581", "Online", string(gateway.OnlineStatus)},
		{"#FAA61A", "Idle", string(gateway.IdleStatus)},
		{"#F04747", "Do Not Disturb", string(gateway.DoNotDisturbStatus)},
		{"#747F8D", "Invisible", string(gateway.InvisibleStatus)},
	}

	me, _ := s.Me()
	p, _ := s.Presence(0, me.ID)

	for _, status := range statuses {
		btn := newModelButton(`<span color="` + status[0] + `">●</span> ` + status[1])
		btn.Connect("button-release-event", func() bool {
			destroy()
			return true
		})

		if p != nil && string(p.Status) == status[2] {
			s := btn.StyleContext()
			s.SetState(gtk.StateFlagActive)
		}

		box.Add(btn)
	}

	return box
}

func newModelButton(markup string) *gtk.ModelButton {
	return popup.NewModelButton(markup)
}

func newButton(markup string, callback func()) *gtk.ModelButton {
	return popup.NewButton(markup, callback)
}
