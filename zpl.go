// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The go-zpl package provides methods for consuming and producing data
// in the ZeroMQ Property Language (ZPL).
//
// ZPL is defined here: http://rfc.zeromq.org/spec:4.
//
//     package main
//     
//     import "github.com/jtacoma/go-zpl"
//     
//     var data = `
//     endpoints
//         worker1
//             addr = 192.168.0.10
//             port = 8080
//         worker2
//             addr = 192.168.0.11
//             port = 9999
//     `
//     
//     type Endpoint struct {
//         Address string `zpl:"addr"`
//         Port    int    `zpl:"port"`
//     }
//     
//     type Config struct {
//         Endpoints map[string]*Endpoint `zpl:"endpoints"`
//     }
//     
//     var config Config
//     
//     func main() {
//         if err := zpl.Unmarshal([]byte(data), &config); err != nil {
//             panic(err.Error())
//         }
//         // ...
//         println(config.Endpoints["worker1"].Port)
//         // ...
//     }
//
// This package mimics Go's standard "encodings" packages so should be
// easy to get started with.  Since this package is new, there may be
// some loose ends: please submit bug reports to:
//
// https://github.com/jtacoma/go-zpl/issues
//
// The errors include line numbers but not yet much else.  For example,
// it doesn't mention that you've got an underscore in name, a number
// sign in a value, or a tab character where a multiple of 4 spaces was
// expected.
//
// While ZPL shares some structural qualities with JSON there are some
// major differences worth noting.  First, the root must be either an
// object or a map.  Of course, in a configuration file format, this
// makes perfect sense.  Second, two or more ZPL files can be
// effectively merged by simply concatenating them together.  Given that
// configuration files are often split into different scopes (e.g.
// system, application, user), this feature is also good news.
//
package zpl

import (
	"bytes"
	"reflect"
	"strconv"
)

// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
//
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "zpl: Unmarshal(nil)"
	}
	if e.Type.Kind() != reflect.Ptr {
		return "zpl: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "zpl: Unmarshal(nil " + e.Type.String() + ")"
}

// A SyntaxError is a description of a ZPL syntax error.
//
type SyntaxError struct {
	msg  string // description of error
	Line uint64 // error occurred on this line
}

func (e *SyntaxError) Error() string {
	return strconv.FormatUint(e.Line, 10) + ":" + e.msg
}

// Marshal returns the ZPL encoding of v.
//
// Marshal traverses the value v recursively, using the following type-dependent
// default encodings:
//
// Boolean values encode as ints (0 for false or 1 for true).
//
// Floating point and integer values encode as base-10 numbers.
//
// String values encode as strings.  Invalid character sequences will cause
// Marshal to return an UnsupportedValueError.  Line breaks are invalid.
//
// Array and slice values encode as repetitions of the same property.
//
// Struct values encode as ZPL sections.  Each exported struct field becomes a
// property in the section unless the field's tag is "-".  The "zpl" key in the
// struct field's tag value is the key name.  Examples:
//
//   // Field is ignored by this package.
//   Field int `zpl:"-"`
//
//   // Field appears in ZPL as property "name".
//   Field int `zpl:"name"`
//
// The key name will be used if it's a non-empty string consisting of only
// alphanumeric ([A-Za-z0-9]) characters.
//
// Map values encode as ZPL sections unless their tag is "*", in which case they
// will be collapsed into their parent.  There can be only one "*"-tagged map in
// any marshalled struct.  The map's key type must be string; the map keys are
// used directly as property and sub-section names.
//
// Pointer values encode as the value pointed to.
//
// Interface values encode as the value contained in the interface.
//
// Channel, complex, and function values cannot be encoded in ZPL.  Attempting
// to encode such a value causes Marshal to return an UnsupportedTypeError.
//
// ZPL cannot represent cyclic data structures and Marshal does not handle them.
// Passing cyclic structures to Marshal will result in an infinite recursion.
//
func Marshal(v interface{}) ([]byte, error) {
	var (
		buf = &bytes.Buffer{}
		e   = NewEncoder(buf)
		err = e.Encode(v)
	)
	return buf.Bytes(), err
}

// Unmarshal parses the ZPL-encoded data and stores the result in the value
// pointed to by v.
//
// Unmarshal allocates maps, slices, and pointers as necessary while following
// these rules:
//
// To unmarshal ZPL into a pointer, Unmarshal unmarshals the ZPL into the value
// pointed at by the pointer.  If the pointer is nil, Unmarshal allocates a new
// value for it to point to.
//
// To unmarshal ZPL into an interface value, Unmarshal unmarshals the ZPL into
// the concrete value contained in the interface value.  If the interface value
// is nil, that is, has no concrete value stored in it, Unmarshal stores a
// map[string]interface{} in the interface value.
//
// If a ZPL value is not appropriate for a given target type, or if a ZPL number
// overflows the target type, Unmarshal returns the error after processing the
// remaining data.
//
func Unmarshal(src []byte, dst interface{}) error {
	r := bytes.NewReader(src)
	d := NewDecoder(r)
	return d.Decode(dst)
}

type (
	eventType  int
	parseEvent struct {
		Type  eventType
		Name  string
		Value string
	}
	sink interface {
		consume(*parseEvent) error
	}
)

const (
	addValue eventType = iota
	endSection
	startSection
)
