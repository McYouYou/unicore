package requeue_duration

import (
	"sync"
	"time"
)

// used to store a duration for the next reconcile
var store sync.Map

func Push(key string, duration time.Duration) {
	value, ok := store.Load(key)
	if ok && duration > 0 && duration < value.(time.Duration) {
		store.Store(key, duration)
	}
	if !ok {
		store.Store(key, duration)
	}
}

func Pop(key string) time.Duration {
	value, ok := store.Load(key)
	if ok {
		return value.(time.Duration)
	}
	return 0
}
