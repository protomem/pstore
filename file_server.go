package pstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/protomem/pstore/internal/blobstore"
	"github.com/protomem/pstore/internal/gopeer"
)

type Options struct {
	Path  string
	Addr  string
	Nodes []string
}

type FileServer struct {
	Options Options

	store blobstore.Storage
	node  *gopeer.Node
}

func NewFileServer(options Options) *FileServer {
	s := &FileServer{
		Options: options,
	}
	return s
}

func (s *FileServer) Addr() string {
	return s.Options.Addr
}

func (s *FileServer) Start() error {
	if err := s.initNode(); err != nil {
		return err
	}

	if err := s.initStore(); err != nil {
		return err
	}

	if err := s.connectToNodes(); err != nil {
		return err
	}

	return s.process()
}

func (s *FileServer) Close() error {
	var errs error
	errs = errors.Join(errs, s.store.Close(context.TODO()))
	errs = errors.Join(errs, s.node.Close())
	return errs
}

func (s *FileServer) initNode() error {
	var err error

	client := gopeer.NewTCPClient()
	lis, err := gopeer.NewTCPListener(s.Options.Addr)
	if err != nil {
		return err
	}

	s.node, err = gopeer.NewNode(lis, client, pingPongHandshaker)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileServer) initStore() error {
	var err error

	s.store, err = blobstore.NewFS(blobstore.FSOptions{
		Path: s.Options.Path,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *FileServer) connectToNodes() error {
	for _, addr := range s.Options.Nodes {
		err := s.node.Dial(addr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *FileServer) process() error {
	for p := range s.node.Read() {
		log.Printf("INFO: read packet from %s: %s", p.Addr, p.Payload)

		if bytes.HasPrefix(p.Payload, []byte("PING")) {
			log.Printf("DEBUG: received PING from %s", p.Addr)

			if err := s.node.Write(gopeer.NewPacket("", []byte("PONG\r\n")), p.Addr); err != nil {
				return err
			}

			continue
		}
	}
	return nil
}

var pingPongHandshaker = gopeer.HandshakerFunc(func(peer gopeer.Peer) error {
	_, err := peer.Write([]byte("PONG\r\n"))
	if err != nil {
		return err
	}

	buf := make([]byte, 32)
	read, err := peer.Read(buf)
	if err != nil {
		return err
	}

	if read < 4 || string(buf[:4]) != "PING" {
		log.Printf("DEBUG: failed handshake with %s", peer.RemoteAddr())
		return fmt.Errorf("invalid handshake with %s", peer.RemoteAddr())
	}

	log.Printf("DEBUG: success handshake with %s", peer.RemoteAddr())
	return nil
})
