package plugin

import (
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/gtkcord3/gtkcord/components/window"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"io/ioutil"
	"os"
	"path"
	"plugin"
)

// TODO add functions in here to have plugins handle
// PluginHook is the structure for the plugins event handlers
type PluginHook interface {
	typingStart(t *gateway.TypingStartEvent)
	setWindow(window *gtk.Window)
}

var Plugins []PluginHook

// LoadPlugins loads shared object files that add modularity to GTKCord.
func LoadPlugins() {
	if _, err := os.Stat("plugins"); os.IsNotExist(err) {
		_ = os.Mkdir("plugins", 0777)
	}
	files, err := ioutil.ReadDir("plugins")
	if err != nil {
		log.Panicln("Error loading plugins : plugins folder not available!")
	}
	for _, f := range files {
		if f.IsDir() {
			continue // skip because it's a folder
		}
		pl, err := plugin.Open(path.Join("plugins", f.Name()))
		if err != nil {
			log.Errorf("error loading plugin : %v", err)
			continue // we dont want the whole app dying because of a shitty plugin
		}
		s, err := pl.Lookup("Hook")
		if err != nil {
			log.Errorf("invalid plugin : %v", err)
		}
		var hook PluginHook
		hook, ok := s.(PluginHook)
		if !ok {
			log.Errorf("invalid plugin : %v", err)
		}
		hook.setWindow(window.Window.Window)
		Plugins = append(Plugins, hook)
	}
}