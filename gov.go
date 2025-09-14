package gov

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

var nullValue = []byte{'n', 'u', 'l', 'l'}

type State uint8

const (
	StateNone State = iota
	StateNil
	StateFilled
)

type Value[T any] struct {
	state State
	value T
}

func (v Value[T]) State() State {
	return v.state
}

func (v Value[T]) Present() bool {
	return v.state != StateNone
}

func (v Value[T]) Filled() bool {
	return v.state == StateFilled
}

func (v Value[T]) Nil() bool {
	return v.state == StateNil
}

func (v Value[T]) IsZero() bool {
	return !v.Present()
}

func (v Value[T]) Value() (driver.Value, error) {
	if v.Nil() {
		return nil, nil
	}
	if valuer, ok := any(v.value).(driver.Valuer); ok {
		return valuer.Value()
	}
	return driver.DefaultParameterConverter.ConvertValue(any(v.value))
}

func (v Value[T]) MarshalJSON() ([]byte, error) {
	if !v.Filled() {
		return nullValue, nil
	}
	return json.Marshal(v.value)
}

func (v *Value[T]) UnmarshalJSON(data []byte) (err error) {
	if bytes.Equal(data, nullValue) {
		v.state = StateNil
		return nil
	}
	if err = json.Unmarshal(data, &v.value); err != nil {
		return err
	}
	v.state = StateFilled
	return
}

func (v *Value[T]) Scan(src any) error {
	if src == nil {
		v.state = StateNil
		return nil
	}
	var s sql.Null[T]
	if err := s.Scan(src); err != nil {
		return err
	}
	if !s.Valid {
		v.state = StateNil
	} else {
		v.state = StateFilled
		v.value = s.V
	}
	return nil
}

func (v Value[T]) Get() (_ T, _ bool) {
	if v.Filled() {
		return v.value, true
	}
	return
}

func (v Value[T]) GetOrZero() (_ T) {
	if v.Filled() {
		return v.value
	}
	return
}

func (v Value[T]) GetOr(fallback T) T {
	result, ok := v.Get()
	if !ok {
		return fallback
	}
	return result
}

func (v Value[T]) AsPointer() *T {
	if v.Filled() {
		return &v.value
	}
	return nil
}

func None[T any]() Value[T] {
	return Value[T]{}
}

func Nil[T any]() Value[T] {
	return Value[T]{
		state: StateNil,
	}
}

func Filled[T any](value T) Value[T] {
	return Value[T]{
		state: StateFilled,
		value: value,
	}
}

func When[T any](value T, ok bool, noneOnFalse ...bool) Value[T] {
	if ok {
		return Filled(value)
	}
	return Pointer[T](nil, noneOnFalse...)
}

func Pointer[T any](ptr *T, noneOnNil ...bool) Value[T] {
	nof := false
	if len(noneOnNil) > 0 {
		nof = noneOnNil[0]
	}
	if ptr == nil {
		state := StateNil
		if nof {
			state = StateNone
		}
		return Value[T]{
			state: state,
		}
	}
	return Filled(*ptr)
}
