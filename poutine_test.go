package poutine

import (
	"context"
	"testing"

	"github.com/calumari/jwalk"
	"github.com/calumari/poutine/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDriver struct{ mock.Mock }

var _ database.Driver = (*mockDriver)(nil)

func (m *mockDriver) Seed(ctx context.Context, root jwalk.Document) (jwalk.Document, error) {
	args := m.Called(ctx, root)
	return args.Get(0).(jwalk.Document), args.Error(1)
}
func (m *mockDriver) Snapshot(ctx context.Context) (jwalk.Document, error) {
	args := m.Called(ctx)
	return args.Get(0).(jwalk.Document), args.Error(1)
}
func (m *mockDriver) Teardown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type mockRegistrarDriver struct{ mockDriver }

var _ Registrar = (*mockRegistrarDriver)(nil)

func (m *mockRegistrarDriver) RegisterTypes(reg *jwalk.Registry) error {
	args := m.Called(reg)
	return args.Error(0)
}

func docKV(k string, v any) jwalk.Document { return jwalk.Document{{Key: k, Value: v}} }

func TestPoutine_Seed(t *testing.T) {
	t.Run("seed success returns document", func(t *testing.T) {
		md := &mockDriver{}
		root := docKV("root", 0)
		want := docKV("seed", 1)
		md.On("Seed", mock.Anything, root).Return(want, nil).Once()
		p := New(md)
		got, err := p.Seed(t.Context(), root)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		md.AssertExpectations(t)
	})

	t.Run("seed driver error returns error", func(t *testing.T) {
		md := &mockDriver{}
		root := docKV("root", 0)
		md.On("Seed", mock.Anything, root).Return(jwalk.Document(nil), assert.AnError).Once()
		p := New(md)
		got, err := p.Seed(t.Context(), root)
		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Equal(t, assert.AnError, err)
		md.AssertExpectations(t)
	})
}

func TestPoutine_Snapshot(t *testing.T) {
	t.Run("snapshot success returns document", func(t *testing.T) {
		md := &mockDriver{}
		want := docKV("snap", 2)
		md.On("Snapshot", mock.Anything).Return(want, nil).Once()
		p := New(md)
		got, err := p.Snapshot(t.Context())
		require.NoError(t, err)
		assert.Equal(t, want, got)
		md.AssertExpectations(t)
	})

	t.Run("snapshot driver error returns error", func(t *testing.T) {
		md := &mockDriver{}
		md.On("Snapshot", mock.Anything).Return(jwalk.Document(nil), assert.AnError).Once()
		p := New(md)
		got, err := p.Snapshot(t.Context())
		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Equal(t, assert.AnError, err)
		md.AssertExpectations(t)
	})
}

func TestPoutine_Teardown(t *testing.T) {
	t.Run("teardown success returns nil", func(t *testing.T) {
		md := &mockDriver{}
		md.On("Teardown", mock.Anything).Return(nil).Once()
		p := New(md)
		err := p.Teardown(t.Context())
		require.NoError(t, err)
		md.AssertExpectations(t)
	})

	t.Run("teardown driver error returns error", func(t *testing.T) {
		md := &mockDriver{}
		md.On("Teardown", mock.Anything).Return(assert.AnError).Once()
		p := New(md)
		err := p.Teardown(t.Context())
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
		md.AssertExpectations(t)
	})
}

func TestPoutine_RegisterTypes(t *testing.T) {
	t.Run("bind registrar driver success invokes register", func(t *testing.T) {
		md := &mockRegistrarDriver{}
		reg, err := jwalk.NewRegistry()
		require.NoError(t, err)
		md.On("RegisterTypes", reg).Return(nil).Once()
		p := New(md)
		err = p.RegisterTypes(reg)
		require.NoError(t, err)
		md.AssertExpectations(t)
	})

	t.Run("bind registrar driver error returns error", func(t *testing.T) {
		md := &mockRegistrarDriver{}
		reg, err := jwalk.NewRegistry()
		require.NoError(t, err)
		md.On("RegisterTypes", reg).Return(assert.AnError).Once()
		p := New(md)
		gotErr := p.RegisterTypes(reg)
		assert.Error(t, gotErr)
		assert.Equal(t, assert.AnError, gotErr)
		md.AssertExpectations(t)
	})

	t.Run("bind non-registrar driver no-op succeeds", func(t *testing.T) {
		md := &mockDriver{}
		p := New(md)
		reg, err := jwalk.NewRegistry()
		require.NoError(t, err)
		// no expectation set for RegisterTypes (not implemented)
		err = p.RegisterTypes(reg)
		require.NoError(t, err)
		md.AssertExpectations(t)
	})
}
