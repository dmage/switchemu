package hw

type Handler interface {
	HandlePacket(w *World, p *Packet)
}

type HandlerFunc func(*World, *Packet)

func (h HandlerFunc) HandlePacket(w *World, p *Packet) {
	h(w, p)
}
