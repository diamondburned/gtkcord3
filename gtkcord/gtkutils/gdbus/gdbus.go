package gdbus

/*
#cgo pkg-config: glib-2.0 gio-2.0
#include <glib-2.0/glib.h>
#include <gio/gio.h>

extern void gdbusSignalCallback(
	GDBusConnection *conn,
	gchar *sender_name,
	gchar *object_path,
	gchar *interface_name,
	gchar *signal_name,
	GVariant *parameters,
	gpointer key
);
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gotk3/gotk3/glib"
	"github.com/pkg/errors"
)

//export gdbusSignalCallback
func gdbusSignalCallback(
	conn *C.GDBusConnection,
	senderName, objectPath, interfaceName, signalName *C.gchar,
	parameters *C.GVariant,
	key C.gpointer,
) {

	cb := cbGet(key)
	if cb == nil {
		return
	}

	cb.fn.(SignalCallback)(
		cb.receiver.(*Connection),
		C.GoString(senderName),
		C.GoString(objectPath),
		C.GoString(interfaceName),
		C.GoString(signalName),
		glib.TakeVariant(unsafe.Pointer(parameters)),
	)
}

type SignalCallback func(
	conn *Connection,
	senderName, objectPath, interfaceName, signalName string,
	parameters *glib.Variant,
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

type SignalFlags int

const (
	DBUS_SIGNAL_FLAGS_NONE                 SignalFlags = C.G_DBUS_SIGNAL_FLAGS_NONE
	DBUS_SIGNAL_FLAGS_NO_MATCH_RULE        SignalFlags = C.G_DBUS_SIGNAL_FLAGS_NO_MATCH_RULE
	DBUS_SIGNAL_FLAGS_MATCH_ARG0_NAMESPACE SignalFlags = C.G_DBUS_SIGNAL_FLAGS_MATCH_ARG0_NAMESPACE
	DBUS_SIGNAL_FLAGS_MATCH_ARG0_PATH      SignalFlags = C.G_DBUS_SIGNAL_FLAGS_MATCH_ARG0_PATH
)

type Connection struct {
	Native *C.GDBusConnection
}

func GetSessionBusSync() (*Connection, error) {
	var err *C.GError
	v := C.g_bus_get_sync(C.GBusType(BUS_TYPE_SESSION), nil, &err)
	if err != nil {
		return nil, errors.New(errorMessage(err))
	}

	return wrapConnection(v), nil
}

func FromApplication(app *glib.Application) *Connection {
	v := C.g_application_get_dbus_connection((*C.GApplication)(unsafe.Pointer(app.Native())))
	if v == nil {
		return nil
	}
	return wrapConnection(v)
}

func wrapConnection(v *C.GDBusConnection) *Connection {
	c := &Connection{Native: v}
	runtime.SetFinalizer(c, func(c *Connection) {
		cbDelete(unsafe.Pointer(c.Native))
	})
	return c
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
		return nil, errors.New(errorMessage(err))
	}

	return glib.TakeVariant(unsafe.Pointer(v)), nil
}

func (c *Connection) SignalSubscribe(
	sender,
	interfaceName,
	member,
	objectPath,
	arg0 string,
	flags SignalFlags,
	callback SignalCallback,
) uint {

	ptr, call := cbAssign(unsafe.Pointer(c.Native), c, callback)

	v := C.g_dbus_connection_signal_subscribe(
		c.Native,
		cstringOpt(sender),
		cstringOpt(interfaceName),
		cstringOpt(member),
		cstringOpt(objectPath),
		cstringOpt(arg0),
		C.GDBusSignalFlags(flags),
		C.GDBusSignalCallback(C.gdbusSignalCallback),
		ptr, nil,
	)

	call.id = uint(v)
	return call.id
}

func (c *Connection) SignalUnsubscribe(id uint) {
	cbForEach(func(i int, cb *callback) bool {
		if cb.id == id {
			delete(registry, i)
			return true
		}
		return false
	})
	C.g_dbus_connection_signal_unsubscribe(
		c.Native,
		C.uint(id),
	)
}

func cstringOpt(str string) *C.gchar {
	if str == "" {
		return nil
	}
	return C.CString(str)
}

func VariantTuple(v *glib.Variant, n int) []*glib.Variant {
	_v := (*C.GVariant)(v.ToGVariant())

	params := make([]*glib.Variant, n)

	for i := 0; i < n; i++ {
		params[i] = glib.TakeVariant(unsafe.Pointer(
			C.g_variant_get_child_value(_v, C.gsize(i)),
		))
	}

	return params
}
