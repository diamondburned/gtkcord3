package gdbus

// #include <glib-2.0/glib.h>
//
// gchar* error_message(GError *err) {
//		return err->message;
// }
import "C"

import (
	"sync"
	"unsafe"
)

func errorMessage(err *C.GError) string {
	return C.GoString(C.error_message(err))
}

var (
	registry = map[int]*callback{}
	regMutex = sync.RWMutex{}

	serial int
)

type callback struct {
	receiver interface{}

	ptr unsafe.Pointer
	id  uint // signal id
	fn  interface{}
}

func cbAssign(ptr unsafe.Pointer, receiver, fn interface{}) (C.gpointer, *callback) {
	regMutex.Lock()
	defer regMutex.Unlock()

	id := serial
	serial++

	cb := &callback{
		receiver: receiver,
		ptr:      ptr,
		fn:       fn,
	}

	registry[id] = cb
	return C.gpointer(uintptr(id)), cb
}

func cbGet(ptr C.gpointer) *callback {
	regMutex.RLock()
	defer regMutex.RUnlock()

	if v, ok := registry[int(uintptr(ptr))]; ok {
		return v
	}
	return nil
}

func cbDelete(ptr unsafe.Pointer) {
	regMutex.Lock()
	defer regMutex.Unlock()

	for i, call := range registry {
		if call.ptr == ptr {
			delete(registry, i)
		}
	}
}

func cbForEach(fn func(int, *callback) bool) {
	regMutex.Lock()
	defer regMutex.Unlock()

	for i, call := range registry {
		if fn(i, call) {
			return
		}
	}
}
