package settings

import "github.com/diamondburned/handy"

type PluginsPage struct {
	*handy.PreferencesPage
}

func NewPluginsPage() *PluginsPage {
	p := handy.PreferencesPageNew()
	p.SetIconName("application-x-addon-symbolic")
	p.SetTitle("Plugins")

	return &PluginsPage{
		PreferencesPage: p,
	}
}
