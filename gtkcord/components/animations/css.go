package animations

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const CSS = `
	@keyframes breathing {
		0% {   opacity: 0.66; }
		100% { opacity: 0.12; }
	}
	.anim-breathing label {
		animation: breathing 800ms infinite alternate;
	}
	.anim-breathing label:nth-child(1) {
		animation-delay: 000ms;
	}
	.anim-breathing label:nth-child(2) {
		animation-delay: 150ms;
	}
	.anim-breathing label:nth-child(3) {
		animation-delay: 300ms;
	}
`

func LoadCSS(s *gdk.Screen) error {
	css, err := gtk.CssProviderNew()
	if err != nil {
		return errors.Wrap(err, "Failed to make a CSS provider")
	}

	if err := css.LoadFromData(CSS); err != nil {
		return errors.Wrap(err, "Failed to parse CSS")
	}

	gtk.AddProviderForScreen(s, css, uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION))
	return nil
}
