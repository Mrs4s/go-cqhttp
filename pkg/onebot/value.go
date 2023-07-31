// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package onebot

import (
	"fmt"
	"math"
	"strconv"
	"time"
	"unsafe"
)

// A Value can represent any Go value, but unlike type any,
// it can represent most small values without an allocation.
// The zero Value corresponds to nil.
type Value struct {
	_   [0]func() // disallow ==
	num uint64    // hold number value
	any any       // hold Kind or other value
}

type (
	stringptr *byte // used in Value.any when the Value is a string
	groupptr  *Attr // used in Value.any when the Value is a []Attr
)

//go:generate stringer -type=Kind -trimprefix=Kind

// Kind is the kind of Value.
type Kind int

// Kind
const (
	KindAny Kind = iota
	KindBool
	KindDuration
	KindFloat64
	KindInt64
	KindString
	KindTime
	KindUint64
	KindGroup
)

// Unexported version of Kind, just so we can store Kinds in Values.
// (No user-provided value has this type.)
type kind Kind

// Kind returns v's Kind.
func (v Value) Kind() Kind {
	switch x := v.any.(type) {
	case Kind:
		return x
	case stringptr:
		return KindString
	case timeLocation:
		return KindTime
	case groupptr:
		return KindGroup
	case kind: // a kind is just a wrapper for a Kind
		return KindAny
	default:
		return KindAny
	}
}

//////////////// Constructors

// StringValue returns a new Value for a string.
func StringValue(value string) Value {
	return Value{num: uint64(len(value)), any: stringptr(unsafe.StringData(value))}
}

// IntValue returns a Value for an int.
func IntValue(v int) Value {
	return Int64Value(int64(v))
}

// Int64Value returns a Value for an int64.
func Int64Value(v int64) Value {
	return Value{num: uint64(v), any: KindInt64}
}

// Uint64Value returns a Value for a uint64.
func Uint64Value(v uint64) Value {
	return Value{num: v, any: KindUint64}
}

// Float64Value returns a Value for a floating-point number.
func Float64Value(v float64) Value {
	return Value{num: math.Float64bits(v), any: KindFloat64}
}

// BoolValue returns a Value for a bool.
func BoolValue(v bool) Value {
	u := uint64(0)
	if v {
		u = 1
	}
	return Value{num: u, any: KindBool}
}

// Unexported version of *time.Location, just so we can store *time.Locations in
// Values. (No user-provided value has this type.)
type timeLocation *time.Location

// TimeValue returns a Value for a time.Time.
// It discards the monotonic portion.
func TimeValue(v time.Time) Value {
	if v.IsZero() {
		// UnixNano on the zero time is undefined, so represent the zero time
		// with a nil *time.Location instead. time.Time.Location method never
		// returns nil, so a Value with any == timeLocation(nil) cannot be
		// mistaken for any other Value, time.Time or otherwise.
		return Value{any: timeLocation(nil)}
	}
	return Value{num: uint64(v.UnixNano()), any: timeLocation(v.Location())}
}

// DurationValue returns a Value for a time.Duration.
func DurationValue(v time.Duration) Value {
	return Value{num: uint64(v.Nanoseconds()), any: KindDuration}
}

// GroupValue returns a new Value for a list of Attrs.
// The caller must not subsequently mutate the argument slice.
func GroupValue(as ...Attr) Value {
	return Value{num: uint64(len(as)), any: groupptr(unsafe.SliceData(as))}
}

// AnyValue returns a Value for the supplied value.
//
// If the supplied value is of type Value, it is returned
// unmodified.
//
// Given a value of one of Go's predeclared string, bool, or
// (non-complex) numeric types, AnyValue returns a Value of kind
// String, Bool, Uint64, Int64, or Float64. The width of the
// original numeric type is not preserved.
//
// Given a time.Time or time.Duration value, AnyValue returns a Value of kind
// KindTime or KindDuration. The monotonic time is not preserved.
//
// For nil, or values of all other types, including named types whose
// underlying type is numeric, AnyValue returns a value of kind KindAny.
func AnyValue(v any) Value {
	switch v := v.(type) {
	case string:
		return StringValue(v)
	case int:
		return Int64Value(int64(v))
	case uint:
		return Uint64Value(uint64(v))
	case int64:
		return Int64Value(v)
	case uint64:
		return Uint64Value(v)
	case bool:
		return BoolValue(v)
	case time.Duration:
		return DurationValue(v)
	case time.Time:
		return TimeValue(v)
	case uint8:
		return Uint64Value(uint64(v))
	case uint16:
		return Uint64Value(uint64(v))
	case uint32:
		return Uint64Value(uint64(v))
	case uintptr:
		return Uint64Value(uint64(v))
	case int8:
		return Int64Value(int64(v))
	case int16:
		return Int64Value(int64(v))
	case int32:
		return Int64Value(int64(v))
	case float64:
		return Float64Value(v)
	case float32:
		return Float64Value(float64(v))
	case []Attr:
		return GroupValue(v...)
	case Kind:
		return Value{any: kind(v)}
	case Value:
		return v
	default:
		return Value{any: v}
	}
}

//////////////// Accessors

// Any returns v's value as an any.
func (v Value) Any() any {
	switch v.Kind() {
	case KindAny:
		if k, ok := v.any.(kind); ok {
			return Kind(k)
		}
		return v.any
	case KindGroup:
		return v.group()
	case KindInt64:
		return int64(v.num)
	case KindUint64:
		return v.num
	case KindFloat64:
		return v.float()
	case KindString:
		return v.str()
	case KindBool:
		return v.bool()
	case KindDuration:
		return v.duration()
	case KindTime:
		return v.time()
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}

// String returns Value's value as a string, formatted like fmt.Sprint. Unlike
// the methods Int64, Float64, and so on, which panic if v is of the
// wrong kind, String never panics.
func (v Value) String() string {
	if sp, ok := v.any.(stringptr); ok {
		return unsafe.String(sp, v.num)
	}
	var buf []byte
	return string(v.append(buf))
}

func (v Value) str() string {
	return unsafe.String(v.any.(stringptr), v.num)
}

// Int64 returns v's value as an int64. It panics
// if v is not a signed integer.
func (v Value) Int64() int64 {
	if g, w := v.Kind(), KindInt64; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return int64(v.num)
}

// Uint64 returns v's value as a uint64. It panics
// if v is not an unsigned integer.
func (v Value) Uint64() uint64 {
	if g, w := v.Kind(), KindUint64; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.num
}

// Bool returns v's value as a bool. It panics
// if v is not a bool.
func (v Value) Bool() bool {
	if g, w := v.Kind(), KindBool; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.bool()
}

func (v Value) bool() bool {
	return v.num == 1
}

// Duration returns v's value as a time.Duration. It panics
// if v is not a time.Duration.
func (v Value) Duration() time.Duration {
	if g, w := v.Kind(), KindDuration; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}

	return v.duration()
}

func (v Value) duration() time.Duration {
	return time.Duration(int64(v.num))
}

// Float64 returns v's value as a float64. It panics
// if v is not a float64.
func (v Value) Float64() float64 {
	if g, w := v.Kind(), KindFloat64; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}

	return v.float()
}

func (v Value) float() float64 {
	return math.Float64frombits(v.num)
}

// Time returns v's value as a time.Time. It panics
// if v is not a time.Time.
func (v Value) Time() time.Time {
	if g, w := v.Kind(), KindTime; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.time()
}

func (v Value) time() time.Time {
	loc := v.any.(timeLocation)
	if loc == nil {
		return time.Time{}
	}
	return time.Unix(0, int64(v.num)).In(loc)
}

// Group returns v's value as a []Attr.
// It panics if v's Kind is not KindGroup.
func (v Value) Group() []Attr {
	if sp, ok := v.any.(groupptr); ok {
		return unsafe.Slice(sp, v.num)
	}
	panic("Group: bad kind")
}

func (v Value) group() []Attr {
	return unsafe.Slice((*Attr)(v.any.(groupptr)), v.num)
}

// append appends a text representation of v to dst.
// v is formatted as with fmt.Sprint.
func (v Value) append(dst []byte) []byte {
	switch v.Kind() {
	case KindString:
		return append(dst, v.str()...)
	case KindInt64:
		return strconv.AppendInt(dst, int64(v.num), 10)
	case KindUint64:
		return strconv.AppendUint(dst, v.num, 10)
	case KindFloat64:
		return strconv.AppendFloat(dst, v.float(), 'g', -1, 64)
	case KindBool:
		return strconv.AppendBool(dst, v.bool())
	case KindDuration:
		return append(dst, v.duration().String()...)
	case KindTime:
		return append(dst, v.time().String()...)
	case KindGroup:
		return fmt.Append(dst, v.group())
	case KindAny:
		return fmt.Append(dst, v.any)
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}
