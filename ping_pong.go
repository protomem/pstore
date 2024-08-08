package pstore

import (
	"bytes"
	"context"
	"errors"
	"log"

	"github.com/protomem/pstore/internal/p2p"
)

var PingPongHandshaker = p2p.HandshakerFunc(func(ctx context.Context, p p2p.Peer) error {
	var err error

	_, err = p.Write([]byte("PING\r\n"))
	if err != nil {
		return err
	}

	resp := make([]byte, 1024)
	read, err := p.Read(resp)
	if err != nil {
		return err
	}

	if !DetectPong(resp[:read]) {
		return errors.New("invalid handshake")
	}

	log.Printf("DEBUG: success handshake with %s", p.RemoteAddr())

	return nil
})

func DetectPing(data []byte) bool {
	return bytes.Contains(data, []byte("PING"))
}

func DetectPong(data []byte) bool {
	return bytes.Contains(data, []byte("PONG"))
}
