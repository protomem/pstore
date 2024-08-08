package pstore

import (
	"context"
	"errors"
	"log"
	"net"

	"github.com/protomem/pstore/internal/blobstore"
	"github.com/protomem/pstore/internal/p2p"
)

type FileServerOptions struct {
	Nodes []string
}

type FileServer struct {
	Options FileServerOptions

	store     blobstore.Storage
	transport p2p.Transport

	preader *p2p.PacketReader

	quit chan struct{}
}

func NewFileServer(store blobstore.Storage, transport p2p.Transport, opts FileServerOptions) *FileServer {
	transport.SetHandshaker(PingPongHandshaker)
	transport.SetErrorHandler(func(err error) {
		if err == nil {
			return
		}
		log.Printf("ERROR: file server transport: %v", err)
	})

	reader := p2p.NewPacketReader()
	transport.SetHandler(p2p.NewPacketHandler(reader.Handle))

	return &FileServer{
		Options:   opts,
		store:     store,
		transport: transport,
		preader:   reader,
		quit:      make(chan struct{}, 1),
	}
}

func (s *FileServer) Addr() net.Addr {
	return s.transport.Addr()
}

func (s *FileServer) Process() {
	for p := range s.preader.Read() {
		go s.handlePacket(p)
	}
}

func (s *FileServer) Start() error {
	if err := s.bootstrapNodes(); err != nil {
		return err
	}

	return s.transport.ListenAndAccept()
}

func (s *FileServer) Close(ctx context.Context) error {
	s.quit <- struct{}{}
	close(s.quit)

	var errs error
	errs = errors.Join(errs, s.transport.Close())
	errs = errors.Join(errs, s.store.Close(ctx))

	return errs
}

func (s *FileServer) bootstrapNodes() error {
	var errs error
	for _, addr := range s.Options.Nodes {
		if err := s.transport.Dial(addr); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func (s *FileServer) handlePacket(p p2p.Packet) {
	if DetectPing(p.Payload) {
		log.Printf("DEBUG: ping-pong with %s", p.From)

		peer, _ := s.transport.AcquirePeer(p.From)
		defer s.transport.ReleasePeer(peer)

		_, err := peer.Write([]byte("PONG\r\n"))
		if err != nil {
			log.Printf("ERROR: write ping-pong response: %v", err)
		}
	} else {
		log.Printf("DEBUG: read packet from %s with payload: %s", p.From, p.Payload)
	}
}
