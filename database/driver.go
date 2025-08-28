package database

import (
	"context"

	"github.com/calumari/jwalk"
)

type Driver interface {
	Seed(ctx context.Context, root jwalk.Document) (jwalk.Document, error)
	Snapshot(ctx context.Context) (jwalk.Document, error)
	Teardown(ctx context.Context) error
}
