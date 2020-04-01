package settings

import "github.com/diamondburned/handy"

type IntegrationsPage struct {
	*handy.PreferencesPage
}

func NewIntegrationsPage() *IntegrationsPage {
	p := handy.PreferencesPageNew()
	p.SetIconName("package-x-generic-symbolic")
	p.SetTitle("Integrations")

	return &IntegrationsPage{
		PreferencesPage: p,
	}
}

func integrationsRichPresence() *handy.PreferencesGroup {
	g := handy.PreferencesGroupNew()
	g.SetTitle("Rich Presence")

	mpris := handy.PreferencesRowNew()
	mpris.SetTitle("MPRIS D-Bus")

	rpc := handy.PreferencesRowNew()
	rpc.SetTitle("Rich Presence IPC")

	g.Add(mpris)
	g.Add(rpc)

	return g
}
