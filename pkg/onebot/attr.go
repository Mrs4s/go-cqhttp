// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package onebot

import (
	"time"
)

// An Attr is a key-value pair.
type Attr struct {
	Key   string
	Value Value
}

// String returns an Attr for a string value.
func String(key, value string) Attr {
	return Attr{key, StringValue(value)}
}

// Int64 returns an Attr for an int64.
func Int64(key string, value int64) Attr {
	return Attr{key, Int64Value(value)}
}

// Int converts an int to an int64 and returns
// an Attr with that value.
func Int(key string, value int) Attr {
	return Int64(key, int64(value))
}

// Uint64 returns an Attr for a uint64.
func Uint64(key string, v uint64) Attr {
	return Attr{key, Uint64Value(v)}
}

// Float64 returns an Attr for a floating-point number.
func Float64(key string, v float64) Attr {
	return Attr{key, Float64Value(v)}
}

// Bool returns an Attr for a bool.
func Bool(key string, v bool) Attr {
	return Attr{key, BoolValue(v)}
}

// Time returns an Attr for a time.Time.
// It discards the monotonic portion.
func Time(key string, v time.Time) Attr {
	return Attr{key, TimeValue(v)}
}

// Duration returns an Attr for a time.Duration.
func Duration(key string, v time.Duration) Attr {
	return Attr{key, DurationValue(v)}
}

// Group returns an Attr for a Group Value.
// The caller must not subsequently mutate the
// argument slice.
//
// Use Group to collect several Attrs under a single
// key on a log line, or as the result of LogValue
// in order to log a single value as multiple Attrs.
func Group(key string, as ...Attr) Attr {
	return Attr{key, GroupValue(as...)}
}

// Any returns an Attr for the supplied value.
// See [Value.AnyValue] for how values are treated.
func Any(key string, value any) Attr {
	return Attr{key, AnyValue(value)}
}

func (a Attr) String() string {
	return a.Key + "=" + a.Value.String()
}
