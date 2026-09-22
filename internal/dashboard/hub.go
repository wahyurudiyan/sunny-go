package dashboard

import "sync"

// hub fans out a "something changed, refetch" signal to every
// connected SSE client — no payload travels through it, since a config
// value should only ever cross the wire when a client explicitly asks
// for it (GET /api/config/{key}), never pushed unprompted.
type hub struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func newHub() *hub {
	return &hub{subs: make(map[chan struct{}]struct{})}
}

func (h *hub) subscribe() (ch chan struct{}, cancel func()) {
	ch = make(chan struct{}, 1)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

func (h *hub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default: // a refresh signal is already queued for this subscriber
		}
	}
}
