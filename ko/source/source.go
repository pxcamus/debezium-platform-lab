package source

import "context"

type Source interface {
	Wait(ctx context.Context) error
	Setup(ctx context.Context) error
	Populate(ctx context.Context) error
	Reset(ctx context.Context) error
	Close(ctx context.Context) error
}
