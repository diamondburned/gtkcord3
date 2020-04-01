package settings

import (
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type Window struct {
	*handy.PreferencesWindow
	Pages []gtk.IWidget
}

func NewWindow() *Window {
	w := handy.PreferencesWindowNew()
	return &Window{
		PreferencesWindow: w,
	}
}
