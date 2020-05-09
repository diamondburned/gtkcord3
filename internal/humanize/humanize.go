package humanize

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Xuanwo/go-locale"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/goodsign/monday"
)

const (
	Day  = 24 * time.Hour
	Week = 7 * Day
	Year = 365 * Day
)

var Locale monday.Locale = monday.LocaleEnUS // changed on init

var localeOnce sync.Once

func lettersOnly(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		return -1
	}, str)
}

func ensureLocale() {
	localeOnce.Do(func() {
		if tag, err := locale.Detect(); err == nil {
			Locale = monday.Locale(lettersOnly(tag.String()))
		}

		// Check if locale is supported
		for _, locale := range monday.ListLocales() {
			if lettersOnly(string(locale)) == string(Locale) {
				return
			}
		}

		log.Println("Locale", Locale, "not found, defaulting to en_US")
		Locale = monday.LocaleEnUS
	})
}

func TimeKitchen(t time.Time) string {
	ensureLocale()

	return monday.Format(t, time.Kitchen, Locale)
}

func TimeAgo(t time.Time) string {
	ensureLocale()

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

func Strings(list []string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	default:
		return strings.Join(list[:len(list)-1], ", ") + " and " + list[len(list)-1]
	}
}

func TrimString(str string, maxlen int) string {
	if len(str) < maxlen-3 {
		return str
	}

	return str[:maxlen-3] + "..."
}

var ByteUnits = [...]string{"bytes", "KB", "MB"}

func Size(size uint64) string {
	// hmm today i will do dumb shit
	size *= 100 // 2 decimal points
	unit := "GB"

	for _, u := range ByteUnits {
		if size < 1024 {
			unit = u
			break
		}
		size /= 1024
	}

	return fmt.Sprintf("%d.%d %s", size/2, size%2, unit)
}
