package plugin

import (
	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/internal/log"
	"io/ioutil"
	"os"
	"path"
	"plugin"
)

// TODO add functions in here to have plugins handle

// Hook is the structure for the plugins event handlers
type Hook interface {
	Init(gtkcord *gtkcord.Application)
}

var Plugins []Hook

// LoadPlugins loads shared object files that add modularity to GTKCord.
func LoadPlugins(a *gtkcord.Application) {
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
		var hook Hook
		hook, ok := s.(Hook)
		if !ok {
			log.Errorf("invalid plugin : %v", err)
		}
		hook.Init(a)
		Plugins = append(Plugins, hook)
	}
}