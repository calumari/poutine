package poutine

import (
	"context"

	"github.com/calumari/jwalk"

	"github.com/calumari/poutine/database"
)

type Registrar interface {
	RegisterTypes(*jwalk.Registry) error
}

type Poutine struct {
	driver database.Driver
}

var _ Registrar = (*Poutine)(nil)

func New(driver database.Driver) *Poutine {
	return &Poutine{
		driver: driver,
	}
}

func (p *Poutine) Seed(ctx context.Context, root jwalk.Document) (jwalk.Document, error) {
	return p.driver.Seed(ctx, root)
}

func (p *Poutine) Snapshot(ctx context.Context) (jwalk.Document, error) {
	return p.driver.Snapshot(ctx)
}

func (p *Poutine) Teardown(ctx context.Context) error {
	return p.driver.Teardown(ctx)
}

func (p *Poutine) RegisterTypes(reg *jwalk.Registry) error {
	if r, ok := p.driver.(Registrar); ok {
		return r.RegisterTypes(reg)
	}
	return nil
}
