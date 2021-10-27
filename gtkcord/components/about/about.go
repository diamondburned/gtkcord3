package about

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
)

// Changed on build.
var Version = "(dev)"

func Spawn() {
	a := gtk.NewAboutDialog()
	a.SetLogo(logo.Pixbuf(64))

	a.SetProgramName("gtkcord3")
	a.SetAuthors([]string{
		`diamondburned: "Astolfo is cute."`,
		"GitHub Contributors",
	})
	a.SetVersion("v" + Version)

	a.SetCopyright("Copyright (C) 2020 diamondburned")
	a.SetLicense("GNU General Public License v3.0")
	a.SetLicenseType(gtk.LicenseGPL30)

	a.SetWebsite("https://github.com/diamondburned/gtkcord3")
	a.SetWebsiteLabel("Source code")

	// SWITCH!
	a.ShowAll()
}
