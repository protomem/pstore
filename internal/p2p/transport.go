package p2p

import (
	"io"
	"net"
)

type Peer interface {
	RemoteAddr() net.Addr
	Outbound() bool
	io.ReadWriteCloser
}

type Transport interface {
	Addr() net.Addr

	ListenAndAccept() error
	Dial(string) error

	SetHandler(h Handler)
	SetHandshaker(h Handshaker)

	SetErrorHandler(h func(error))

	AcquirePeer(addr net.Addr) (Peer, bool)
	ReleasePeer(peer Peer)

	Close() error
}
