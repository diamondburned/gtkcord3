package semaphore

import (
	"context"
	"runtime"

	"github.com/diamondburned/gtkcord3/log"
	"golang.org/x/sync/semaphore"
)

var MaxWorkers = runtime.GOMAXPROCS(0)

var sema *semaphore.Weighted

func createSema() {
	if sema == nil {
		sema = semaphore.NewWeighted(int64(MaxWorkers))
	}
}

func Go(fn func()) {
	createSema()

	if err := sema.Acquire(context.TODO(), 1); err != nil {
		log.Errorln("Semaphore: Failed to acquire shared semaphore:", err)
		return
	}
	go func() {
		fn()
		sema.Release(1)
	}()
}
