// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zpl

import (
	"bytes"
	"io"
	"reflect"
	"strconv"
	"strings"
)

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

// An Encoder write ZPL to an output stream.
//
type Encoder struct {
	w      io.Writer
	indent string
	br     string
}

// NewEncoder returns a new encoder that writes to w.
//
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:  w,
		br: "\n",
	}
}

// Encode writes the ZPL encoding of v to the connection.
//
// See the documentation for Marshal for details about the conversion of Go
// values to ZPL.
//
func (w *Encoder) Encode(v interface{}) error {
	return w.encode(reflect.ValueOf(v))
}

func (w *Encoder) encode(value reflect.Value) error {
	var fault error
	switch value.Type().Kind() {
	case reflect.Ptr:
		return w.encode(value.Elem())
	case reflect.Map:
		if value.Type().Key().Kind() == reflect.String {
			for _, key := range value.MapKeys() {
				v := value.MapIndex(key)
				if err := marshalProperty(w, key.String(), v); err != nil {
					if fault == nil {
						fault = err
					}
				}
			}
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			tag := value.Type().Field(i).Tag
			var name string
			if strings.Contains(string(tag), ":") {
				name = tag.Get("zpl")
			} else {
				name = string(tag)
			}
			if len(tag) > 0 {
				if err := marshalProperty(w, name, value.Field(i)); err != nil {
					if fault == nil {
						fault = err
					}
				}
			}
		}
	}
	return fault
}

func (e *Encoder) addValue(name string, value string) error {
	_, err := e.w.Write([]byte(e.indent + name + " = " + value + e.br))
	return err
}

func (e *Encoder) startSection(name string) error {
	if _, err := e.w.Write([]byte(e.indent + name + e.br)); err != nil {
		return err
	}
	e.indent += "    "
	return nil
}

func (e *Encoder) endSection() error {
	if len(e.indent) < 4 {
		panic("zpl: unexpected end of section.")
	}
	e.indent = e.indent[:len(e.indent)-4]
	return nil
}

func marshalProperty(e *Encoder, name string, value reflect.Value) error {
	switch value.Type().Kind() {
	case reflect.Map:
		if name != "*" {
			e.startSection(name)
		}
		for _, key := range value.MapKeys() {
			v := value.MapIndex(key)
			if err := marshalProperty(e, key.Interface().(string), v); err != nil {
				return err
			}
		}
		if name != "*" {
			if err := e.endSection(); err != nil {
				return err
			}
		}
	case reflect.Struct:
		e.startSection(name)
		e.encode(value)
		if err := e.endSection(); err != nil {
			return err
		}
	case reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		e.addValue(name, strconv.FormatInt(value.Int(), 10))
	case reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		e.addValue(name, strconv.FormatUint(value.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		e.addValue(name, strconv.FormatFloat(value.Float(), 'f', -1, value.Type().Bits()))
	case reflect.Bool:
		if value.Bool() {
			e.addValue(name, "1")
		} else {
			e.addValue(name, "0")
		}
	case reflect.String:
		e.addValue(name, value.String())
	case reflect.Ptr, reflect.Interface:
		if !value.IsNil() {
			marshalProperty(e, name, value.Elem())
		}
	default:
		// Silently fail to marshal what we don't know how to marshal.
	}
	return nil
}
