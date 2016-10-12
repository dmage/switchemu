package stat

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type TimeInt64 struct {
	Time  time.Duration
	Value int64
}

type Int64Buckets struct {
	interval     time.Duration
	buckets      []TimeInt64
	nextBucketAt time.Duration
}

func NewInt64Buckets(interval time.Duration) Int64Buckets {
	return Int64Buckets{
		interval:     interval,
		nextBucketAt: 0,
	}
}

func (s *Int64Buckets) Get(now time.Duration) *TimeInt64 {
	if now >= s.nextBucketAt {
		for {
			s.buckets = append(s.buckets, TimeInt64{Time: s.nextBucketAt})
			s.nextBucketAt += s.interval
			if now < s.nextBucketAt {
				break
			}
			if len(s.buckets) >= 2 {
				s.buckets[len(s.buckets)-1].Value = s.buckets[len(s.buckets)-2].Value
			}
		}
	}
	return &s.buckets[len(s.buckets)-1]
}

func (s *Int64Buckets) Dump(start time.Time, filename string) error {
	return CreateFile(filename, func(w io.Writer) error {
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
