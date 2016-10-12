package stat

import (
	"time"

	"github.com/dmage/switchemu/hw"
)

type InterarrivalTime struct {
	FastInt64Counter
	output hw.Handler
	prev   time.Duration
}

func NewInterarrivalTime(output hw.Handler) *InterarrivalTime {
	return &InterarrivalTime{
		FastInt64Counter: NewFastInt64Counter(50000),
		output:           output,
		prev:             -1,
	}
}

func (s *InterarrivalTime) HandlePacket(w *hw.World, p *hw.Packet) {
	now := w.Time()
	if s.prev != -1 {
		delta := now - s.prev
		s.Increment(int64(delta))
	}
	s.prev = now

	s.output.HandlePacket(w, p)
}
