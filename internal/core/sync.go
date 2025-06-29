package core

import (
	"io"
	"sync"
)

// syncWriter wraps an io.Writer with a mutex to provide thread-safe writes.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// newSyncWriter creates a new synchronized writer that wraps the given writer.
func newSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{w: w}
}

// Write implements io.Writer with thread-safe access.
func (sw *syncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}
