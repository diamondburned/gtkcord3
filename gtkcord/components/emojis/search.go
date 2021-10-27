package emojis

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v3"
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
	search.ScrolledWindow = gtk.NewScrolledWindow(nil, nil)
	search.Flow = newFlowBox()

	search.ScrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	search.ScrolledWindow.SetPropagateNaturalHeight(true)
	search.ScrolledWindow.SetMinContentHeight(400)
	search.ScrolledWindow.SetMaxContentHeight(400)
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
		if strings.Contains(e.Name(), text) {
			s.Add(e)
			s.visible = append(s.visible, i)
		}
	}
}
