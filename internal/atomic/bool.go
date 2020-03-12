package atomic

import "sync/atomic"

type AtomicBool struct {
	val uint32
}

func (b *AtomicBool) Get() bool {
	return atomic.LoadUint32(&b.val) == 1
}

func (b *AtomicBool) Set(val bool) {
	var x = uint32(0)
	if val {
		x = 1
	}
	atomic.StoreUint32(&b.val, x)
}
