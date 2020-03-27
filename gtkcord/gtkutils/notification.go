package gtkutils

// #cgo pkg-config: glib-2.0
// #include <gio/gio.h>
// #include <glib.h>
import "C"

import (
	"unsafe"

	"github.com/gotk3/gotk3/glib"
)

func NAddButtonWithTargetValue(n *glib.Notification, label, action string, target *glib.Variant) {
	C.g_notification_add_button_with_target_value(
		(*C.GNotification)(unsafe.Pointer(n.GObject)),
		C.CString(label),
		C.CString(action),
		(*C.GVariant)(target.ToGVariant()),
	)
}

func NSetDefaultActionAndTargetValue(n *glib.Notification, action string, target *glib.Variant) {
	C.g_notification_set_default_action_and_target_value(
		(*C.GNotification)(unsafe.Pointer(n.GObject)),
		C.CString(action),
		(*C.GVariant)(target.ToGVariant()),
	)
}
