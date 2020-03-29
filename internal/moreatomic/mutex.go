package moreatomic

import "github.com/sasha-s/go-deadlock"

type BusyMutex struct {
	busy Bool
	mut  deadlock.Mutex
}

func (m *BusyMutex) TryLock() bool {
	if m.busy.Get() {
		return false
	}

	m.mut.Lock()
	m.busy.Set(true)

	return true
}

func (m *BusyMutex) IsBusy() bool {
	return m.busy.Get()
}

func (m *BusyMutex) Lock() {
	m.mut.Lock()
	m.busy.Set(true)
}

func (m *BusyMutex) Unlock() {
	m.busy.Set(false)
	m.mut.Unlock()
}
