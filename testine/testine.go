package testine

import (
	"context"

	"github.com/calumari/jwalk"
	"github.com/calumari/poutine"
	"github.com/calumari/testequals"
)

type Tester interface {
	Test(expected, actual any) error
}

type Options struct {
	Tester         Tester
	Registry       *jwalk.Registry
	cacheDocuments bool
}

type Option func(*Options)

func WithTester(t Tester) Option {
	return func(o *Options) { o.Tester = t }
}
func WithRegistry(r *jwalk.Registry) Option {
	return func(o *Options) { o.Registry = r }
}
func WithDocumentCache() Option {
	return func(o *Options) { o.cacheDocuments = true }
}

type Poutine interface {
	Seed(ctx context.Context, root jwalk.Document) (jwalk.Document, error)
	Snapshot(ctx context.Context) (jwalk.Document, error)
	Teardown(ctx context.Context) error
}

type TestingT interface {
	Context() context.Context
	Cleanup(func())
	Fatalf(format string, args ...any)
	Helper()
}

type T struct {
	poutine  Poutine
	tester   Tester
	registry *jwalk.Registry
	loader   *documentLoader
}

func New(p Poutine, opts ...Option) (*T, error) {
	op := &Options{Tester: testequals.New()}
	for _, o := range opts {
		o(op)
	}
	reg := op.Registry
	if reg == nil {
		r, err := jwalk.NewRegistry()
		if err != nil {
			return nil, err
		}
		reg = r
	}
	if regDriver, ok := p.(poutine.Registrar); ok {
		if err := regDriver.RegisterTypes(reg); err != nil {
			return nil, err
		}
	}
	t := &T{
		poutine:  p,
		tester:   op.Tester,
		registry: reg,
	}
	t.loader = newDocumentLoader(reg, op.cacheDocuments)
	return t, nil
}

func (pt *T) Seed(t TestingT, root jwalk.Document) *Snapshot {
	t.Helper()
	actual, err := pt.poutine.Seed(t.Context(), root)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return &Snapshot{pt: pt, expected: actual}
}

func (pt *T) Assert(t TestingT, expected jwalk.Document) {
	t.Helper()
	actual, err := pt.poutine.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if err := pt.tester.Test(expected, actual); err != nil {
		t.Fatalf("assert: %v", err)
	}
}

func (pt *T) Cleanup(t TestingT) {
	t.Helper()
	t.Cleanup(func() {
		// use background context for cleanup as the test context may be done
		// TODO: consider allowing passing a context to Cleanup? If this
		// matters, the caller can call teardown themselves
		if err := pt.poutine.Teardown(context.Background()); err != nil {
			t.Fatalf("teardown: %v", err)
		}
	})
}

func (pt *T) LoadJSON(t TestingT, path string) jwalk.Document {
	t.Helper()
	doc, err := pt.loader.load(path)
	if err != nil {
		t.Fatalf("load json %s: %v", path, err)
	}
	return doc
}

type Snapshot struct {
	pt       *T
	expected jwalk.Document
}

func (s *Snapshot) Assert(t TestingT) {
	t.Helper()
	s.pt.Assert(t, s.expected)
}
