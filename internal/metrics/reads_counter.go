package metrics

import (
	"sync/atomic"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// readsSlotBytes is 128 and not 64, because x86 prefetches cache lines in
	// pairs
	readsSlotBytes = 128
	readsSlotBits  = 6
)

type readsSlot struct {
	n atomic.Uint64
	_ [readsSlotBytes - 8]byte
}

// ReadsCounter is a Prometheus counter split into slots, so that reads from
// several cores do not fight over one cache line the way a plain counter does.
// Prometheus gets the sum of all slots.
type ReadsCounter struct {
	desc  *prometheus.Desc
	slots [1 << readsSlotBits]readsSlot
}

var _ prometheus.Collector = (*ReadsCounter)(nil)

func newReadsCounter(desc *prometheus.Desc) *ReadsCounter {
	return &ReadsCounter{desc: desc}
}

// Inc picks the slot by the stack address of the calling goroutine. A goroutine
// keeps its slot and a new goroutine mostly reuses a stack freed on the same P,
// so a slot tends to stay in the cache of one core without any shared state.
func (c *ReadsCounter) Inc() {
	var marker byte
	h := uint64(uintptr(unsafe.Pointer(&marker))>>12) * 0x9E3779B97F4A7C15
	c.slots[h>>(64-readsSlotBits)].n.Add(1)
}

func (c *ReadsCounter) Value() (sum uint64) {
	for i := range c.slots {
		sum += c.slots[i].n.Load()
	}

	return
}

func (c *ReadsCounter) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *ReadsCounter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		c.desc,
		prometheus.CounterValue,
		float64(c.Value()),
	)
}
