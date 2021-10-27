package gdbus

import (
	"time"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gtkcord3/internal/log"
)

// MPRISWatcher wraps around a GIO DBus Connection to listen to the system's
// MPRIS events.
type MPRISWatcher struct {
	*gio.DBusConnection

	id      uint
	enabled bool

	// Last states
	metadata Metadata
	playing  bool
	changed  bool

	debounce     time.Time
	bounceHandle glib.SourceHandle

	OnPlayback func(m Metadata, playing bool)
}

// Metadata maps some fields from
// https://www.freedesktop.org/wiki/Specifications/mpris-spec/metadata/#index5h3
type Metadata struct {
	Title   string
	Artists []string
	Album   string
}

// NewMPRISWatcher creates a new MPRIS watcher instance.
func NewMPRISWatcher(c *gio.DBusConnection) *MPRISWatcher {
	if c == nil {
		return &MPRISWatcher{}
	}

	w := &MPRISWatcher{
		DBusConnection: c,
		enabled:        true,
		OnPlayback: func(m Metadata, playing bool) {
			log.Println("Playing:", playing)
		},
	}

	w.id = c.SignalSubscribe(
		"", "org.freedesktop.DBus.Properties",
		"PropertiesChanged", "/org/mpris/MediaPlayer2", "",
		gio.DBusSignalFlagsNone,
		func(_ *gio.DBusConnection, _, _, _, _ string, params *glib.Variant) {
			// Brief checks.
			if params.NChildren() < 2 {
				return
			}

			k := params.ChildValue(0)
			v := params.ChildValue(1)
			if k == nil || v == nil {
				return
			}

			if k.String() != "org.mpris.MediaPlayer2.Player" {
				return
			}

			glib.IdleAdd(func() {
				if !w.IsEnabled() {
					return
				}
				readDict(v, map[string]dictEntry{
					"PlaybackStatus": {"s", w.onPlaybackStatusChange},
					"Metadata":       {"", w.onMetadataChange},
				})
				w.update()
			})
		},
	)

	return w
}

// Close stops the watcher.
func (w *MPRISWatcher) Close() {
	w.SignalUnsubscribe(w.id)
	w.id = 0
}

func (w *MPRISWatcher) SetEnabled(enabled bool) {
	if w.DBusConnection == nil {
		w.enabled = false
		return
	}

	w.enabled = enabled

	// Force pause if we're disabling:
	if !w.IsEnabled() && w.playing {
		w.playing = false
		w.OnPlayback(w.metadata, false)
	}
}

func (w *MPRISWatcher) IsEnabled() bool {
	return w.enabled && w.id > 0
}

func (w *MPRISWatcher) onPlaybackStatusChange(v *glib.Variant) {
	playing := v.String() == "Playing"

	w.playing = playing
	w.changed = true

	// Don't update a zero-value
	if w.metadata.Title == "" {
		return
	}
}

func (w *MPRISWatcher) onMetadataChange(dict *glib.Variant) {
	// Clear
	w.metadata.Title = ""
	w.metadata.Album = ""
	w.metadata.Artists = w.metadata.Artists[:0]

	w.playing = true
	w.changed = true

	readDict(dict, map[string]dictEntry{
		"xesam:title": {"s", func(v *glib.Variant) { w.metadata.Title = v.String() }},
		"xesam:album": {"s", func(v *glib.Variant) { w.metadata.Album = v.String() }},
		"xesam:artist": {"", func(v *glib.Variant) {
			switch v.TypeString() {
			case "s":
				w.metadata.Artists = []string{v.String()}
			case "as":
				w.metadata.Artists = v.Strv()
			}
		}},
	})
}

const debounce = 3 * time.Second

func (w *MPRISWatcher) update() {
	now := time.Now()

	if t := w.debounce.Add(debounce); t.After(now) {
		// Too fast. Check if we've already debounced. If not, queue.
		if w.bounceHandle == 0 {
			secs := uint(t.Sub(now).Round(time.Second).Seconds())

			w.bounceHandle = glib.TimeoutSecondsAdd(secs, func() {
				w.mustUpdate()
				w.debounce = time.Now()
				w.bounceHandle = 0
			})
		}

		return
	}

	w.mustUpdate()
	w.debounce = now
}

func (w *MPRISWatcher) mustUpdate() {
	w.OnPlayback(w.metadata, w.playing)
}
