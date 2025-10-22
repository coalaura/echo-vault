package main

import (
	"encoding/json"
	"sync"
	"time"
)

type Timer struct {
	mx sync.Mutex

	timers map[string]time.Time
	times  map[string]string
}

func NewTimer() *Timer {
	return &Timer{
		timers: make(map[string]time.Time),
		times:  make(map[string]string),
	}
}

func (t *Timer) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.times)
}

func (t *Timer) Start(name string) *Timer {
	t.mx.Lock()
	defer t.mx.Unlock()

	t.timers[name] = time.Now()

	return t
}

func (t *Timer) Stop(name string) *Timer {
	t.mx.Lock()
	defer t.mx.Unlock()

	start, ok := t.timers[name]
	if !ok {
		return t
	}

	t.times[name] = time.Since(start).String()

	return t
}
