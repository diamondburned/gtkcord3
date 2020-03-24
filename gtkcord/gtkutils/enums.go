package gtkutils

// #cgo pkg-config: gtk+-3.0
// #include <gtk/gtk.h>
import "C"

type ScrollablePolicy int

const (
	SCROLL_MINIMUM = C.GTK_SCROLL_MINIMUM
	SCROLL_NATURAL = C.GTK_SCROLL_NATURAL
)
