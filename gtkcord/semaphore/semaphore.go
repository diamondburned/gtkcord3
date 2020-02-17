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

	_ "github.com/ianlancetaylor/cgosymbolizer"
)

var MaxWorkers = runtime.GOMAXPROCS(0)

var sema *semaphore.Weighted

var idleAdds = make(chan *idleCall, 0)
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
	glib.IdleAdd(func(idleAdds chan *idleCall) bool {
		select {
		case call := <-idleAdds:
			log.Debugln(call.trace, "IdleAdd() called.")
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

		default:
		}

		return true
	}, idleAdds)
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
