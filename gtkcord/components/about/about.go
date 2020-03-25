package about

import (
	"log"

	"github.com/diamondburned/gtkcord3/gtkcord/components/logo"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/gotk3/gotk3/gtk"
)

func Spawn() {
	a, _ := gtk.AboutDialogNew()

	logo, err := logo.Pixbuf(64)
	if err != nil {
		log.Panicln("Failed to load logo:", err)
	}
	a.SetLogo(logo)

	a.SetProgramName("gtkcord3")
	a.SetAuthors([]string{
		`diamondburned: "Astolfo is cute."`,
		"GitHub Contributors",
	})

	a.SetCopyright("Copyright (C) 2020 diamondburned")
	a.SetLicense("GNU General Public License v3.0")
	a.SetLicenseType(gtk.LICENSE_GPL_3_0)

	a.SetWebsite("https://github.com/diamondburned/gtkcord3")
	a.SetWebsiteLabel("Source code")

	// SWITCH!
	d := gtkutils.HandyDialog(a, window.Window)
	d.Run()
	d.GrabFocus()
}
