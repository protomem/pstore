package pstore

import (
	"context"
	"errors"
	"log"
	"net"

	"github.com/protomem/pstore/internal/blobstore"
	"github.com/protomem/pstore/internal/p2p"
)

type FileServerOptions struct{}

type FileServer struct {
	Options FileServerOptions

	store     blobstore.Storage
	transport p2p.Transport

	preader *p2p.PacketReader
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
	}
}

func (s *FileServer) Addr() net.Addr {
	return s.transport.Addr()
}

func (s *FileServer) Proccess() {
	for p := range s.preader.Read() {
		s.handlerPacket(p)
	}
}

func (s *FileServer) Start() error {
	return s.transport.ListenAndAccept()
}

func (s *FileServer) Shutdown(ctx context.Context) error {
	var errs error
	errs = errors.Join(errs, s.transport.Close())
	errs = errors.Join(errs, s.store.Close(ctx))
	return errs
}

func (s *FileServer) handlerPacket(p p2p.Packet) {
	log.Printf("DEBUG: read packet from %s with payload: %s", p.From, p.Payload)
}
