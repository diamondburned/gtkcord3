package gtkcord

import (
	"os"
	"path/filepath"
	"plugin"

	"github.com/diamondburned/gtkcord3/gtkcord/config"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/pkg/errors"
)

// TODO add functions in here to have plugins handle

type Plugin struct {
	// Plugin constants/functions
	Name    string // optional
	Author  string // optional
	OnReady func(a *Application)

	// Auto-detected
	Path string
	Err  error
}

func (a *Application) readyPlugins() {
	for _, plugin := range a.Plugins {
		if plugin.Err != nil {
			continue
		}
		plugin.OnReady(a)
	}
}

func (a *Application) removePlugin(path string) bool {
	for i, plugin := range a.Plugins {
		if plugin.Path == path {
			// Remove from list
			a.Plugins = append(a.Plugins[:i], a.Plugins[i+1:]...)

			// Remove from the filesystem:
			if err := os.Remove(path); err != nil {
				log.Errorln("Failed to remove", path+":", err)
			}

			return true
		}
	}

	return false
}

func loadPlugins() ([]*Plugin, error) {
	files, path, err := config.MustRead("plugins")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read plugins")
	}

	var plugins = make([]*Plugin, 0, len(files))

	for _, f := range files {
		if f.IsDir() {
			continue // skip because it's a folder
		}

		p := loadPlugin(filepath.Join(path, f.Name()))
		plugins = append(plugins, p)
	}

	return plugins, nil
}

func loadPlugin(path string) *Plugin {
	p, err := plugin.Open(path)
	if err != nil {
		return newErrPlugin(path, err)
	}

	s, err := p.Lookup("Ready")
	if err != nil {
		return newErrPlugin(path, err)
	}

	rd, ok := s.(func(*Application))
	if !ok {
		return newErrPlugin(path, errors.New("Ready() is not func(*gtkcord.Application)"))
	}

	plugin := &Plugin{
		Path:    path,
		OnReady: rd,
		Name:    filepath.Base(path),
	}

	// Everything beyond is optional
	if v, err := p.Lookup("Name"); err == nil {
		if name, ok := v.(string); ok {
			plugin.Name = name
		}
	}

	if v, err := p.Lookup("Author"); err == nil {
		if auth, ok := v.(string); ok {
			plugin.Author = auth
		}
	}

	return plugin
}

func newErrPlugin(path string, err error) *Plugin {
	return &Plugin{
		Name: filepath.Base(path),
		Path: path,
		Err:  err,
	}
}
