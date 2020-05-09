package quickswitcher

import (
	"sync"

	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const (
	AccelSpawnDialog = "<gtkcord>/quickswitcher.SpawnDialog"
)

var bindOnce sync.Once

func Bind(spawner Spawner) {
	bindOnce.Do(func() {
		gtk.AccelMapAddEntry(AccelSpawnDialog, gdk.KEY_K, gdk.CONTROL_MASK)
	})

	window.Window.Accel.ConnectByPath(AccelSpawnDialog, spawner.Spawn)
}
