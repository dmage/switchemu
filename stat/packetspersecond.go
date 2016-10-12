package stat

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dmage/switchemu/hw"
)

type PacketsPerSecond struct {
	interval time.Duration
	output   hw.Handler

	buckets      []TimeInt64
	nextBucketAt time.Duration
}

func NewPacketsPerSecond(interval time.Duration, output hw.Handler) *PacketsPerSecond {
	return &PacketsPerSecond{
		output:       output,
		interval:     interval,
		nextBucketAt: 0,
	}
}

func (s *PacketsPerSecond) HandlePacket(w *hw.World, p *hw.Packet) {
	now := w.Time()

	for now >= s.nextBucketAt {
		s.buckets = append(s.buckets, TimeInt64{Time: s.nextBucketAt})
		s.nextBucketAt += s.interval
	}

	s.buckets[len(s.buckets)-1].Value += 1

	s.output.HandlePacket(w, p)
}

func (s *PacketsPerSecond) Dump(world *hw.World, filename string) error {
	return CreateFile(filename, func(w io.Writer) error {
		start := world.StartTime()
		for _, bucket := range s.buckets {
			t := start.Add(bucket.Time)
			ts := fmt.Sprintf("%d.%09d", t.Unix(), t.Nanosecond())
			ts = strings.TrimRight(ts, "0")

			// TODO(dmage): rescale to interval?
			v := bucket.Value

			fmt.Fprintf(w, "%s\t%d\n", ts, v)
		}
		return nil
	})
}
