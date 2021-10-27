package moreatomic

import "sync/atomic"

type Serial struct {
	serial uint32
}

func (s *Serial) Get() int {
	return int(atomic.LoadUint32(&s.serial))
}

func (s *Serial) Incr() int {
	atomic.AddUint32(&s.serial, 1)
	return s.Get()
}
