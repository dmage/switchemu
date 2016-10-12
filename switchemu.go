package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/google/gopacket/pcapgo"

	"github.com/dmage/switchemu/hw"
	"github.com/dmage/switchemu/stat"
)

const (
	readerBufferSize = 4 << 20
)

type BufferStatistics struct {
	prev          time.Duration
	bufferedBytes int
	nextBucketAt  time.Duration

	ByTime    stat.Int64Buckets
	Histogram stat.FastInt64Counter

	BufferLimit int
}

func NewBufferStatistics(precision time.Duration) *BufferStatistics {
	return &BufferStatistics{
		ByTime:    stat.NewInt64Buckets(precision),
		Histogram: stat.NewFastInt64Counter(100000),
	}
}

func (s *BufferStatistics) updateHistogram(now time.Duration) {
	delta := now - s.prev
	s.Histogram.Add(int64(s.bufferedBytes), int64(delta))
	s.prev = now
}

func (s *BufferStatistics) updateByTime(now time.Duration) {
	b := s.ByTime.Get(now)
	if int64(s.bufferedBytes) > b.Value {
		b.Value = int64(s.bufferedBytes)
	}
}

var dropCount int64
var lastDrop time.Duration
var nextDropReset time.Duration

func (s *BufferStatistics) PortInput(h hw.Handler) hw.Handler {
	return hw.HandlerFunc(func(w *hw.World, p *hw.Packet) {
		now := w.Time()
		if s.BufferLimit > 0 && s.bufferedBytes+p.Length > s.BufferLimit {
			for now >= nextDropReset {
				if dropCount != 0 {
					log.Println(w.StartTime().Local().Add(nextDropReset), "dropped", dropCount/60)
					dropCount = 0
				}
				nextDropReset += 60 * time.Second
			}
			dropCount++
			return
		}
		s.updateHistogram(now)
		s.bufferedBytes += p.Length
		s.updateByTime(now)
		h.HandlePacket(w, p)
	})
}

func (s *BufferStatistics) PortOutput(h hw.Handler) hw.Handler {
	return hw.HandlerFunc(func(w *hw.World, p *hw.Packet) {
		now := w.Time()
		s.updateHistogram(now)
		s.bufferedBytes -= 1 //p.Length
		s.updateByTime(now)
		h.HandlePacket(w, p)
	})
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var blockprofile = flag.String("blockprofile", "", "write block profile to `file`")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if *blockprofile != "" {
		runtime.SetBlockProfileRate(1)
	}

	defer func() {
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
			f.Close()
		}

		if *blockprofile != "" {
			f, err := os.Create(*blockprofile)
			if err != nil {
				log.Fatal("could not create block profile: ", err)
			}
			if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
				log.Fatal("could not write block profile: ", err)
			}
			f.Close()
		}
	}()

	var sources []*pcapgo.Reader
	for _, filename := range flag.Args() {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		r := bufio.NewReaderSize(f, readerBufferSize)

		h, err := pcapgo.NewReader(r)
		if err != nil {
			log.Fatal(err)
		}

		// h, err := pcap.OpenOffline(filename)
		// if err != nil {
		//	log.Fatal(err)
		// }
		// defer h.Close()

		sources = append(sources, h)
	}

	var dumpers []func() error
	w := &hw.World{}

	bufferTotal := NewBufferStatistics(100 * time.Microsecond)
	// bufferTotal.BufferLimit = 1024000
	dumpers = append(dumpers, func() error {
		err := bufferTotal.ByTime.Dump(w.StartTime(), "./output/summary.buffer_by_time.txt")
		if err != nil {
			return err
		}
		return bufferTotal.Histogram.Dump("./output/summary.buffer_histogram.txt")
	})

	demux := hw.NewIPv6DestinationDemux(
		func(dstIP net.IP) hw.Handler {
			bufferOutput := NewBufferStatistics(1 * time.Millisecond)
			dumpers = append(dumpers, func() error {
				err := bufferOutput.ByTime.Dump(w.StartTime(), fmt.Sprintf("./output/output.buffer_by_time.%s.txt", dstIP.String()))
				if err != nil {
					return err
				}
				return bufferOutput.Histogram.Dump(fmt.Sprintf("./output/output.buffer_histogram.%s.txt", dstIP.String()))
			})

			output := hw.Handler(hw.NullHandler{})

			output = bufferTotal.PortOutput(output)

			output = bufferOutput.PortOutput(output)

			output = hw.NewTransmitter(10*1000*1000*1000, output)

			output = bufferOutput.PortInput(output)

			sourceCounter := stat.NewIPv6SourceCounter(10*time.Millisecond, output)
			dumpers = append(dumpers, func() error {
				return sourceCounter.Dump(w, fmt.Sprintf("./output/output.sources.%s.txt", dstIP.String()))
			})
			output = sourceCounter

			return output
		},
		func() hw.Handler {
			output := hw.Handler(hw.NullHandler{})

			output = bufferTotal.PortOutput(output)

			return output
		}(),
	)

	for i, source := range sources {
		i := i // freeze i value for dumpers

		output := hw.Handler(demux)

		interarrivalTime := stat.NewInterarrivalTime(output)
		dumpers = append(dumpers, func() error {
			return interarrivalTime.Dump(fmt.Sprintf("./output/input.interarrival_time.%d.txt", i))
		})
		output = interarrivalTime

		bitsPerSecond := stat.NewBitsPerSecond(time.Second, output)
		dumpers = append(dumpers, func() error {
			return bitsPerSecond.Dump(w, fmt.Sprintf("./output/input.bits_per_second.%d.txt", i))
		})
		output = bitsPerSecond

		packetsPerSecond := stat.NewPacketsPerSecond(time.Second, output)
		dumpers = append(dumpers, func() error {
			return packetsPerSecond.Dump(w, fmt.Sprintf("./output/input.packets_per_second.%d.txt", i))
		})
		output = packetsPerSecond

		output = bufferTotal.PortInput(output)

		hw.NewReceiver(w, source, source.LinkType(), 40*1000*1000*1000, output)
	}

	w.Simulate()

	for _, d := range dumpers {
		err := d()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("done")
}
