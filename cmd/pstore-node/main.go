package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/protomem/pstore/internal/gopeer"
)

var _addr = flag.String("addr", ":1337", "listen address")

func init() {
	flag.Parse()
}

func main() {
	tcpClient := gopeer.NewTCPClient()
	tcpLis, err := gopeer.NewTCPListener(*_addr)
	if err != nil {
		log.Printf("ERROR: new tcp listener: %v", err)
		panic(err)
	}

	handshake := gopeer.HandshakerFunc(func(peer gopeer.Peer) error {
		log.Printf("DEBUG: success handshake with %s", peer.RemoteAddr())
		return nil
	})

	node, err := gopeer.NewNode(tcpLis, tcpClient, handshake)
	if err != nil {
		log.Printf("ERROR: new node: %v", err)
		panic(err)
	}

	go func() {
		for p := range node.Read() {
			log.Printf("INFO: read packet from %s: %s", p.Addr, p.Payload)
		}
	}()

	log.Printf("INFO: start server on addr %s", node.Addr())
	defer log.Printf("INFO: stop server")

	<-quit()

	if err := node.Close(); err != nil {
		log.Printf("ERROR: close node: %v", err)
	}
}

func quit() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
