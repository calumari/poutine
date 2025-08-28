package exp

import (
	"fmt"
	"reflect"

	"github.com/calumari/testequals"
)

// Pattern represents a value that may be explicitly set by the user, or merely
// asserted to exist (wildcard) for its type. It has three logical states:
//  1. absent    : not present (zero value of Pattern) -> !IsPresent()
//  2. any       : present but not explicit (wildcard) -> IsPresent() && !IsExplicit()
//  3. explicit  : present and explicit -> IsPresent() && IsExplicit()
//
// In all present states a stable value is stored (placeholder for Any, user
// value for Value). This allows patterns like {"$oid": true} (Any with
// placeholder) or {"$oid":"abc"} (explicit Value).
type Pattern[T any] struct {
	value    T
	present  bool // true for any or explicit
	explicit bool // true only for explicit
}

var _ testequals.Rule = Pattern[any]{}

func Absent[T any]() Pattern[T] {
	return Pattern[T]{}
}

func Any[T any](placeholder T) Pattern[T] {
	return Pattern[T]{value: placeholder, present: true}
}

func Value[T any](v T) Pattern[T] {
	return Pattern[T]{value: v, present: true, explicit: true}
}

func (p Pattern[T]) IsPresent() bool {
	return p.present
}
func (p Pattern[T]) IsExplicit() bool {
	return p.explicit
}
func (p Pattern[T]) IsWildcard() bool {
	return p.present && !p.explicit
}

func (p Pattern[T]) UnwrapValue() any {
	if p.IsPresent() {
		return p.value
	}
	var zero T
	return zero
}

// Value returns the typed value. For absent patterns, returns the zero value.
func (p Pattern[T]) Value() T {
	return p.value
}

func (p Pattern[T]) String() string {
	if !p.IsPresent() {
		return "absent"
	}
	if p.IsExplicit() {
		return fmt.Sprintf("%v", p.value)
	}
	return fmt.Sprintf("%v (wildcard)", p.value)
}

func (p Pattern[T]) Test(rc *testequals.RuleContext, actual any) error {
	if !p.IsPresent() {
		return rc.Test(nil, actual)
	}
	if p.IsExplicit() {
		return rc.Test(p.value, actual)
	}
	// Wildcard: enforce only type compatibility, not value equality.
	if _, ok := actual.(T); !ok { // fast path
		tv := reflect.TypeOf((*T)(nil)).Elem()
		if actual == nil {
			switch tv.Kind() {
			case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
				return nil // nil is acceptable for wildcard of a nilable type
			default:
				return fmt.Errorf("expected non-nil value of type %v, got <nil>", tv)
			}
		}
		av := reflect.TypeOf(actual)
		if !av.AssignableTo(tv) && !av.ConvertibleTo(tv) {
			return fmt.Errorf("expected type assignable/convertible to %v, got %v", tv, av)
		}
	}
	return nil
}
