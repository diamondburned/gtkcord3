package gdbus

import (
	"context"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// SessionBusSync gets the global session bus.
func SessionBusSync() (*gio.DBusConnection, error) {
	return gio.BusGetSync(context.Background(), gio.BusTypeSession)
}

type dictEntry struct {
	typ string
	fun func(v *glib.Variant)
}

func readDict(dict *glib.Variant, entries map[string]dictEntry) {
	if !dict.Type().IsContainer() {
		return
	}

	for k, entry := range entries {
		var vtype *glib.VariantType
		if entry.typ != "" {
			vtype = glib.NewVariantType(entry.typ)
		}

		v := dict.LookupValue(k, vtype)
		if v == nil {
			// v = dict.LookupValue(k, nil)
			// if v == nil {
			// 	log.Infof("dbus: variant dict missing key %q", k)
			// } else {
			// 	log.Infof("dbus: key %q missing type %s", k, v.TypeString())
			// }
			continue
		}

		if strings.HasPrefix(v.TypeString(), "v") {
			// Unbox.
			v = v.Variant()
		}

		entry.fun(v)
	}
}
