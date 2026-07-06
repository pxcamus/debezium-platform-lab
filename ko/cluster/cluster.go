package cluster

import "context"

type Cluster interface {
	Recreate(ctx context.Context) error
	Delete(ctx context.Context) error
	UseContext(ctx context.Context) error
}
