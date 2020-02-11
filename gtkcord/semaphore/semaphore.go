package semaphore

import (
	"context"
	"log"
	"runtime"

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
		log.Println("Semaphore: Failed to acquire shared semaphore:", err)
	}
	go func() {
		fn()
		sema.Release(1)
	}()
}
