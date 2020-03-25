package semaphore

import (
	"reflect"
	"runtime"
	"sync"

	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/glib"
)

// var MaxWorkers = runtime.GOMAXPROCS(0)
// var sema = semaphore.NewWeighted(int64(MaxWorkers))

var idleAdds = make(chan *idleCall, 100)
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
			call := call

			// log.Debugln(call.trace, "adding into main thread")

			glib.IdleAdd(func() {
				// now := time.Now()

				var val []reflect.Value

				log.Debugln(call.trace, "main thread")

				if fn, ok := call.fn.(func()); ok {
					fn()
				} else {
					val = call.fn.(reflect.Value).Call(call.args)
				}

				log.Debugln(call.trace, "main thread done")

				if call.done != nil {
					call.done <- val
				}

				// log.Debugln(call.trace, "main thread replied")
			})
		}
	}()
}

func idleAdd(trace string, async bool, fn interface{}, v ...interface{}) []reflect.Value {
	var ch chan []reflect.Value
	if !async {
		ch = recvPool.Get().(chan []reflect.Value)
		defer recvPool.Put(ch)
	}

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

	if !async {
		return <-ch
	}
	return nil
}

func IdleNow(fn interface{}, v ...interface{}) []interface{} {
	var values = idleAdd(log.Trace(1), false, fn, v...)
	var interfaces = make([]interface{}, len(values))

	for i, v := range values {
		interfaces[i] = v.Interface()
	}

	return interfaces
}

func Idle(fn interface{}, v ...interface{}) (interface{}, error) {
	return idle(log.Trace(1), fn, v...)
}

func Async(fn interface{}, v ...interface{}) {
	idleAdd(log.Trace(1), true, fn, v...)
}

func idle(trace string, fn interface{}, v ...interface{}) (interface{}, error) {
	var values = idleAdd(trace, false, fn, v...)
	switch len(values) {
	case 2:
		if v := values[1].Interface(); v != nil {
			if err := v.(error); err != nil {
				return nil, err
			}
		}

		fallthrough
	case 1:
		v := values[0].Interface()
		if err, ok := v.(error); ok {
			return nil, err
		}
		return v, nil

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
