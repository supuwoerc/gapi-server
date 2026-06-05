package server

import "context"

type IServerHook interface {
	OnReady(ctx context.Context) error
}
