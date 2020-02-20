package semaphore

import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/glib"
	"golang.org/x/sync/semaphore"
)

var MaxWorkers = runtime.GOMAXPROCS(0)

var sema = semaphore.NewWeighted(int64(MaxWorkers))

var idleAdds = make(chan *idleCall, 1000)
var recvPool = sync.Pool{
	New: func() interface{} {
		return make(chan []reflect.Value)
	},
}

type idleCall struct {
	fn    interface{}
	args  []reflect.Value
	trace string
	done  chan []reflect.Value
}

func init() {
	go func() {
		runtime.LockOSThread()

		for call := range idleAdds {
			glib.IdleAdd(func(call *idleCall) {
				// log.Debugln(call.trace, "IdleAdd() called.")
				now := time.Now()

				if fn, ok := call.fn.(func()); ok {
					fn()
					call.done <- nil
				} else {
					call.done <- call.fn.(reflect.Value).Call(call.args)
				}

				if delta := time.Now().Sub(now); delta > time.Millisecond {
					log.Infoln(call.trace, "took", time.Now().Sub(now))
				}
			}, call)
		}
	}()
}

func idleAdd(trace string, fn interface{}, v ...interface{}) []reflect.Value {
	ch := recvPool.Get().(chan []reflect.Value)
	defer recvPool.Put(ch)

	switch fn := fn.(type) {
	case func():
		idleAdds <- &idleCall{fn, nil, trace, ch}
	default:
		var argv = make([]reflect.Value, len(v))
		for i, arg := range v {
			argv[i] = reflect.ValueOf(arg)
		}

		idleAdds <- &idleCall{
			fn:    reflect.ValueOf(fn),
			args:  argv,
			trace: trace,
			done:  ch,
		}
	}

	return <-ch
}

func IdleNow(fn interface{}, v ...interface{}) []interface{} {
	var values = idleAdd(log.Trace(1), fn, v...)
	var interfaces = make([]interface{}, len(values))

	for i, v := range values {
		interfaces[i] = v.Interface()
	}

	return interfaces
}

func Idle(fn interface{}, v ...interface{}) (interface{}, error) {
	return idle(log.Trace(1), fn, v...)
}

func idle(trace string, fn interface{}, v ...interface{}) (interface{}, error) {
	var values = idleAdd(trace, fn, v...)
	switch len(values) {
	case 2:
		if v := values[1].Interface(); v != nil {
			if err := v.(error); err != nil {
				return nil, err
			}
		}

		fallthrough
	case 1:
		return values[0].Interface(), nil
	case 0:
		return nil, nil
	default:
		log.Panicln(trace, "Unknown returns:", values)
		return nil, nil
	}
}

func IdleMust(fn interface{}, v ...interface{}) interface{} {
	var trace = log.Trace(1)

	r, err := idle(trace, fn, v...)
	if err != nil {
		log.Panicln(trace, "callback returned err != nil:", err)
	}

	return r
}

func Go(fn func()) {
	if err := sema.Acquire(context.TODO(), 1); err != nil {
		log.Panicln("Semaphore: Failed to acquire shared semaphore:", err)
		return
	}

	go func() {
		defer sema.Release(1)
		fn()
	}()
}
