package variables

import "github.com/diamondburned/gotk4/pkg/gtk/v3"

var (
	EmbedAvatarSize = 24
	EmbedMaxWidth   = 300
	EmbedImgHeight  = 300 // max
	EmbedMargin     = 8

	AvatarSize    = 42 // gtk.ICON_SIZE_DND
	AvatarPadding = 10

	// used as fallback, the settings one overrides this
	MaxMessageWidth = 750

	InputIconSize = gtk.IconSizeLargeToolbar
)
