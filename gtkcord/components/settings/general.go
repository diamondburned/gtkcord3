package settings

import "github.com/diamondburned/handy"

type GeneralPage struct {
	*handy.PreferencesPage
}

func NewGeneralPage() *GeneralPage {
	p := handy.PreferencesPageNew()
	p.SetIconName("preferences-system-symbolic")
	p.SetTitle("General")

	return &GeneralPage{
		PreferencesPage: p,
	}
}
