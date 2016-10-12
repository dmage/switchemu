package hw

import "net"

type IPv6DestinationDemux struct {
	NewOutput func(net.IP) Handler
	Drop      Handler

	destinations map[string]Handler
}

func NewIPv6DestinationDemux(newOutput func(net.IP) Handler, drop Handler) *IPv6DestinationDemux {
	return &IPv6DestinationDemux{
		NewOutput:    newOutput,
		Drop:         drop,
		destinations: make(map[string]Handler),
	}
}

func (r *IPv6DestinationDemux) HandlePacket(w *World, p *Packet) {
	data := p.CapturedData
	if data[12] == 0x86 && data[13] == 0xDD {
		// IPv6
		dstIP := net.IP(data[14+24 : 14+24+128/8])
		h, ok := r.destinations[string(dstIP)]
		if !ok {
			dstIPCopy := net.IP(append([]byte{}, dstIP...))
			h = r.NewOutput(dstIPCopy)
			r.destinations[string(dstIP)] = h
		}
		h.HandlePacket(w, p)
		return
	}
	r.Drop.HandlePacket(w, p)
}
