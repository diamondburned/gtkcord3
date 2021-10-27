package window

import "github.com/diamondburned/gotk4/pkg/gtk/v3"

func overrideSettings(s *gtk.Settings) {
	s.SetObjectProperty("gtk-dialogs-use-header", true)
}

func PreferDarkTheme(value bool) {
	Window.Settings.SetObjectProperty("gtk-application-prefer-dark-theme", value)
}
