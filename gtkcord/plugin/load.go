package plugin

import (
	"path/filepath"
	"plugin"

	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/config"
	"github.com/pkg/errors"
)

// TODO add functions in here to have plugins handle

// Hook is the structure for the plugins event handlers
// type Hook interface {
// 	Init(a *gtkcord.Application)
// 	OnReady()
// }

// StartPlugins loads shared object files that add modularity to gtkcord.
func StartPlugins(a *gtkcord.Application) error {
	plugins, path, err := config.MustRead("plugins")
	if err != nil {
		return errors.Wrap(err, "Failed to read plugins")
	}

	for _, f := range plugins {
		if f.IsDir() {
			continue // skip because it's a folder
		}

		p, err := plugin.Open(filepath.Join(path, f.Name()))
		if err != nil {
			return errors.Wrap(err, "Failed to open plugin "+f.Name())
		}

		if err := startPlugin(p, a); err != nil {
			return errors.Wrap(err, "Failed to load plugin "+f.Name())
		}
	}

	return nil
}

func startPlugin(p *plugin.Plugin, a *gtkcord.Application) error {
	s, err := p.Lookup("Ready")
	if err != nil {
		return errors.Wrap(err, "Failed to lookup function Ready")
	}

	rd, ok := s.(func(*gtkcord.Application))
	if !ok {
		return errors.Wrap(err, "Plugin has invalid Ready function signature")
	}

	rd(a)
	return nil
}
