package gdbus

// #cgo pkg-config: glib-2.0 gio-2.0
// #include <glib-2.0/glib.h>
// #include <gio/gio.h>
import "C"

import (
	"sync"
	"unsafe"

	"github.com/gotk3/gotk3/glib"
	"github.com/pkg/errors"
)

type Notifier struct {
	*Connection

	actionMu sync.Mutex
	actions  map[uint32][]*Action
}

func NewNotifier(c *Connection) *Notifier {
	n := &Notifier{
		Connection: c,
		actions:    map[uint32][]*Action{},
	}

	c.SignalSubscribe(
		"", "org.freedesktop.Notifications",
		"", "/org/freedesktop/Notifications", "",
		DBUS_SIGNAL_FLAGS_NONE,
		func(_ *Connection, _, _, _, signal string, _params *glib.Variant) {
			if signal != "ActionInvoked" && signal != "NotificationClosed" {
				return
			}

			var params = VariantTuple(_params, 2)
			var id, _ = params[0].GetUint()

			switch signal {
			case "ActionInvoked":
				n.onAction(uint32(id), params[1].GetString())
			case "NotificationClosed":
				reason, _ := params[1].GetUint()
				n.onClose(uint32(id), uint32(reason))
			}
		},
	)

	return n
}

func (n *Notifier) onClose(id, reason uint32) {
	n.actionMu.Lock()
	defer n.actionMu.Unlock()

	delete(n.actions, id)
}

func (n *Notifier) onAction(id uint32, actionKey string) {
	n.actionMu.Lock()
	defer n.actionMu.Unlock()

	actions, ok := n.actions[id]
	if !ok {
		return
	}

	for _, action := range actions {
		if action.ID == actionKey {
			go action.Callback()
			break
		}
	}
}

type Action struct {
	ID       string
	Label    string
	Callback func() // called in goroutine
}

type Notification struct {
	AppName string
	AppIcon string
	Title   string
	Message string
	Actions []*Action
	Expiry  int32
}

func (c *Notifier) Notify(n Notification) (uint32, error) {
	args := make([]*C.GVariant, 8) // num of structs
	args[0] = C.g_variant_new_take_string(C.CString(n.AppName))
	args[1] = C.g_variant_new_uint32(C.guint32(0))
	args[2] = C.g_variant_new_take_string(C.CString(n.AppIcon))
	args[3] = C.g_variant_new_take_string(C.CString(n.Title))
	args[4] = C.g_variant_new_take_string(C.CString(n.Message))

	var firstAction **C.GVariant

	if len(n.Actions) > 0 {
		var actions = make([]*C.GVariant, len(n.Actions)*2)

		for i := 0; i < len(actions); i += 2 {
			action := n.Actions[i/2]

			k := C.g_variant_new_take_string(C.CString(action.ID))
			v := C.g_variant_new_take_string(C.CString(action.Label))

			actions[i], actions[i+1] = k, v
		}

		firstAction = &actions[0]
	}

	args[5] = C.g_variant_new_array(C.G_VARIANT_TYPE_STRING, firstAction, C.gsize(len(n.Actions)*2))

	dict := C.g_variant_dict_new(nil)
	defer C.g_variant_dict_unref(dict)
	args[6] = C.g_variant_dict_end(dict)

	args[7] = C.g_variant_new_int32(C.gint32(n.Expiry))

	var gerr *C.GError

	v := C.g_dbus_connection_call_sync(
		c.Native,
		C.CString("org.freedesktop.Notifications"),
		C.CString("/org/freedesktop/Notifications"),
		C.CString("org.freedesktop.Notifications"),
		C.CString("Notify"),
		C.g_variant_new_tuple(&args[0], C.gsize(8)),
		C.G_VARIANT_TYPE_ANY,
		C.G_DBUS_CALL_FLAGS_NONE,
		C.gint(-1),
		nil,
		&gerr,
	)

	if gerr != nil {
		return 0, errors.New(errorMessage(gerr))
	}

	child := C.g_variant_get_child_value(v, 0)

	u, err := glib.TakeVariant(unsafe.Pointer(child)).GetUint()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get ID")
	}

	id := uint32(u)

	c.actionMu.Lock()
	c.actions[id] = n.Actions
	c.actionMu.Unlock()

	return id, nil
}
