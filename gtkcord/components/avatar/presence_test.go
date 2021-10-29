package avatar_test

import (
	"testing"

	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/avatar"
	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
)

const url = "https://cdn.discordapp.com/avatars/170132746042081280/0bd1f9cedae3025f239d7f628eb4f992.png?size=128"

func TestGtk(t *testing.T) {
	gtk.Init()

	win := gtk.NewWindow(gtk.WindowToplevel)
	win.SetTitle("Simple Example")
	win.Connect("destroy", func() { gtk.MainQuit() })

	box := gtk.NewBox(gtk.OrientationHorizontal, 8)
	box.SetVExpand(true)
	box.SetHExpand(true)
	box.SetVAlign(gtk.AlignCenter)
	box.SetHAlign(gtk.AlignCenter)
	gtkutils.Margin(box, 8)

	sizes := []int{12, 16, 24, 32, 48, 64}
	statuses := []gateway.Status{
		gateway.OnlineStatus,
		gateway.DoNotDisturbStatus,
		gateway.IdleStatus,
		gateway.InvisibleStatus,
		gateway.OfflineStatus,
		gateway.OnlineStatus,
	}

	for i, size := range sizes {
		avy := avatar.NewWithStatus(size)
		avy.SetInitials("M")
		avy.SetURL(url)
		avy.SetStatus(statuses[i])
		box.Add(avy)
	}

	win.Add(box)
	win.ShowAll()

	gtk.Main()
}
