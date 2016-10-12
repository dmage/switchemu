package stat

import (
	"net"
	"time"

	"github.com/dmage/switchemu/hw"
)

type IPv6SourceCounter struct {
	prevBucket *TimeInt64
	sources    map[string]struct{}
	buckets    Int64Buckets
	output     hw.Handler
}

func NewIPv6SourceCounter(interval time.Duration, output hw.Handler) *IPv6SourceCounter {
	return &IPv6SourceCounter{
		buckets: NewInt64Buckets(interval),
		output:  output,
	}
}

func (s *IPv6SourceCounter) HandlePacket(w *hw.World, p *hw.Packet) {
	data := p.CapturedData
	if data[12] == 0x86 && data[13] == 0xDD {
		// IPv6

		b := s.buckets.Get(w.Time())
		if b != s.prevBucket {
			s.prevBucket = b
			s.sources = make(map[string]struct{})
		}

		srcIP := net.IP(data[14+8 : 14+8+128/8])
		_, ok := s.sources[string(srcIP)]
		if !ok {
			b.Value += 1
			s.sources[string(srcIP)] = struct{}{}
		}
	}
	s.output.HandlePacket(w, p)
}

func (s *IPv6SourceCounter) Dump(w *hw.World, filename string) error {
	return s.buckets.Dump(w.StartTime(), filename)
}
