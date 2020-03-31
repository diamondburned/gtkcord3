package gdbus

// #include <glib-2.0/glib.h>
import "C"

import (
	"log"

	"github.com/gotk3/gotk3/glib"
)

type MPRISWatcher struct {
	*Connection

	OnMetadata       func(m *Metadata)
	OnPlaybackStatus func(playing bool)
}

// Metadata maps some fields from
// https://www.freedesktop.org/wiki/Specifications/mpris-spec/metadata/#index5h3
type Metadata struct {
	Title   string
	Artists []string
	Album   string
}

func NewMPRISWatcher(c *Connection) *MPRISWatcher {
	w := &MPRISWatcher{
		Connection: c,
		OnMetadata: func(m *Metadata) {
			log.Println("MPRIS update:", m)
		},
		OnPlaybackStatus: func(playing bool) {
			log.Println("Playing:", playing)
		},
	}

	c.SignalSubscribe(
		"", "org.freedesktop.DBus.Properties",
		"PropertiesChanged", "/org/mpris/MediaPlayer2", "",
		DBUS_SIGNAL_FLAGS_NONE,
		func(_ *Connection, _, _, _, _ string, vparams *glib.Variant) {
			params := VariantTuple(vparams, 2)
			if params[0].GetString() != "org.mpris.MediaPlayer2.Player" {
				return
			}

			if params[1] == nil {
				return
			}

			arrayIterKeyVariant(variantNative(params[1]), func(k string, v *C.GVariant) {
				switch k {
				case "PlaybackStatus":
					w.onPlaybackStatusChange(v)
				case "Metadata":
					w.onMetadataChange(v)
				}
			})
		},
	)

	return w
}

func (w *MPRISWatcher) onPlaybackStatusChange(vstring *C.GVariant) {
	playing := variantString(vstring) == "Playing"
	w.OnPlaybackStatus(playing)
}

func (w *MPRISWatcher) onMetadataChange(array *C.GVariant) {
	var meta Metadata

	// array is a slice of dictionaries.

	// iterate:
	arrayIterKeyVariant(array, func(k string, v *C.GVariant) {
		switch k {
		case "xesam:title":
			meta.Title = variantString(v)
		case "xesam:album":
			meta.Album = variantString(v)
		case "xesam:artist":
			arrayIter(v, func(v *C.GVariant) {
				// We can probably get away with letting Go grow the slice:
				meta.Artists = append(meta.Artists, variantString(v))
			})
		}
	})

	w.OnMetadata(&meta)
}
