package gtkcord

import (
	"math/rand"

	"github.com/diamondburned/gtkcord3/gtkcord/gtkutils"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

var must = semaphore.IdleMust
var async = semaphore.Async

func idleWait(fn func()) {
	must(fn)
}

func logWrap(err error, str string) {
	if err == nil {
		return
	}

	log.Errorln(str+":", err)
}

func margin4(w gtkutils.Marginator, top, bottom, left, right int) {
	gtkutils.Margin4(w, top, bottom, left, right)
}

func margin2(w gtkutils.Marginator, top, left int) {
	gtkutils.Margin2(w, top, left)
}

func margin(w gtkutils.Marginator, sz int) {
	gtkutils.Margin(w, sz)
}

func randString() string {
	const randLen = 20
	const alphabet = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, randLen)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}

	return string(b)
}

func nilAdjustment() *gtk.Adjustment {
	return (*gtk.Adjustment)(nil)
}
