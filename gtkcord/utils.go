package gtkcord

import (
	"math/rand"

	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/gtk"
)

var must = semaphore.IdleMust

/*
func must(fn interface{}, args ...interface{}) {
	var trace = log.Trace(1)
	// var err error

	switch len(args) {
	case 0:
		switch fn := fn.(type) {
		case func() bool:
			semaphore.IdleAdd(func() {
				log.Debugln(trace, "IdleAdd() called.")
				fn()
			})
		case func():
			semaphore.IdleAdd(func() {
				log.Debugln(trace, "IdleAdd() called.")
				fn()
			})
		default:
			log.Panicln("Unknown callback type")
		}

	case 1:
		fnV := reflect.ValueOf(fn)
		argV := reflect.ValueOf(args[0])

		semaphore.IdleAdd(func() {
			log.Debugln(trace, "IdleAdd() called.")
			fnV.Call([]reflect.Value{argV})
		})

	default:
		log.Panicln("BUG: >1 arguments given to must()")
	}
}
*/

func idleWait(fn func()) {
	must(fn)
}

func logWrap(err error, str string) {
	if err == nil {
		return
	}

	log.Errorln(str+":", err)
}

func margin4(w *gtk.Widget, top, bottom, left, right int) {
	w.SetMarginTop(top)
	w.SetMarginBottom(bottom)
	w.SetMarginStart(left)
	w.SetMarginEnd(right)
}

func margin2(w *gtk.Widget, top, left int) {
	margin4(w, top, top, left, left)
}

func margin(w *gtk.Widget, sz int) {
	margin2(w, sz, sz)
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
