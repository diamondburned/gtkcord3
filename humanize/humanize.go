package humanize

import (
	"log"
	"time"

	"github.com/Xuanwo/go-locale"
	"github.com/goodsign/monday"
)

const (
	Day  = 24 * time.Hour
	Week = 7 * Day
	Year = 365 * Day
)

var Locale monday.Locale = monday.LocaleEnUS // changed on init

func init() {
	if tag, err := locale.Detect(); err == nil {
		Locale = monday.Locale(tag.String())
	}

	// Check if locale is supported
	for _, locale := range monday.ListLocales() {
		if locale == Locale {
			return
		}
	}

	log.Println("Locale", Locale, "not found, defaulting to en_US")
	Locale = monday.LocaleEnUS
}

func TimeKitchen(t time.Time) string {
	return monday.Format(t, time.Kitchen, Locale)
}

func TimeAgo(t time.Time) string {
	trunc := t
	trunc = t.Truncate(Day)

	now := time.Now()
	now = now.Truncate(Day)

	if trunc.Equal(now) {
		return monday.Format(t, "Today at 15:04", Locale)
	}

	trunc = trunc.Truncate(Week)
	now = now.Truncate(Week)

	if trunc.Equal(now) {
		return monday.Format(t, "Last Monday at 15:04", Locale)
	}

	return monday.Format(t, "15:04 02/01/2006", Locale)
}

func DuraCeil(d, acc time.Duration) time.Duration {
	return d.Truncate(acc) + acc
}
