package p2p

import "context"

type Handler interface {
	Handle(context.Context, Peer)
}

type HandlerFunc func(context.Context, Peer)

func (fn HandlerFunc) Handle(ctx context.Context, peer Peer) {
	fn(ctx, peer)
}

var NopHandler = HandlerFunc(func(ctx context.Context, p Peer) {})
