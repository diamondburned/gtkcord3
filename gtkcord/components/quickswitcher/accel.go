package quickswitcher

import (
	"sync"

	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
)

const (
	AccelSpawnDialog = "<gtkcord>/quickswitcher.SpawnDialog"
)

var bindOnce sync.Once

func Bind(spawner Spawner) {
	bindOnce.Do(func() {
		gtk.AccelMapAddEntry(AccelSpawnDialog, gdk.KEY_K, gdk.ControlMask)
	})

	window.Window.Accel.ConnectByPath(AccelSpawnDialog, spawner.Spawn)
}
