package gopeer

import (
	"bytes"
	"errors"
	"io"
	"net"
	"slices"
	"sync"
)

type Packet struct {
	Addr    string
	Payload []byte
}

func NewPacket(addr string, payload []byte) *Packet {
	return &Packet{
		Addr:    addr,
		Payload: payload,
	}
}

type Client interface {
	Dial(addr string) (Peer, error)
}

type Listener interface {
	Addr() string
	Accept() (Peer, error)

	io.Closer
}

type Peer interface {
	io.ReadWriteCloser
	RemoteAddr() string
}

type Handshaker interface {
	Handshaker(Peer) error
}

type HandshakerFunc func(Peer) error

func (fn HandshakerFunc) Handshaker(peer Peer) error {
	return fn(peer)
}

var NopHandshaker = HandshakerFunc(func(_ Peer) error { return nil })

type Node struct {
	lis    Listener
	client Client

	handshake Handshaker

	reader chan *Packet

	errsMux sync.Mutex
	errs    []error

	peersMux sync.Mutex
	peers    map[string]Peer
}

func NewNode(lis Listener, client Client, handshake Handshaker) (*Node, error) {
	n := &Node{
		lis:       lis,
		client:    client,
		handshake: handshake,
		reader:    make(chan *Packet, 1024),
		errs:      make([]error, 0),
		peers:     make(map[string]Peer),
	}
	n.listenAndAccept()
	return n, nil
}

func (n *Node) Addr() string {
	return n.lis.Addr()
}

func (n *Node) Close() error {
	var errs error
	errs = errors.Join(n.acquireErrs()...)
	errs = errors.Join(errs, n.lis.Close())

	close(n.reader)

	peers := n.acquirePeers()
	n.clearPeers()

	for _, peer := range peers {
		errs = errors.Join(errs, peer.Close())
	}

	return errs
}

func (n *Node) Read() <-chan *Packet {
	return n.reader
}

func (n *Node) Write(p *Packet, addrs ...string) error {
	if p == nil || len(addrs) == 0 {
		return nil
	}

	peers := n.acquirePeers(addrs...)
	for _, peer := range peers {
		_, err := peer.Write(p.Payload)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) Dial(addr string) error {
	peer, err := n.client.Dial(addr)
	if err != nil {
		return err
	}

	n.appendPeer(peer)
	go n.onPeer(peer)

	return nil
}

func (n *Node) listenAndAccept() {
	go func() {
		for {
			peer, err := n.lis.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				n.pushErr(err)
				continue
			}

			if err := n.handshake.Handshaker(peer); err != nil {
				n.pushErr(err)
				continue
			}

			n.appendPeer(peer)
			go n.onPeer(peer)
		}
	}()
}

func (n *Node) onPeer(peer Peer) {
LOOP:
	for {
		buff := make([]byte, 1024)
		read, err := peer.Read(buff)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break LOOP
			}

			n.pushErr(err)
			continue
		}

		p := NewPacket(peer.RemoteAddr(), bytes.Clone(buff[:read]))
		n.reader <- p
	}

	n.removePeer(peer.RemoteAddr())
	n.pushErr(peer.Close())
}

func (n *Node) pushErr(err error) {
	n.errsMux.Lock()
	defer n.errsMux.Unlock()
	n.errs = append(n.errs, err)
}

func (n *Node) acquireErrs() []error {
	n.errsMux.Lock()
	defer n.errsMux.Unlock()
	return n.errs
}

func (n *Node) appendPeer(peer Peer) {
	n.peersMux.Lock()
	defer n.peersMux.Unlock()
	n.peers[peer.RemoteAddr()] = peer
}

func (n *Node) acquirePeers(addrs ...string) []Peer {
	n.peersMux.Lock()
	defer n.peersMux.Unlock()
	peers := make([]Peer, 0, len(n.peers))
	for _, peer := range n.peers {
		if len(addrs) == 0 || slices.Contains(addrs, peer.RemoteAddr()) {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (n *Node) removePeer(addr string) {
	n.peersMux.Lock()
	defer n.peersMux.Unlock()
	delete(n.peers, addr)
}

func (n *Node) clearPeers() {
	n.peersMux.Lock()
	defer n.peersMux.Unlock()
	n.peers = make(map[string]Peer)
}
