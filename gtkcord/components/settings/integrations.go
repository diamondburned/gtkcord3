package settings

import (
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type Integrations struct {
	*handy.PreferencesPage `json:"-"`

	RichPresence RichPresence `json:"rich_presence"`
	Plugins      Plugins      `json:"plugins"`
}

func DefaultIntegrations() *Integrations {
	return &Integrations{
		RichPresence: RichPresence{
			MPRIS:        true,
			RichPresence: true,
		},
		Plugins: Plugins{},
	}
}

func (p *Integrations) Initialize() {
	p.PreferencesPage = handy.PreferencesPageNew()
	p.PreferencesPage.Show()
	p.PreferencesPage.SetIconName("package-x-generic-symbolic")
	p.PreferencesPage.SetTitle("Integrations")

	p.RichPresence.Initialize()
	p.PreferencesPage.Add(p.RichPresence)

	p.Plugins.Initialize()
	p.PreferencesPage.Add(p.Plugins)
}

type RichPresence struct {
	*handy.PreferencesGroup `json:"-"`

	MPRIS        bool `json:"mpris"`
	RichPresence bool `json:"rich_presence"`
}

func (p *RichPresence) Initialize() {
	p.PreferencesGroup = handy.PreferencesGroupNew()
	p.PreferencesGroup.Show()
	p.PreferencesGroup.SetTitle("Rich Presence")

	mprisS, _ := gtk.SwitchNew()
	mprisS.SetHAlign(gtk.ALIGN_END)
	bindSwitch(mprisS, &p.MPRIS)

	p.PreferencesGroup.Add(row(
		"MPRIS D-Bus",
		"Broadcast currently played songs to Discord.",
		mprisS,
	))

	rpcS, _ := gtk.SwitchNew()
	rpcS.SetHAlign(gtk.ALIGN_END)
	bindSwitch(rpcS, &p.RichPresence)

	p.PreferencesGroup.Add(row(
		"Rich Presence IPC",
		"Allow interprocess communication across games and applications to Discord.",
		rpcS,
	))
}

type Plugins struct {
	*handy.PreferencesGroup `json:"-"`

	// TODO
}

func (p *Plugins) Initialize() {
	p.PreferencesGroup = handy.PreferencesGroupNew()
	p.PreferencesGroup.SetTitle("Plugins")
	p.PreferencesGroup.ShowAll()
}
