package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/protomem/pstore"
	"github.com/protomem/pstore/internal/blobstore"
	"github.com/protomem/pstore/internal/p2p"
)

func main() {
	ctx := context.Background()
	log.Println("INFO: pstore version 0.1.0")

	transport, err := p2p.NewTCPTransport(p2p.TCPOptions{
		ListenAddr: ":1337",
	})
	if err != nil {
		log.Panicf("ERROR: new tcp transport: %v", err)
	}

	store, err := blobstore.NewFS(blobstore.FSOptions{
		Path: ".database",
	})
	if err != nil {
		log.Panicf("ERROR: new file system storage: %v", err)
	}

	server := pstore.NewFileServer(store, transport, pstore.FileServerOptions{})
	go server.Proccess()

	closeErr := make(chan error)
	go func() {
		<-quit()

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		closeErr <- server.Shutdown(ctx)
	}()

	log.Printf("INFO: start server on addr %s", server.Addr())
	defer log.Printf("INFO: stop server")

	if err := server.Start(); err != nil {
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
