package dashboard_test

import (
	"bytes"
	"sync"
)

// safeBuffer is a bytes.Buffer safe for one writer goroutine (the
// supervised fixture child's stdout pipe) and one reader goroutine
// (a spec polling it via Eventually) at once.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
