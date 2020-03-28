package gdbus

// #cgo pkg-config: glib-2.0 gio-2.0
// #include <glib-2.0/glib.h>
// #include <gio/gio.h>
// #include "error.h"
import "C"

import (
	"errors"
	"math/rand"
	"sync"
	"unsafe"

	"github.com/gotk3/gotk3/glib"
)

type BusType int

const (
	BUS_TYPE_STARTER BusType = C.G_BUS_TYPE_STARTER
	BUS_TYPE_NONE    BusType = C.G_BUS_TYPE_NONE
	BUS_TYPE_SYSTEM  BusType = C.G_BUS_TYPE_SYSTEM
	BUS_TYPE_SESSION BusType = C.G_BUS_TYPE_SESSION
)

type CallFlags int

const (
	DBUS_CALL_FLAGS_NONE                            CallFlags = C.G_DBUS_CALL_FLAGS_NONE
	DBUS_CALL_FLAGS_NO_AUTO_START                   CallFlags = C.G_DBUS_CALL_FLAGS_NO_AUTO_START
	DBUS_CALL_FLAGS_ALLOW_INTERACTIVE_AUTHORIZATION CallFlags = C.G_DBUS_CALL_FLAGS_ALLOW_INTERACTIVE_AUTHORIZATION
)

type Connection struct {
	Native *C.GDBusConnection
}

func GetSessionBusSync() (*Connection, error) {
	var err *C.GError
	v := C.g_bus_get_sync(C.GBusType(BUS_TYPE_SESSION), nil, &err)
	if err != nil {
		return nil, errors.New(C.GoString(C.error_message(err)))
	}

	return &Connection{Native: v}, nil
}

func FromApplication(app *glib.Application) *Connection {
	v := C.g_application_get_dbus_connection((*C.GApplication)(unsafe.Pointer(app.Native())))
	if v == nil {
		return nil
	}
	return &Connection{Native: v}
}

func (c *Connection) CallSync(
	busName,
	objectPath,
	interfaceName,
	methodName string,
	parameters glib.IVariant,
	replyType *glib.VariantType,
	callFlags CallFlags,
	timeoutMsec int,
) (*glib.Variant, error) {

	var err *C.GError

	v := C.g_dbus_connection_call_sync(
		c.Native,
		C.CString(busName),
		C.CString(objectPath),
		C.CString(interfaceName),
		C.CString(methodName),
		(*C.GVariant)(parameters.ToGVariant()),
		(*C.GVariantType)(replyType.GVariantType),
		C.GDBusCallFlags(callFlags),
		C.gint(timeoutMsec),
		nil,
		&err,
	)

	if err != nil {
		return nil, errors.New(C.GoString(C.error_message(err)))
	}

	return glib.TakeVariant(unsafe.Pointer(v)), nil
}

type Notification struct {
	AppName   string
	ReplaceID uint32
	AppIcon   string
	Title     string
	Message   string
	Actions   [][2]string
	Expiry    int32
}

func (c *Connection) Notify(n Notification) error {
	args := make([]*C.GVariant, 8) // num of structs
	args[0] = C.g_variant_new_take_string(C.CString(n.AppName))
	args[1] = C.g_variant_new_uint32(C.guint32(n.ReplaceID))
	args[2] = C.g_variant_new_take_string(C.CString(n.AppIcon))
	args[3] = C.g_variant_new_take_string(C.CString(n.Title))
	args[4] = C.g_variant_new_take_string(C.CString(n.Message))

	var firstAction **C.GVariant

	if len(n.Actions) > 0 {
		var actions = make([]*C.GVariant, len(n.Actions)*2)

		for i := 0; i < len(actions); i += 2 {
			action := n.Actions[i/2]

			k := C.g_variant_new_take_string(C.CString(action[0]))
			v := C.g_variant_new_take_string(C.CString(action[1]))

			actions[i], actions[i+1] = k, v
		}

		firstAction = &actions[0]
	}

	args[5] = C.g_variant_new_array(C.G_VARIANT_TYPE_STRING, firstAction, C.gsize(len(n.Actions)*2))

	dict := C.g_variant_dict_new(nil)
	defer C.g_variant_dict_unref(dict)
	args[6] = C.g_variant_dict_end(dict)

	args[7] = C.g_variant_new_int32(C.gint32(n.Expiry))

	var err *C.GError

	C.g_dbus_connection_call_sync(
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
		&err,
	)

	if err != nil {
		return errors.New(C.GoString(C.error_message(err)))
	}

	return nil
}

var gUUID = map[string]uint32{}
var gMut = sync.Mutex{}

func GetUUID(key string) uint32 {
	const offset = 100

	gMut.Lock()
	defer gMut.Unlock()

	if u, ok := gUUID[key]; ok {
		return u
	}

	id := rand.Uint32() + offset
	gUUID[key] = id

	return id
}
