package hw

import "time"

func OnWireDuration(p *Packet, bandwidth int64) time.Duration {
	// https://en.wikipedia.org/wiki/Ethernet_frame
	preambleLen := 7
	startOfFrameDelimLen := 1
	packetLen := p.Length
	frameCheckSequenceLen := 4
	interpacketGapLen := 12

	totalOctets := preambleLen + startOfFrameDelimLen + packetLen + frameCheckSequenceLen + interpacketGapLen
	totalOctets = packetLen

	bits := 8 * int64(totalOctets)
	return time.Duration(bits*int64(time.Second)/bandwidth + 1)
}

type Transmitter struct {
	Bandwidth int64
	Output    Handler

	buffer []*Packet
}

func NewTransmitter(bandwidth int64, output Handler) *Transmitter {
	return &Transmitter{
		Bandwidth: bandwidth,
		Output:    output,
		buffer:    make([]*Packet, 0, 128),
	}
}

func (t *Transmitter) HandlePacket(w *World, p *Packet) {
	// log.Println("received at", now)

	t.buffer = append(t.buffer, p)
	if len(t.buffer) == 1 {
		sentAt := w.Time() + OnWireDuration(t.buffer[0], t.Bandwidth)
		w.At(sentAt, PrioOutput, t)
	}
}

func (t *Transmitter) packetSent(w *World) {
	// log.Println("sent at", now)

	p := t.buffer[0]
	copy(t.buffer, t.buffer[1:])
	t.buffer = t.buffer[:len(t.buffer)-1]
	if len(t.buffer) != 0 {
		sentAt := w.Time() + OnWireDuration(t.buffer[0], t.Bandwidth)
		w.At(sentAt, PrioOutput, t)
	}
	t.Output.HandlePacket(w, p)
}

func (t *Transmitter) Run(w *World) {
	t.packetSent(w)
}
