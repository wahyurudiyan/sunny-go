package dashboard

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("hub", func() {
	It("delivers a broadcast to every subscriber", func() {
		h := newHub()
		ch1, cancel1 := h.subscribe()
		defer cancel1()
		ch2, cancel2 := h.subscribe()
		defer cancel2()

		h.broadcast()

		Expect(ch1).To(Receive())
		Expect(ch2).To(Receive())
	})

	It("coalesces a second broadcast with nothing yet reading the first", func() {
		h := newHub()
		ch, cancel := h.subscribe()
		defer cancel()

		h.broadcast()
		h.broadcast()

		Expect(ch).To(Receive())
		Expect(ch).NotTo(Receive())
	})

	It("stops delivering to a subscriber once it's cancelled", func() {
		h := newHub()
		ch, cancel := h.subscribe()
		cancel()

		h.broadcast()

		Expect(ch).NotTo(Receive())
	})
})
