package exp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbsent(t *testing.T) {
	t.Run("absent pattern is not present", func(t *testing.T) {
		p := Absent[int]()
		got := p.IsPresent()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("absent pattern is not explicit", func(t *testing.T) {
		p := Absent[int]()
		got := p.IsExplicit()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("absent pattern is not wildcard", func(t *testing.T) {
		p := Absent[int]()
		got := p.IsWildcard()
		want := false
		assert.Equal(t, want, got)
	})
}

func TestAny(t *testing.T) {
	t.Run("any pattern is present", func(t *testing.T) {
		p := Any(0)
		got := p.IsPresent()
		want := true
		assert.Equal(t, want, got)
	})

	t.Run("any pattern is not explicit", func(t *testing.T) {
		p := Any(0)
		got := p.IsExplicit()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("any pattern is wildcard", func(t *testing.T) {
		p := Any(0)
		got := p.IsWildcard()
		want := true
		assert.Equal(t, want, got)
	})
}

func TestValue(t *testing.T) {
	t.Run("value pattern is present", func(t *testing.T) {
		p := Value(42)
		got := p.IsPresent()
		want := true
		assert.Equal(t, want, got)
	})

	t.Run("value pattern is explicit", func(t *testing.T) {
		p := Value(42)
		got := p.IsExplicit()
		want := true
		assert.Equal(t, want, got)
	})

	t.Run("value pattern is not wildcard", func(t *testing.T) {
		p := Value(42)
		got := p.IsWildcard()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("unwrap value returns correct value", func(t *testing.T) {
		p := Value(42)
		got := p.UnwrapValue()
		want := 42
		assert.Equal(t, want, got)
	})
}

func TestPattern_IsPresent(t *testing.T) {
	t.Run("absent pattern is not present", func(t *testing.T) {
		p := Absent[int]()
		got := p.IsPresent()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("any pattern is present", func(t *testing.T) {
		p := Any(0)
		got := p.IsPresent()
		want := true
		assert.Equal(t, want, got)
	})

	t.Run("value pattern is present", func(t *testing.T) {
		p := Value(42)
		got := p.IsPresent()
		want := true
		assert.Equal(t, want, got)
	})
}

func TestPattern_IsExplicit(t *testing.T) {
	t.Run("absent pattern is not explicit", func(t *testing.T) {
		p := Absent[int]()
		got := p.IsExplicit()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("any pattern is not explicit", func(t *testing.T) {
		p := Any(0)
		got := p.IsExplicit()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("value pattern is explicit", func(t *testing.T) {
		p := Value(42)
		got := p.IsExplicit()
		want := true
		assert.Equal(t, want, got)
	})
}

func TestPattern_IsWildcard(t *testing.T) {
	t.Run("absent pattern is not wildcard", func(t *testing.T) {
		p := Absent[int]()
		got := p.IsWildcard()
		want := false
		assert.Equal(t, want, got)
	})

	t.Run("any pattern is wildcard", func(t *testing.T) {
		p := Any(0)
		got := p.IsWildcard()
		want := true
		assert.Equal(t, want, got)
	})

	t.Run("value pattern is not wildcard", func(t *testing.T) {
		p := Value(42)
		got := p.IsWildcard()
		want := false
		assert.Equal(t, want, got)
	})
}

func TestPattern_UnwrapValue(t *testing.T) {
	t.Run("absent pattern returns zero value", func(t *testing.T) {
		p := Absent[int]()
		got := p.UnwrapValue()
		want := 0
		assert.Equal(t, want, got)
	})

	t.Run("any pattern returns placeholder value", func(t *testing.T) {
		p := Any(123)
		got := p.UnwrapValue()
		want := 123
		assert.Equal(t, want, got)
	})

	t.Run("value pattern returns explicit value", func(t *testing.T) {
		p := Value(42)
		got := p.UnwrapValue()
		want := 42
		assert.Equal(t, want, got)
	})
}

func TestPattern_Value(t *testing.T) {
	t.Run("absent pattern returns zero value", func(t *testing.T) {
		p := Absent[int]()
		got := p.Value()
		want := 0
		assert.Equal(t, want, got)
	})

	t.Run("any pattern returns placeholder value", func(t *testing.T) {
		p := Any(123)
		got := p.Value()
		want := 123
		assert.Equal(t, want, got)
	})

	t.Run("value pattern returns explicit value", func(t *testing.T) {
		p := Value(42)
		got := p.Value()
		want := 42
		assert.Equal(t, want, got)
	})
}

// TODO: add tests for Pattern.Test once testequals.RuleContext can be mocked
