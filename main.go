package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
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

	transport.SetHandler(p2p.HandlerFunc(func(_ context.Context, p p2p.Peer) {
		log.Printf("INFO: new incoming tcp conn %s", p.RemoteAddr())
		defer log.Printf("INFO: close tcp conn %s", p.RemoteAddr())

		r := bufio.NewReader(p)
		w := bufio.NewWriter(p)
		s := bufio.NewScanner(r)

		for s.Scan() {
			if err := s.Err(); err != nil {
				log.Printf("ERROR: read from conn %s", p.RemoteAddr())
				break
			}

			req := s.Text()
			res := strings.ToUpper(req)

			w.WriteString(res + "\r\n")
			if err := w.Flush(); err != nil {
				log.Printf("ERROR: write to conn %s", p.RemoteAddr())
			}
		}
	}))

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
