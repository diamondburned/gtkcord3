package mutexlog

import (
	"sync"

	"github.com/diamondburned/gtkcord3/internal/log"
)

type Mutex struct {
	sync.Mutex
}

func (m *Mutex) Lock() {
	trace := log.Trace(1)

	log.Println(trace, "... Acquiring mutex.")
	defer log.Println(trace, "||| Acquired mutex.")

	m.Mutex.Lock()
}

func (m *Mutex) Unlock() {
	log.Println(log.Trace(1), "Unlocking mutex.")
	m.Mutex.Unlock()
}

type RWMutex struct {
	sync.RWMutex
}

func (m *RWMutex) Lock() {
	trace := log.Trace(1)

	log.Println(trace, "... Acquiring mutex.")
	defer log.Println(trace, "||| Acquired mutex.")

	m.RWMutex.Lock()
}

func (m *RWMutex) Unlock() {
	log.Println(log.Trace(1), "Unlocking mutex.")
	m.RWMutex.Unlock()
}
