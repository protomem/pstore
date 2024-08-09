package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/protomem/pstore"
)

var (
	_addr  = flag.String("addr", ":1337", "listen address")
	_db    = flag.String("db", ".database", "db path")
	_nodes = flag.String("nodes", "", "nodes to connect to")
)

func init() {
	flag.Parse()
}

func main() {
	server := pstore.NewFileServer(pstore.Options{
		Path:  *_db,
		Addr:  *_addr,
		Nodes: parseNodes(*_nodes),
	})

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("ERROR: start server: %v", err)
		}
	}()

	log.Printf("INFO: start server on addr %s", server.Addr())
	defer log.Printf("INFO: stop server")

	<-quit()

	if err := server.Close(); err != nil {
		log.Printf("ERROR: close node: %v", err)
	}
}

func quit() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}

func parseNodes(nodes string) []string {
	nodes = strings.TrimSpace(nodes)
	if nodes == "" {
		return nil
	}
	return strings.Split(nodes, ",")
}
