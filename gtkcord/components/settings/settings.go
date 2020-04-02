package settings

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

const Filename = "settings.json"

type Window struct {
	*handy.PreferencesWindow
	Settings Settings

	Save func(s Settings)
}

type Settings struct {
	General      General      `json:"general"`
	Integrations Integrations `json:"integrations"`
}

func Default() *Settings {
	return &Settings{
		General:      *DefaultGeneral(),
		Integrations: *DefaultIntegrations(),
	}
}

func (s *Settings) Initialize() {
	s.General.Initialize()
	s.Integrations.Initialize()
}

func (s *Settings) addTo(w gtkutils.Container) {
	w.Add(s.General)
	w.Add(s.Integrations)
}

func NewWindow(s *Settings) *Window {
	w := &Window{
		Settings: *s,
	}

	w.PreferencesWindow = handy.PreferencesWindowNew()
	w.PreferencesWindow.SetTitle("Preferences")
	w.PreferencesWindow.SetModal(true)
	w.PreferencesWindow.SetTransientFor(window.Window)
	w.PreferencesWindow.SetDefaultSize(500, 400)

	// Save config on close:
	w.PreferencesWindow.Connect("destroy", func() {
		log.Infoln("Saving config")
		w.save()
	})

	w.Settings.Initialize()
	w.Settings.addTo(w.PreferencesWindow)

	return w
}

func (w *Window) save() {
	if w.Save == nil {
		log.Errorln("Failed to save: w.Save == nil")
		return
	}
	w.Save(w.Settings)
}

func row(title, subtitle string, w gtk.IWidget) *handy.ActionRow {
	r := handy.ActionRowNew()
	r.SetTitle(title)
	r.SetSubtitle(subtitle)
	r.SetActivatableWidget(w)
	r.Add(w)
	r.ShowAll()

	return r
}

func bindSwitch(s *gtk.Switch, b *bool) {
	s.SetActive(*b)
	s.Connect("state-set", func(_ *gtk.Switch, state bool) {
		*b = state
	})
}

func bindFileChooser(fsb *gtk.FileChooserButton, s *string) {
	fsb.SetFilename(*s)
	fsb.Connect("file-set", func() {
		*s = fsb.GetFilename()
	})
}

func bindEntry(e *gtk.Entry, s *string) {
	e.SetText(*s)
	e.Connect("changed", func() {
		t, err := e.GetText()
		if err != nil {
			log.Errorln("Failed to get entry text:", err)
			return
		}

		*s = t
	})
}
