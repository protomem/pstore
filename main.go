package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/protomem/pstore/internal/p2p"
)

func main() {
	log.Println("INFO: pstore version 0.1.0")

	transport, err := p2p.NewTCPTransport(p2p.TCPOptions{
		ListenAddr: ":1337",
	})
	if err != nil {
		log.Panicf("ERROR: new tcp transport: %v", err)
	}

	transport.SetHandshaker(p2p.HandshakerFunc(func(_ context.Context, p p2p.Peer) error {
		log.Printf("DEBUG: success handshake with %s", p.RemoteAddr())
		return nil
	}))

	reader := p2p.NewPacketReader()
	transport.SetHandler(p2p.NewPacketHandler(reader.Handle))

	go func() {
		for p := range reader.Read() {
			log.Printf("DEBUG: read packet from %s with payload: %s", p.From, p.Payload)
		}
	}()

	closeErr := make(chan error)
	go func() {
		<-quit()
		closeErr <- transport.Close()
	}()

	log.Printf("INFO: listen on addr %s", transport.Addr())
	defer log.Printf("INFO: close tcp transport")

	if err := transport.ListenAndAccept(); err != nil {
		log.Panicf("ERROR: listen: %v", err)
	}

	if err := <-closeErr; err != nil {
		log.Panicf("ERROR: close: %v", err)
	}
}

func quit() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
