package window

import (
	"os"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const CSS = `
	headerbar { padding: 0; }
	headerbar button { box-shadow: none; }
	textview, textview > text { background-color: transparent; }
	
	.message:not(.condensed), .message-input {
		border-top: 1px solid rgba(0, 0, 0, 0.12);
	}

	.user-info, .user-info > box > *:nth-child(n+3) {
		background-color: @theme_base_color;
	}
	.user-info.spotify {
		background-color: #1db954;
	}

	.message-input {
		background-image: linear-gradient(transparent, rgba(10, 10, 10, 0.3));
		transition-property: background-image;
		transition: 75ms background-image linear;
	}
	.message-input.editing {
		background-image: linear-gradient(transparent, rgba(114, 137, 218, 0.3));
	}

	.message-input .completer {
		background-color: transparent;
		border-bottom: 1px solid rgba(0, 0, 0, 0.12);
	}

	.message-input button {
		background: none;
		box-shadow: none;
		border: none;
		opacity: 0.65;
	}
	.message-input button:hover {
		opacity: 1;
	}
	
	.guild image {
	    box-shadow: 0px 0px 4px -1px rgba(0,0,0,0.5);
	    border-radius: 50%;
		background-color: grey;
	}

	.message image:not(.avatar) {
		border: 1px solid rgba(0, 0, 0, 0.25);
	}
`

var CustomCSS = os.Getenv("GTKCORD_CUSTOM_CSS")

// I don't like this:
// list row:selected { box-shadow: inset 2px 0 0 0 white; }

func loadCSS(s *gdk.Screen) error {
	css, err := gtk.CssProviderNew()
	if err != nil {
		return errors.Wrap(err, "Failed to make a CSS provider")
	}

	if err := css.LoadFromData(CSS + CustomCSS); err != nil {
		return errors.Wrap(err, "Failed to parse CSS")
	}

	Window.CSS = css

	gtk.AddProviderForScreen(s, css,
		uint(gtk.STYLE_PROVIDER_PRIORITY_USER))

	return nil
}
