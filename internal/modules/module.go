package modules

import "context"

type Module interface {
	Start(ctx context.Context) error
	Stop() error
}
