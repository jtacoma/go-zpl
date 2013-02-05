// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The gozpl package provides methods for consuming and producing data in the
// ZeroMQ Property Language (ZPL) encoding.
//
package gozpl

import (
	"bytes"
	"reflect"
	"regexp"
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
	Line int64  // error occurred line number Line
}

func (e *SyntaxError) Error() string { return e.msg }

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
// Marshal to return an UnsupportedValueError.
//
// Array and slice values encode as repetitions of the same property.
//
// Struct values encode as ZPL sections.  Each exported struct field becomes a
// member of the object unless the field's tag is "-".  The "zpl" key in the
// struct field's tag value is the key name.  Examples:
//
//   // Field is ignored by this package.
//   Field int `zpl:"-"`
//
//   // Field appears in ZPL as property "myName".
//   Field int `zpl:"myName"`
//
// The key name will be used if it's a non-empty string consisting of only
// Unicode letters, digits, dollar signs, percent signs, hyphens, underscores
// and slashes.
//
// Map values encode as ZPL sections unless their tag is "*", in which case they
// will be collapsed into their parent.  There can be only one "*"-tagged
// map in any marshalled struct.  The map's key type must be string; the object
// keys are used directly as map keys.
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
		w   writer
		err = marshal(&w, reflect.ValueOf(v))
	)
	if err != nil {
		return nil, err
	}
	return []byte(w.s), nil
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
// overflows the target type, Unmarshal fails and returns the error immediately
// without returning any partially decoded data.
//
func Unmarshal(src []byte, dst interface{}) error {
	var builder sink
	switch dst.(type) {
	case sink:
		builder = dst.(sink)
	default:
		builder = newBuilder(dst)
	}
	prevDepth := 0
	for lineno, line := range splitLines(src) {
		// TODO: handle whitespace more correctly
		line = bytes.TrimRight(line, " \t\n\r")
		if len(line) == 0 || reskip.Match(line) {
			continue
		}
		match := rekeyquoted.FindSubmatch(line)
		if match == nil {
			match = rekeyvalue.FindSubmatch(line)
		}
		if match != nil {
			depth := len(match[1]) / 4
			for depth < prevDepth {
				if err := builder.consume(&parseEvent{Type: endSection}); err != nil {
					return err
				}
				prevDepth--
			}
			key := string(match[3])
			if len(match[5]) > 0 {
				value := string(match[6])
				if err := builder.consume(&parseEvent{Type: addValue, Name: key, Value: value}); err != nil {
					return err
				}
			} else {
				if err := builder.consume(&parseEvent{Type: startSection, Name: key}); err != nil {
					return err
				}
				prevDepth++
			}
		} else {
			return &SyntaxError{
				Line: int64(lineno + 1),
				msg:  "this line is neither a comment, a section header, nor a key = value setting.",
			}
		}
	}
	return nil
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

var (
	reskip     = regexp.MustCompile(`^\s*(#.*)?$`)
	rekeyvalue = regexp.MustCompile(
		`^(?P<indent>(    )*)(?P<key>[a-zA-Z0-9][a-zA-Z0-9/]*)(\s*(?P<hasvalue>=)\s*(?P<value>[^ ].*))?$`)
	rekeyquoted = regexp.MustCompile(
		`^(?P<indent>(    )*)(?P<key>[a-zA-Z0-9][a-zA-Z0-9/]*)(\s*(?P<hasvalue>=)\s*"(?P<value>[^ ].*)")?$`)
)

func splitLines(blob []byte) [][]byte {
	// Splitting precisely on actual line breaks is not so straightforward...
	var lines [][]byte
	var eol int
	for i := 0; i < len(blob); eol = bytes.IndexAny(blob[i:], "\x0A\x0D") {
		if eol == -1 {
			lines = append(lines, blob[i:])
			break
		} else {
			lines = append(lines, blob[i:i+eol])
			if blob[i+eol] == 0x0D && eol+1 < len(blob) && blob[i+eol+1] == 0x0A {
				i += eol + 2
			} else {
				i += eol + 1
			}
		}
	}
	return lines
}
