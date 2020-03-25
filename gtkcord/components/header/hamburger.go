package header

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/about"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type MainHamburger struct {
	gtkutils.ExtendedWidget
	Button *gtk.MenuButton

	State *ningen.State
}

func NewMainHamburger() (*MainHamburger, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make hamburger box")
	}
	b.Show()
	b.SetSizeRequest(guild.IconSize+guild.IconPadding*2, -1)

	mb, err := gtk.MenuButtonNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create menu button")
	}
	mb.SetSensitive(true)
	mb.SetHAlign(gtk.ALIGN_CENTER)
	mb.Show()
	b.Add(mb)

	i, err := gtk.ImageNewFromIconName("open-menu", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar placeholder")
	}
	i.Show()
	mb.Add(i)

	hm := &MainHamburger{ExtendedWidget: b, Button: mb}

	p := popup.NewDynamicPopover(mb, func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		if hm.State == nil {
			return nil
		}

		body := popup.NewStatefulPopupBody(hm.State, hm.State.Ready.User.ID, 0)
		body.ParentStyle, _ = p.GetStyleContext()
		wrapHamburger(hm.State, body.UserPopupBody, p.Hide)

		return body
	})

	mb.SetPopover(p.Popover)
	mb.SetUsePopover(true)

	return hm, nil
}

func (h *MainHamburger) UseState(s *ningen.State) {
	h.State = s
}

func wrapHamburger(s *ningen.State, body *popup.UserPopupBody, destroy func()) {
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
