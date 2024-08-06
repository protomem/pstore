package p2p

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
)

// TODO: Add logging

var (
	_ Peer      = (*TCPPeer)(nil)
	_ Transport = (*TCPTransport)(nil)
)

type TCPOptions struct {
	ListenAddr string
}

type TCPPeer struct {
	conn     net.Conn
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

func (p *TCPPeer) Read(b []byte) (int, error) {
	return p.conn.Read(b)
}

func (p *TCPPeer) Write(b []byte) (int, error) {
	return p.conn.Write(b)
}

func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransport struct {
	Options TCPOptions

	lis        net.Listener
	handshaker Handshaker
	handler    Handler

	isDone atomic.Bool
	close  chan struct{}

	mux   sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPOptions) (*TCPTransport, error) {
	t := &TCPTransport{
		Options:    opts,
		lis:        nil,
		handshaker: NopHandshaker,
		handler:    NopHandler,
		isDone:     atomic.Bool{},
		close:      make(chan struct{}, 1),
		peers:      make(map[net.Addr]Peer),
	}
	return t, t.initListener()
}

func (t *TCPTransport) Addr() net.Addr {
	return t.lis.Addr()
}

func (t *TCPTransport) SetHandshaker(h Handshaker) {
	if h == nil {
		return
	}
	t.handshaker = h
}

func (t *TCPTransport) SetHandler(h Handler) {
	if h == nil {
		return
	}
	t.handler = h
}

func (t *TCPTransport) ListenAndAccept() error {
	defer t.lis.Close()

	for !t.isDone.Load() {
		conn, err := t.lis.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			continue
		}

		go t.handleConn(conn)
	}

	t.close <- struct{}{}

	return nil
}

func (t *TCPTransport) Close() error {
	t.isDone.Store(true)

	var errs error
	if t.lis != nil {
		errs = errors.Join(errs, t.lis.Close())
	}

	t.mux.RLock()
	peers := make([]Peer, 0, len(t.peers))
	for _, peer := range t.peers {
		peers = append(peers, peer)
	}
	t.peers = make(map[net.Addr]Peer)
	t.mux.RUnlock()

	for _, peer := range peers {
		errs = errors.Join(errs, peer.Close())
	}

	<-t.close
	return errs
}

func (t *TCPTransport) initListener() (err error) {
	t.lis, err = net.Listen("tcp", t.Options.ListenAddr)
	return
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, false)

	if err := t.handshaker.Handshake(context.TODO(), peer); err != nil {
		return
	}

	t.mux.Lock()
	t.peers[peer.conn.RemoteAddr()] = peer
	t.mux.Unlock()

	t.handler.Handle(context.TODO(), peer)

	t.mux.Lock()
	peer.Close()
	delete(t.peers, peer.conn.RemoteAddr())
	t.mux.Unlock()
}
