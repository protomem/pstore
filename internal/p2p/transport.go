package p2p

import (
	"io"
	"net"
)

type Peer interface {
	RemoteAddr() net.Addr
	io.ReadWriteCloser
}

type Transport interface {
	Addr() net.Addr
	ListenAndAccept() error

	SetHandler(h Handler)
	SetHandshaker(h Handshaker)

	Close() error
}
