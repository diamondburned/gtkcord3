package hamburger

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/about"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/gotk3/gotk3/gtk"
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
	semaphore.IdleMust(func() {
		BindToButtonUnsafe(button, opts)
	})
}

func BindToButtonUnsafe(button *gtk.MenuButton, opts Opts) {
	popup.NewDynamicPopover(button, callHamburger(opts))
}

func callHamburger(opts Opts) popup.PopoverCreator {
	return func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		body := popup.NewStatefulPopupBody(opts.State, opts.State.Ready.User.ID, 0)
		body.ParentStyle, _ = p.GetStyleContext()
		wrapHamburger(body.UserPopupBody, p.Hide, opts)

		return body
	}
}

func wrapHamburger(body *popup.UserPopupBody, destroy func(), opts Opts) {
	// body MUST starts at 3

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.Show()
	body.Attach(main, 3)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	sep.Show()
	main.Add(sep)

	menu, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	menu.Show()
	gtkutils.Margin(menu, popup.SectionPadding)

	stack, _ := gtk.StackNew()
	stack.AddNamed(menu, "main")
	stack.AddNamed(newStatusPage(opts.State, destroy), "status")
	stack.Show()
	main.Add(stack)

	gtkutils.InjectCSSUnsafe(stack, "", `
		stack { margin: 0; }
	`)

	statusBtn := newModelButton("Status")
	statusBtn.SetProperty("menu-name", "status")
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

func newStatusPage(s *ningen.State, destroy func()) gtk.IWidget {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	gtkutils.Margin(box, popup.SectionPadding)

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
	return popup.NewModelButton(markup)
}

func newButton(markup string, callback func()) *gtk.ModelButton {
	return popup.NewButton(markup, callback)
}
