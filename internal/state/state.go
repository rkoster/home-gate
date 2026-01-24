// Package state manages in-memory monitoring summaries.
// It is concurrency-safe and allows updates and queries by other packages or a future HTTP API.
package state

import (
	"home-gate/internal/monitor"
	"sync"
)

var (
	mu     sync.RWMutex
	latest monitor.Summary
)

// Update stores the provided summary in memory.
func Update(summary monitor.Summary) {
	mu.Lock()
	defer mu.Unlock()
	latest = summary
}

// Get returns the latest monitoring summary.
func Get() monitor.Summary {
	mu.RLock()
	defer mu.RUnlock()
	return latest
}

// Reset clears the stored summary.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	latest = monitor.Summary{}
}
