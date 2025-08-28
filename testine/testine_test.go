package testine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/calumari/jwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockPoutine struct{ mock.Mock }

func (m *mockPoutine) Seed(ctx context.Context, root jwalk.Document) (jwalk.Document, error) {
	args := m.Called(ctx, root)
	if doc, ok := args.Get(0).(jwalk.Document); ok {
		return doc, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPoutine) Snapshot(ctx context.Context) (jwalk.Document, error) {
	args := m.Called(ctx)
	if doc, ok := args.Get(0).(jwalk.Document); ok {
		return doc, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPoutine) Teardown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockPoutine) RegisterTypes(reg *jwalk.Registry) error { // optional registrar
	args := m.Called(reg)
	return args.Error(0)
}

type mockTester struct{ mock.Mock }

func (m *mockTester) Test(expected, actual any) error {
	args := m.Called(expected, actual)
	return args.Error(0)
}

type mockTestingT struct{ mock.Mock }

func (ft *mockTestingT) Context() context.Context {
	return context.Background()
}

func (ft *mockTestingT) Cleanup(f func()) {
	ft.Called(f)
	f()
}

func (ft *mockTestingT) Fatalf(format string, args ...any) {
}

func (ft *mockTestingT) Helper() {
}

func docKV(k string, v any) jwalk.Document { return jwalk.Document{{Key: k, Value: v}} }

func TestNew(t *testing.T) {
	t.Run("default options success creates registry", func(t *testing.T) {
		mp := &mockPoutine{}
		// no RegisterTypes expectation: treat as non-registrar by shadowing method call via type assertion block
		// Provide expectation returning nil to satisfy call made by New when driver implements Registrar.
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		pt, err := New(mp)
		require.NoError(t, err)
		assert.NotNil(t, pt.registry)
		assert.NotNil(t, pt.tester)
	})

	t.Run("provided registry success uses provided registry", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		r, _ := jwalk.NewRegistry()
		pt, err := New(mp, WithRegistry(r))
		require.NoError(t, err)
		assert.Equal(t, r, pt.registry)
	})

	t.Run("custom tester success uses provided tester", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Snapshot", mock.Anything).Return(jwalk.Document(nil), nil).Once()
		mt := &mockTester{}
		mt.On("Test", mock.Anything, mock.Anything).Return(nil).Once()
		pt, err := New(mp, WithTester(mt))
		require.NoError(t, err)
		pt.Assert(&mockTestingT{}, nil)
		mt.AssertExpectations(t)
	})

	t.Run("registrar poutine success registers directives", func(t *testing.T) {
		mp := &mockPoutine{}
		r, _ := jwalk.NewRegistry()
		mp.On("RegisterTypes", r).Return(nil).Once()
		_, err := New(mp, WithRegistry(r))
		require.NoError(t, err)
		mp.AssertExpectations(t)
	})
}

func TestT_Seed(t *testing.T) {
	root := docKV("a", 1)

	t.Run("seed success returns snapshot", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Seed", mock.Anything, root).Return(root, nil).Once()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		snap := pt.Seed(ft, root)
		assert.Equal(t, root, snap.expected)
		mp.AssertExpectations(t)
	})

	t.Run("seed nil root returns snapshot", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Seed", mock.Anything, jwalk.Document(nil)).Return(jwalk.Document(nil), nil).Once()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		snap := pt.Seed(ft, nil)
		assert.Nil(t, snap.expected)
		mp.AssertExpectations(t)
	})

	t.Run("seed underlying error returns fatal", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Seed", mock.Anything, root).Return(jwalk.Document(nil), assert.AnError).Once()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		_ = pt.Seed(ft, root)
		mp.AssertExpectations(t)
	})
}

func TestT_Assert(t *testing.T) {
	want := docKV("a", 1)

	t.Run("assert success matches documents", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Snapshot", mock.Anything).Return(want, nil).Once()
		mt := &mockTester{}
		mt.On("Test", want, want).Return(nil).Once()
		pt, err := New(mp, WithTester(mt))
		require.NoError(t, err)
		ft := &mockTestingT{}
		pt.Assert(ft, want)
		mp.AssertExpectations(t)
		mt.AssertExpectations(t)
	})

	t.Run("assert tester mismatch returns fatal", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Snapshot", mock.Anything).Return(want, nil).Once()
		mt := &mockTester{}
		mt.On("Test", mock.Anything, mock.Anything).Return(assert.AnError).Once()
		pt, err := New(mp, WithTester(mt))
		require.NoError(t, err)
		ft := &mockTestingT{}
		pt.Assert(ft, docKV("a", 2))
		mp.AssertExpectations(t)
		mt.AssertExpectations(t)
	})

	t.Run("assert snapshot error returns fatal", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Snapshot", mock.Anything).Return(nil, assert.AnError).Once()
		mt := &mockTester{}
		// Snapshot fails before tester invoked; allow zero calls without panic.
		mt.On("Test", mock.Anything, mock.Anything).Return(nil).Maybe()
		pt, err := New(mp, WithTester(mt))
		require.NoError(t, err)
		ft := &mockTestingT{}
		pt.Assert(ft, want)
		mp.AssertExpectations(t)
	})
}

func TestT_Cleanup(t *testing.T) {
	t.Run("cleanup success calls teardown", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Teardown", mock.Anything).Return(nil).Once()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		ft.On("Cleanup", mock.Anything).Once()
		pt.Cleanup(ft)
		mp.AssertExpectations(t)
	})

	t.Run("cleanup teardown error returns fatal", func(t *testing.T) {
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Teardown", mock.Anything).Return(assert.AnError).Once()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		ft.On("Cleanup", mock.Anything).Once()
		pt.Cleanup(ft)
		mp.AssertExpectations(t)
	})
}

func TestT_LoadJSON(t *testing.T) {
	t.Run("load json success returns document", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "doc.json")
		err := os.WriteFile(file, []byte(`{"key":"val"}`), 0o600)
		require.NoError(t, err)
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		got := pt.LoadJSON(ft, file)
		require.Equal(t, 1, len(got))
		assert.Equal(t, "key", got[0].Key)
		assert.Equal(t, "val", got[0].Value)
	})

	t.Run("load json invalid returns fatal", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "bad.json")
		err := os.WriteFile(file, []byte(`not-json`), 0o600)
		require.NoError(t, err)
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		_ = pt.LoadJSON(ft, file)
	})
}

func TestSnapshot_Assert(t *testing.T) {
	t.Run("snapshot assert success matches expected", func(t *testing.T) {
		want := docKV("a", 1)
		mp := &mockPoutine{}
		mp.On("RegisterTypes", mock.Anything).Return(nil).Maybe()
		mp.On("Snapshot", mock.Anything).Return(want, nil).Once()
		pt, err := New(mp)
		require.NoError(t, err)
		ft := &mockTestingT{}
		snap := &Snapshot{pt: pt, expected: want}
		snap.Assert(ft)
		mp.AssertExpectations(t)
	})
}
