package gtkutils

// #cgo pkg-config: gtk+-3.0
// #include <gtk/gtk.h>
// #include <gdk/gdk.h>
import "C"

import (
	"unsafe"

	"github.com/gotk3/gotk3/gdk"
)

type ScrollablePolicy int

const (
	SCROLL_MINIMUM = C.GTK_SCROLL_MINIMUM
	SCROLL_NATURAL = C.GTK_SCROLL_NATURAL
)

func WindowSetEvents(w *gdk.Window, events gdk.EventMask) {
	native := (*C.GdkWindow)(unsafe.Pointer(w.Native()))
	C.gdk_window_set_events(native, C.GdkEventMask(events))
}
