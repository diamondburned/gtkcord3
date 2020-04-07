package gdbus

/*
#include <glib-2.0/glib.h>

gboolean iter_next_kv(GVariantIter* iter, gchar** k, GVariant** v) {
	return g_variant_iter_next(iter, "{sv}", k, v);
}
*/
import "C"

import (
	"unsafe"

	"github.com/gotk3/gotk3/glib"
)

func cstringOpt(str string) *C.gchar {
	if str == "" {
		return nil
	}
	return C.CString(str)
}

func freeNonNil(v unsafe.Pointer) {
	if v != nil {
		C.free(v)
	}
}

func variantNative(v *glib.Variant) *C.GVariant {
	return (*C.GVariant)(v.ToGVariant())
}

func VariantTuple(v *glib.Variant, n int) []*glib.Variant {
	_v := variantNative(v)

	var params = make([]*glib.Variant, n)

	for i := 0; i < n; i++ {
		params[i] = glib.TakeVariant(unsafe.Pointer(
			C.g_variant_get_child_value(_v, C.gsize(i)),
		))
	}

	return params
}

func dictLookup(dict *C.GVariantDict, key string) *C.GVariant {
	if dict == nil {
		return nil
	}

	ckey := C.CString(key)
	defer C.free(unsafe.Pointer(ckey))

	return C.g_variant_dict_lookup_value(dict, ckey, nil)
}

func variantString(v *C.GVariant) string {
	// nil for length since NULL value works.
	return C.GoString(C.g_variant_get_string(v, nil))
}

// each return false == break
func arrayIter(array *C.GVariant, each func(*C.GVariant)) {
	var iter C.GVariantIter
	C.g_variant_iter_init(&iter, array)

	item := C.g_variant_iter_next_value(&iter)
	if item == nil {
		return
	}

	for ; item != nil; item = C.g_variant_iter_next_value(&iter) {
		each(item)
		C.g_variant_unref(item)
	}
}

func arrayIterKeyVariant(array *C.GVariant, each func(k string, v *C.GVariant)) {
	var iter C.GVariantIter
	C.g_variant_iter_init(&iter, array)

	var k *C.gchar
	var v *C.GVariant

	for C.iter_next_kv(&iter, &k, &v) == C.TRUE {
		each(C.GoString(k), v)

		// free stuff
		C.free(unsafe.Pointer(k))
		C.g_variant_unref(v)
	}
}
