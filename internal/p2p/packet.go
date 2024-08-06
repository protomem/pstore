package p2p

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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
	defer func() {
		packet := NewPacket(p.RemoteAddr(), nil)
		err := p.Close()
		if err != nil {
			err = fmt.Errorf("%w: %w", ErrHandlerClosed, err)
		} else {
			err = ErrHandlerClosed
		}
		h.Handler(ctx, packet, err)
	}()

	for {
		packet := NewPacket(p.RemoteAddr(), nil)
		err := h.Decoder.Decode(p, &packet)
		if err != nil &&
			(errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
				errors.Is(err, io.ErrClosedPipe) || errors.Is(err, net.ErrClosed)) {
			return
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
