package gopeer

import (
	"net"
)

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

func (t *TCPPeer) RemoteAddr() string {
	return t.conn.RemoteAddr().String()
}

func (t *TCPPeer) Read(p []byte) (int, error) {
	return t.conn.Read(p)
}

func (t *TCPPeer) Write(p []byte) (int, error) {
	return t.conn.Write(p)
}

func (t *TCPPeer) Close() error {
	return t.conn.Close()
}

type TCPClient struct{}

func NewTCPClient() *TCPClient {
	return &TCPClient{}
}

func (t *TCPClient) Dial(addr string) (Peer, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return NewTCPPeer(conn, true), nil
}

type TCPListener struct {
	lis net.Listener
}

func NewTCPListener(addr string) (*TCPListener, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &TCPListener{lis: lis}, nil
}

func (t *TCPListener) Addr() string {
	return t.lis.Addr().String()
}

func (t *TCPListener) Accept() (Peer, error) {
	conn, err := t.lis.Accept()
	if err != nil {
		return nil, err
	}

	return NewTCPPeer(conn, false), nil
}

func (t *TCPListener) Close() error {
	return t.lis.Close()
}
