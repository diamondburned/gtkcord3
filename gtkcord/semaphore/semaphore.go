package semaphore

import (
	"context"
	"reflect"
	"runtime"
	"sync"

	"github.com/diamondburned/gtkcord3/log"
	"github.com/gotk3/gotk3/glib"
	"golang.org/x/sync/semaphore"
)

var MaxWorkers = runtime.GOMAXPROCS(0)

var sema *semaphore.Weighted

/*
var idleAddReturn = sync.Pool{
	New: func() interface{} {
		return make(chan [2]interface{})
	},
}

// IdleAddReturns prepares for the apocalypse.
func IdleAddReturns(fn interface{}, args ...interface{}) (interface{}, error) {
	// Unsafely use this because why not.
	var fnV = reflect.ValueOf(fn)
	var argv = make([]reflect.Value, len(args)+1)
	for i, arg := range args {
		argv[i+1] = reflect.ValueOf(arg)
	}
	argv[0] = fnV

	trace := log.Trace(1)

	// ch := idleAddReturn.Get().(chan [2]interface{})
	// defer idleAddReturn.Put(ch)
	ch := make(chan []reflect.Value, 1)

	_, err := glib.IdleAdd(func(values []reflect.Value) bool {
		log.Debugln(trace, "Semaphore: IdleAdd() called.")

		ch <- values[0].Call(values[1:])
		return false
	}, argv)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to IdleAdd")
	}

	log.Debugln(trace, "Waiting for ch")

	// v := <-ch
	// returns := v[0].([]reflect.Value)
	returns := <-ch

	log.Debugln("Channel received")

	switch len(returns) {
	case 0:
		return nil, nil
	case 1:
		return returns[0].Interface(), nil
	default:
		return returns[0].Interface(), returns[1].Interface().(error)
	}
}
*/

var idleAdds = make(chan *idleCall)
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
	glib.IdleAdd(func() bool {
		select {
		case call := <-idleAdds:
			log.Debugln(call.trace, "IdleAdd()")

			if fn, ok := call.fn.(func()); ok {
				fn()
				call.done <- nil
			} else {
				call.done <- call.fn.(reflect.Value).Call(call.args)
			}

		default:
		}

		return true
	})
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

func IdleMust(fn interface{}, v ...interface{}) interface{} {
	var trace = log.Trace(1)

	var values = idleAdd(trace, fn, v...)
	switch len(values) {
	case 2:
		if v := values[1].Interface(); v != nil {
			if err := v.(error); err != nil {
				log.Panicln(trace, "callback returned err != nil:", err)
			}

			log.Errorln("Unknown second return:", v)
		}

		fallthrough
	case 1:
		return values[0].Interface()
	case 0:
		return nil
	default:
		log.Panicln(trace, "Unknown returns:", values)
		return nil
	}
}

func Go(fn func()) {
	if sema == nil {
		sema = semaphore.NewWeighted(int64(MaxWorkers))
	}

	if err := sema.Acquire(context.TODO(), 1); err != nil {
		log.Errorln("Semaphore: Failed to acquire shared semaphore:", err)
		return
	}

	go func() {
		defer sema.Release(1)

		fn()
	}()
}

func GoNow(fn func()) {
	go func() {
		fn()
	}()
}
