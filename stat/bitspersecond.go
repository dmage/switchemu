package stat

import (
	"time"

	"github.com/dmage/switchemu/hw"
)

type BitsPerSecond struct {
	output       hw.Handler
	buckets      Int64Buckets
	nextBucketAt time.Duration
}

func NewBitsPerSecond(interval time.Duration, output hw.Handler) *BitsPerSecond {
	return &BitsPerSecond{
		output:  output,
		buckets: NewInt64Buckets(interval),
	}
}

func (s *BitsPerSecond) HandlePacket(w *hw.World, p *hw.Packet) {
	s.buckets.Get(w.Time()).Value += 8 * int64(p.Length)
	s.output.HandlePacket(w, p)
}

func (s *BitsPerSecond) Dump(w *hw.World, filename string) error {
	return s.buckets.Dump(w.StartTime(), filename)
}
