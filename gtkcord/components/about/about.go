package about

import (
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/gotk3/gotk3/gtk"
)

func Spawn() {
	a, _ := gtk.AboutDialogNew()
	a.SetModal(true)
	a.SetTransientFor(window.Window)

	a.SetProgramName("gtkcord3")
	a.SetAuthors([]string{"diamondburned", "Contributors"})

	a.SetCopyright("Copyright (C) 2020 diamondburned")
	a.SetLicense("GNU General Public License v3.0")
	a.SetLicenseType(gtk.LICENSE_GPL_3_0)

	a.SetWebsite("https://github.com/diamondburned/gtkcord3")
	a.SetWebsiteLabel("Source code")

	a.Run()
}
