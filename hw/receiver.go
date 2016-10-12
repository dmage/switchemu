package hw

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type Packet struct {
	CapturedData []byte
	Timestamp    time.Time
	Length       int
	returnTo     *Cartridge
}

func (p *Packet) free() {
	p.returnTo.Release()
}

type Cartridge struct {
	Packets [256]Packet
	idx     int32
	inUse   int32
}

var cartridgeFree = sync.Pool{
	New: func() interface{} {
		c := new(Cartridge)
		for i := range c.Packets {
			c.Packets[i].returnTo = c
		}
		return c
	},
}

func (c *Cartridge) NewPacket() *Packet {
	idx := atomic.AddInt32(&c.idx, 1)
	if idx >= int32(len(c.Packets)) {
		return nil
	}
	atomic.AddInt32(&c.inUse, 1)

	p := &c.Packets[idx]
	if p.CapturedData != nil {
		p.CapturedData = p.CapturedData[:0]
	}
	return p
}

func (c *Cartridge) Release() {
	idx := atomic.LoadInt32(&c.idx)
	inUse := atomic.AddInt32(&c.inUse, -1)
	if inUse != 0 {
		return
	}
	for {
		if idx >= int32(len(c.Packets)) {
			c.idx = 0
			c.inUse = 0
			cartridgeFree.Put(c)
			break
		}
		prevIdx := idx
		idx = atomic.LoadInt32(&c.idx)
		if idx == prevIdx {
			break // someone else will release cartridge
		}
	}
}

type Receiver struct {
	inputChan <-chan []*Packet
	output    Handler

	cartridge  *Cartridge
	bufferFree chan []*Packet

	buffer []*Packet
	idx    int
}

const bufferSize = 10000
const buffers = 50
const inputChanSize = buffers

func NewReceiver(w *World, source gopacket.PacketDataSource, linkType layers.LinkType, bandwidth int64, output Handler) *Receiver {
	ch := make(chan []*Packet, inputChanSize)
	r := &Receiver{
		bufferFree: make(chan []*Packet, buffers),
		inputChan:  ch,
		output:     output,
	}
	for i := 0; i < buffers; i++ {
		r.bufferFree <- make([]*Packet, 0, bufferSize)
	}
	go r.packetsToChannel(ch, source, bandwidth)

	r.getBuffer()
	if len(r.buffer) == 0 {
		panic("no data?")
	}
	w.AtStart(r.buffer[0].Timestamp, r)

	return r
}

func (r *Receiver) newPacket() *Packet {
	if r.cartridge == nil {
		r.cartridge = cartridgeFree.Get().(*Cartridge)
	}
	p := r.cartridge.NewPacket()
	if p == nil {
		r.cartridge = cartridgeFree.Get().(*Cartridge)
		p = r.cartridge.NewPacket()
		if p == nil {
			panic("got nil from new cartridge")
		}
	}
	return p
}

func (r *Receiver) packetsToChannel(ch chan<- []*Packet, source gopacket.PacketDataSource, bandwidth int64) {
	var nextTimestamp time.Time
	buffer := <-r.bufferFree
	for {
		data, ci, err := source.ReadPacketData()
		if err == io.EOF {
			if len(buffer) != 0 {
				ch <- buffer
			} else {
				r.bufferFree <- buffer
			}
			close(ch)
			return
		} else if err != nil {
			panic(err)
		}

		p := r.newPacket()
		p.CapturedData = append(p.CapturedData, data...)
		if ci.Timestamp.Before(nextTimestamp) {
			p.Timestamp = nextTimestamp
		} else {
			p.Timestamp = ci.Timestamp
		}
		p.Length = ci.Length

		nextTimestamp = p.Timestamp.Add(OnWireDuration(p, bandwidth))

		buffer = append(buffer, p)
		if len(buffer) == cap(buffer) {
			ch <- buffer
			buffer = <-r.bufferFree
		}
	}
}

func (r *Receiver) getBuffer() {
	if r.buffer != nil {
		r.buffer = r.buffer[:0]
		r.bufferFree <- r.buffer
		r.buffer = nil
	}

	var ok bool
	r.buffer, ok = <-r.inputChan
	if !ok {
		return
	}
	r.idx = 0

	if len(r.buffer) == 0 {
		panic("no data in buffer")
	}
}

func (r *Receiver) next(w *World) {
	if r.idx >= len(r.buffer)-1 {
		r.getBuffer()
		if r.buffer == nil {
			return
		}
	} else {
		r.idx++
	}
	t := r.buffer[r.idx].Timestamp
	w.At(t.Sub(w.start), PrioInput, r)
}

func (r *Receiver) Run(w *World) {
	r.output.HandlePacket(w, r.buffer[r.idx])
	r.next(w)
}

type NullHandler struct{}

func (h NullHandler) HandlePacket(w *World, p *Packet) {
	p.free()
}
