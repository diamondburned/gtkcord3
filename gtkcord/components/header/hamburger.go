package header

import (
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/about"
	"github.com/diamondburned/gtkcord3/gtkcord/components/guild"
	"github.com/diamondburned/gtkcord3/gtkcord/components/popup"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Hamburger struct {
	gtkutils.ExtendedWidget
	Popover *popup.Popover

	// About
}

const HeaderWidth = 240

func NewHeaderMenu(s *ningen.State) (*Hamburger, error) {
	b, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make hamburger box")
	}
	b.SetSizeRequest(guild.IconSize+guild.IconPadding*2, -1)

	mb, err := gtk.MenuButtonNew()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create menu button")
	}
	mb.SetHAlign(gtk.ALIGN_CENTER)
	b.Add(mb)

	i, err := gtk.ImageNewFromIconName("open-menu", gtk.ICON_SIZE_LARGE_TOOLBAR)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create avatar placeholder")
	}
	mb.Add(i)

	// Header box
	p := popup.NewDynamicPopover(mb, func(p *gtk.Popover) gtkutils.WidgetDestroyer {
		body := popup.NewStatefulPopupBody(s, s.Ready.User.ID, 0)
		body.ParentStyle, _ = p.GetStyleContext()
		return wrapHamburger(s, body.UserPopupBody)
	})

	mb.SetPopover(p.Popover)
	mb.SetUsePopover(true)

	hm := &Hamburger{
		ExtendedWidget: b,
		Popover:        p,
	}
	hm.ShowAll()

	return hm, nil
}

func wrapHamburger(s *ningen.State, body *popup.UserPopupBody) gtkutils.WidgetDestroyer {
	// body MUST starts at 3

	stack, _ := gtk.StackNew()
	stack.AddNamed(body, "main")
	stack.AddNamed(newStatusPage(s), "status")
	stack.Show()

	gtkutils.InjectCSSUnsafe(stack, "", `
		stack { margin: 0; }
	`)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	body.Attach(main, 3)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	main.Add(sep)

	menu, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	menu.Show()

	gtkutils.Margin(menu, popup.SectionPadding)
	menu.SetMarginTop(0)
	main.Add(menu)

	statusBtn := newModelButton("Status")
	statusBtn.SetProperty("menu-name", "status")
	menu.Add(statusBtn)

	aboutBtn := newModelButton("About")
	aboutBtn.Connect("activate", about.Spawn)
	menu.Add(aboutBtn)

	return stack
}

func newStatusPage(s *ningen.State) gtk.IWidget {
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
		btn.Connect("activate", func() {
			log.Println("Changing status to", status[2])
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
