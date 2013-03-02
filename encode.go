// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zpl

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

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
		return fmt.Errorf("zpl: unexpected end of section.")
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
		marshalProperty(e, name, value.Elem())
	default:
		// Silently fail to marshal what we don't know how to marshal.
	}
	return nil
}
