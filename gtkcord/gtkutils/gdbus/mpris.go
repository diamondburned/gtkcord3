package gdbus

// #include <glib-2.0/glib.h>
import "C"

import (
	"log"

	"github.com/gotk3/gotk3/glib"
)

type MPRISWatcher struct {
	*Connection

	enabled bool

	// Last states
	metadata Metadata
	playing  bool

	OnMetadata       func(m Metadata, playing bool)
	OnPlaybackStatus func(m Metadata, playing bool)
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
		enabled:    true,
		OnMetadata: func(m Metadata, playing bool) {
			log.Println("MPRIS update:", m)
		},
		OnPlaybackStatus: func(m Metadata, playing bool) {
			log.Println("Playing:", playing)
		},
	}

	c.SignalSubscribe(
		"", "org.freedesktop.DBus.Properties",
		"PropertiesChanged", "/org/mpris/MediaPlayer2", "",
		DBUS_SIGNAL_FLAGS_NONE,
		func(_ *Connection, _, _, _, _ string, vparams *glib.Variant) {
			if !w.enabled {
				return
			}

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

func (w *MPRISWatcher) SetEnabled(enabled bool) {
	w.enabled = enabled

	// Force pause if we're disabling:
	if !enabled && w.playing {
		w.playing = false
		w.OnPlaybackStatus(w.metadata, false)
	}
}

func (w *MPRISWatcher) onPlaybackStatusChange(vstring *C.GVariant) {
	playing := variantString(vstring) == "Playing"

	w.playing = playing

	// Don't update a zero-value
	if w.metadata.Title == "" {
		return
	}

	go w.OnPlaybackStatus(w.metadata, playing)
}

func (w *MPRISWatcher) onMetadataChange(array *C.GVariant) {
	// Clear
	w.metadata.Title = ""
	w.metadata.Album = ""
	w.metadata.Artists = w.metadata.Artists[:0]

	w.playing = true

	// array is a slice of dictionaries.

	// iterate:
	arrayIterKeyVariant(array, func(k string, v *C.GVariant) {
		switch k {
		case "xesam:title":
			w.metadata.Title = variantString(v)
		case "xesam:album":
			w.metadata.Album = variantString(v)
		case "xesam:artist":
			arrayIter(v, func(v *C.GVariant) {
				// We can probably get away with letting Go grow the slice:
				w.metadata.Artists = append(w.metadata.Artists, variantString(v))
			})
		}
	})

	go w.OnMetadata(w.metadata, w.playing)
}
