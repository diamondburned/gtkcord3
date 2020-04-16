package emojis

import (
	"strings"

	"github.com/gotk3/gotk3/gtk"
)

type SearchPage struct {
	*gtk.ScrolledWindow
	Flow *gtk.FlowBox

	// all emojis, basically copy of mainpage's emojis
	emojis  map[string]*gtk.Button // button.Name() -> emoji.String()
	visible []string
}

func newSearchPage(p *Picker) SearchPage {
	search := SearchPage{}
	search.ScrolledWindow, _ = gtk.ScrolledWindowNew(nil, nil)
	search.Flow = newFlowBox()

	search.ScrolledWindow.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	search.ScrolledWindow.SetProperty("propagate-natural-height", true)
	search.ScrolledWindow.SetProperty("min-content-height", 400)
	search.ScrolledWindow.SetProperty("max-content-height", 400)
	search.ScrolledWindow.Add(search.Flow)

	return search
}

func (s *SearchPage) search(text string) {
	// Remove old entries.
	for _, v := range s.visible {
		s.Remove(s.emojis[v])
	}
	s.visible = s.visible[:0]

	for i, e := range s.emojis {
		if n, _ := e.GetName(); strings.Contains(n, text) {
			s.Add(e)
			s.visible = append(s.visible, i)
		}
	}
}
