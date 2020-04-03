package gtkcord

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/settings"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/config"
	"github.com/diamondburned/gtkcord3/gtkcord/md"
	"github.com/diamondburned/gtkcord3/internal/log"
)

const SettingsFile = "settings.json"

func loadSettings() *settings.Settings {
	var s = settings.Default()
	if err := config.UnmarshalFromFile(SettingsFile, s); err != nil {
		log.Errorln("Failed to load settings, using default. Error:", err)
	}

	return s
}

func (a *Application) spawnSettings() {
	s := loadSettings()
	w := settings.NewWindow(s)
	w.Save = func(s settings.Settings) {
		if err := config.MarshalToFile(SettingsFile, s); err != nil {
			log.Errorln("Failed to save config:", err)
		}

		// Apply anyway
		a.applySettings(&s)
	}
	w.Show()
}

func (a *Application) applySettings(s *settings.Settings) {
	// Customizations
	window.FileCSS = s.General.Customization.CSSFile
	window.ReloadCSS()
	md.ChangeStyle(s.General.Customization.HighlightScheme)

	// Integrations
	a.MPRIS.SetEnabled(s.Integrations.RichPresence.MPRIS)
}
