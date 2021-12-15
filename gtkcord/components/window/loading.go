package window

import (
	"github.com/diamondburned/gotk4-handy/pkg/handy"
	"github.com/diamondburned/gtkcord3/gtkcord/components/animations"
)

const LoadingTitle = "Connecting to Discord — gtkcord3"

// NowLoading fades the internal stack view to show a spinning circle.
func NowLoading() {
	// Use a spinner:
	s := animations.NewSpinner(75)

	// Use a custom header instead of the actual Header:
	h := handy.NewHeaderBar()
	h.SetTitle(LoadingTitle)
	h.SetShowCloseButton(true)
	h.ShowAll()

	p := SwitchToPage("__loading__")
	p.SetHeader(h)
	p.SetChild(s)
}
