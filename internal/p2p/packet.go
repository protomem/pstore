package p2p

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

const _defaultPacketBufferSize = 1024

var (
	_ Handler = (*PacketHandler)(nil)
	_ Decoder = (*DefaultDecoder)(nil)
)

type Packet struct {
	From    net.Addr
	Payload []byte
}

func NewPacket(from net.Addr, payload []byte) Packet {
	return Packet{
		From:    from,
		Payload: bytes.Clone(payload),
	}
}

type PacketHandler struct {
	Decoder Decoder
	Handler func(context.Context, Packet, error)
}

func NewPacketHandler(h func(context.Context, Packet, error)) *PacketHandler {
	return &PacketHandler{
		Decoder: DefaultDecoder{BufferSize: _defaultPacketBufferSize},
		Handler: h,
	}
}

func (h *PacketHandler) Handle(ctx context.Context, p Peer) {
	for {
		packet := NewPacket(p.RemoteAddr(), nil)
		err := h.Decoder.Decode(p, &packet)
		if err != nil && errors.Is(err, io.EOF) {
			continue
		}
		h.Handler(ctx, packet, err)
	}
}

type Decoder interface {
	Decode(r io.Reader, p *Packet) error
}

type DefaultDecoder struct {
	BufferSize int
}

func (d DefaultDecoder) Decode(r io.Reader, p *Packet) error {
	if p == nil {
		return nil
	}

	buff := make([]byte, d.BufferSize)
	read, err := r.Read(buff)

	payload := bytes.Clone(buff[:read])
	p.Payload = payload

	return err
}

type PacketReader struct {
	reader chan Packet
	closed atomic.Bool

	errMux  sync.RWMutex
	lastErr error
}

func NewPacketReader() *PacketReader {
	return &PacketReader{
		reader:  make(chan Packet),
		lastErr: nil,
	}
}

func (r *PacketReader) Handle(ctx context.Context, p Packet, err error) {
	r.errMux.Lock()
	defer r.errMux.Unlock()

	if r.closed.Load() {
		return
	}

	r.lastErr = nil

	if err != nil {
		r.lastErr = err
		close(r.reader)
		r.closed.Store(true)
		return
	}

	r.reader <- p
}

func (r *PacketReader) Err() error {
	r.errMux.RLock()
	defer r.errMux.RUnlock()
	return r.lastErr
}

func (r *PacketReader) Read() <-chan Packet {
	return r.reader
}
