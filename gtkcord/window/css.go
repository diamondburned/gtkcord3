package window

import (
	"os"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const CSS = `
	.status {
		padding: 3px;
		border-radius: 9999px;
	}
	.status.online {
		background-color: #43B581;
	}
	.status.busy {
		background-color: #F04747;
	}
	.status.idle {
		background-color: #FAA61A;
	}
	.status.offline {
		background-color: #747F8D;
	}
	.status.unknown {
		background-color: #FFFFFF;
	}

	@define-color color_pinged rgb(240, 71, 71);

	headerbar { padding: 0; }
	headerbar button { box-shadow: none; }
	textview, textview > text { background-color: transparent; }

	.guilds, .channels {
		background-color: @theme_bg_color;
	}
	.messages {
		background-color: @theme_base_color;
	}

	.guild-folder, .guild {
		padding-left: 0;
		padding-right: 0;
	}
	.guild-folder.unread {
		background-color: alpha(@theme_selected_bg_color, 0.15);
	}
	.guild-folder.pinged {
		background-color: alpha(@color_pinged, 0.15);
	}
	.guild-folder:selected {
		border-top: 5px solid alpha(@theme_selected_bg_color, 0.5);
	}
	.guild-folder:selected list {
		border-bottom: 5px solid alpha(@theme_selected_bg_color, 0.5);
	}
	.guild:selected {
		background-color: alpha(@theme_selected_bg_color, 0.5);
	}

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
	
	.guild > image {
	    box-shadow: 0px 0px 4px -1px rgba(0,0,0,0.5);
	    border-radius: 50%;
	}
	.guild.unread > image {
		border: 2px solid @theme_fg_color;
		padding: 2px;
	}
	.guild.pinged > image {
		border: 2px solid rgb(240, 71, 71);
		padding: 2px;
	}

	.channel {
		opacity: 0.5;
		color: white;
	}
	.channel.muted {
		opacity: 0.25;
	}
	.channel.unread {
		opacity: 1;
	}
	.channel.pinged {
		opacity: 1;
		color: @color_pinged;
		background-color: alpha(@color_pinged, 0.15);
	}

	.dmchannel.pinged {
		background-color: alpha(@color_pinged, 0.15);
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
