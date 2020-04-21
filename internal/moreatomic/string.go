package moreatomic

import "sync/atomic"

type String struct {
	v atomic.Value
}

func (s *String) Get() string {
	if v, ok := s.v.Load().(string); ok {
		return v
	}
	return ""
}

func (s *String) Set(str string) {
	s.v.Store(str)
}
