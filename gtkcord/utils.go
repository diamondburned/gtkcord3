package gtkcord

import (
	"log"
	"runtime"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var logError = func(err error) {
	log.Println("gtkcord error:", err)
}

func must(fn interface{}, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	log.Println("IdleAdd @", file+":", line)

	if _, err := glib.IdleAdd(fn, args...); err != nil {
		logError(errors.Wrap(err, "FATAL: IdleAdd in must()"))
	}
}

func idleWait(fn func()) {
	must(fn)
}

func logWrap(err error, str string) {
	if err == nil {
		return
	}

	logError(errors.Wrap(err, str))
}

func margin4(w *gtk.Widget, top, bottom, left, right int) {
	must(w.SetMarginTop, top)
	must(w.SetMarginBottom, bottom)

	must(w.SetMarginStart, left)
	must(w.SetMarginEnd, right)
}

func margin2(w *gtk.Widget, top, left int) {
	margin4(w, top, top, left, left)
}

func margin(w *gtk.Widget, sz int) {
	margin2(w, sz, sz)
}
