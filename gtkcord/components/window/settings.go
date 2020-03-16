package window

import "github.com/gotk3/gotk3/gtk"

func overrideSettings(s *gtk.Settings) {
	s.SetProperty("gtk-dialogs-use-header", true)
}

func PreferDarkTheme(value bool) {
	Window.Settings.SetProperty("gtk-application-prefer-dark-theme", value)
}
