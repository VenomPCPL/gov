package gov

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"github.com/segmentio/encoding/json"
)

var (
	jsonNull = []byte(`null`)
)

type State uint8

const (
	StateNotSet State = iota
	StateNull
	StateValue
)

type Value[T any] struct {
	state State
	value T
}

func (v Value[T]) State() State {
	return v.state
}

func (v Value[T]) Value() (driver.Value, error) {
	if v.IsNull() {
		return nil, nil
	}
	if valuer, ok := any(v.value).(driver.Valuer); ok {
		return valuer.Value()
	}
	return driver.DefaultParameterConverter.ConvertValue(any(v.value))
}

func (v *Value[T]) Scan(src any) error {
	if src == nil {
		v.Null()
		return nil
	}
	var s sql.Null[T]
	if err := s.Scan(src); err != nil {
		return err
	}
	if !s.Valid {
		v.Null()
	} else {
		v.Set(s.V)
	}
	return nil
}

func (v Value[T]) IsZero() bool {
	return v.state == StateNotSet
}

func (v Value[T]) GetOrZero() T {
	if v.Valid() {
		return v.value
	}
	var zero T
	return zero
}

func (v *Value[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, jsonNull) {
		v.state = StateNull
		return nil
	}
	v.state = StateValue
	return json.Unmarshal(data, &v.value)
}

func (v Value[T]) MarshalJSON() ([]byte, error) {
	if v.IsNull() || !v.IsSet() {
		return jsonNull, nil
	}
	return json.Marshal(v.value)
}

func (v Value[T]) IsNull() bool {
	return v.state == StateNull
}

func (v Value[T]) IsSet() bool {
	return v.state == StateValue
}

func (v Value[T]) Valid() bool {
	return v.IsNull() || v.IsSet()
}

func (v *Value[T]) Null() {
	v.state = StateNull
}

func (v *Value[T]) Reset() {
	v.state = StateNotSet
}

func (v *Value[T]) Set(value T) {
	v.state = StateValue
	v.value = value
}

func (v *Value[T]) SetPtr(ptr *T) {
	if ptr == nil {
		v.Null()
	} else {
		v.Set(*ptr)
	}
}

func (v Value[T]) MustGet() (r T) {
	r = v.value
	return
}

func (v Value[T]) Get() (_ T, ok bool) {
	if v.state != StateValue {
		return
	}
	return v.value, true
}

func (v Value[T]) GetOr(fallback T) T {
	if !v.Valid() {
		return fallback
	}
	return v.value
}

func EmptyValue[T any]() Value[T] {
	return Value[T]{state: StateNotSet}
}

func FromCondition[T any](cond bool, val T, falseState ...State) Value[T] {
	state := StateNotSet
	if len(falseState) > 0 {
		state = falseState[0]
	}
	if cond {
		return From(val)
	} else {
		return FromPtr[T](nil, state)
	}
}

func FromBuilder[T any](fn func(val Value[T])) Value[T] {
	val := EmptyValue[T]()
	fn(val)
	return val
}

func From[T any](val T) Value[T] {
	return Value[T]{
		state: StateValue,
		value: val,
	}
}

func FromPtr[T any](ptr *T, nilState State) Value[T] {
	if ptr == nil {
		return Value[T]{
			state: nilState,
		}
	} else {
		return From(*ptr)
	}
}
