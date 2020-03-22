package plugin

import (
	"github.com/diamondburned/gtkcord3/internal/log"
	"io/ioutil"
	"path"
	"plugin"
)

// TODO add functions in here to have plugins handle
type PluginHook struct {

}

var Plugins []PluginHook

// LoadPlugins loads shared object files that add modularity to GTKCord.
func LoadPlugins() {
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
		s, err := pl.Lookup("PluginHook")
		if err != nil {
			log.Errorf("invalid plugin : %v", err)
		}
		var hook PluginHook
		hook, ok := s.(PluginHook)
		if !ok {
			log.Errorf("invalid plugin : %v", err)
		}
		Plugins = append(Plugins, hook)
	}
}