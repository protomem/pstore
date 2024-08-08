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
	transport.SetHandshaker(p2p.HandshakerFunc(func(_ context.Context, p p2p.Peer) error {
		log.Printf("DEBUG: success handshake with %s", p.RemoteAddr())
		return nil
	}))

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
		s.handlerPacket(p)
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

func (s *FileServer) handlerPacket(p p2p.Packet) {
	log.Printf("DEBUG: read packet from %s with payload: %s", p.From, p.Payload)
}
