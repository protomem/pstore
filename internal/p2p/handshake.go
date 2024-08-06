package p2p

import (
	"context"
	"errors"
)

var ErrInvalidHandshake = errors.New("invalid handshake")

type Handshaker interface {
	Handshake(context.Context, Peer) error
}

type HandshakerFunc func(context.Context, Peer) error

func (fn HandshakerFunc) Handshake(ctx context.Context, peer Peer) error {
	return fn(ctx, peer)
}

var NopHandshaker = HandshakerFunc(func(_ context.Context, _ Peer) error {
	return nil
})
