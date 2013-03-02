// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zpl

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// A Decoder represents a ZPL parser reading a particular input stream.  The
// parser assumes that its input is encoded in UTF-8.
//
type Decoder struct {
	r         io.Reader
	prevDepth int
	buffer    []byte
	lineno    int
	queue     []*parseEvent
}

// NewDecoder creates a new ZPL parser that reads from r.
//
// The decoder introduces its own buffering and may read data from r beyond
// the ZPL values requested.
//
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: r,
	}
}

// Decode reads the next ZPL-encoded value from its input and stores it in the
// value pointed to by v.
//
// See the documentation for Unmarshal for details about the conversion of ZPL
// into a Go value.
//
func (d *Decoder) Decode(v interface{}) error {
	var (
		builder sink
		fault   error
	)
	switch v.(type) {
	case sink:
		builder = v.(sink)
	default:
		if builder, fault = newBuilder(v); fault != nil {
			return fault
		}
	}
	for {
		e, err := d.next()
		if e != nil {
			if e := builder.consume(e); e != nil && fault == nil {
				fault = e
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	return fault
}

func (d *Decoder) next() (e *parseEvent, err error) {
	if len(d.queue) > 0 {
		e = d.queue[0]
		d.queue = d.queue[1:]
		return
	}
	var line []byte
	for {
		d.lineno += 1
		for {
			n := bytes.IndexAny(d.buffer, "\n\r")
			if n >= 0 {
				line = d.buffer[:n]
				if n+1 < len(d.buffer) {
					switch d.buffer[n] {
					case '\r':
						if d.buffer[n+1] == '\n' {
							n += 1
						}
					case '\n':
						if d.buffer[n+1] == '\r' {
							n += 1
						}
					}
				}
				d.buffer = d.buffer[n+1:]
				break
			}
			b := make([]byte, 64)
			n, err = d.r.Read(b)
			if err == io.EOF {
				d.buffer = append(d.buffer, b[:n]...)
				break
			} else if err != nil {
				return // error from Read()
			} else {
				d.buffer = append(d.buffer, b[:n]...)
			}
		}
		if err == io.EOF {
			break
		} else if len(line) == 0 || reskip.Match(line) {
			continue
		} else {
			break
		}
	}
	if err == io.EOF && len(line) == 0 {
		if len(d.buffer) > 0 {
			line = d.buffer
		} else {
			return // nothing left to read
		}
	}
	match := rekeyquoted.FindSubmatch(line)
	if match == nil {
		match = rekeyvalue.FindSubmatch(line)
	}
	if match != nil {
		depth := len(match[1]) / 4
		for depth < d.prevDepth {
			d.queue = append(d.queue, &parseEvent{Type: endSection})
			d.prevDepth--
		}
		key := string(match[3])
		if len(match[5]) > 0 {
			value := string(match[6])
			d.queue = append(d.queue, &parseEvent{Type: addValue, Name: key, Value: value})
		} else {
			d.queue = append(d.queue, &parseEvent{Type: startSection, Name: key})
			d.prevDepth++
		}
		e = d.queue[0]
		d.queue = d.queue[1:]
	} else {
		err = &SyntaxError{
			Line: int64(d.lineno),
			msg:  "is neither a comment, a section header, nor a key = value setting.",
		}
	}
	return
}

type builder struct {
	refs []reflect.Value
}

func newBuilder(v interface{}) (*builder, error) {
	var (
		value = reflect.ValueOf(v)
		err   error
	)
	switch value.Kind() {
	case reflect.Ptr:
		value = value.Elem()
	case reflect.Map:
		// Ok.
	default:
		err = fmt.Errorf("cannot modify: must be a map or a pointer to a struct: %v.", value.Type())
	}
	if err != nil {
		return nil, err
	}
	return &builder{refs: []reflect.Value{value}}, nil
}

func (b *builder) consume(e *parseEvent) error {
	if b == nil {
		return fmt.Errorf("nil builder cannot consume events.")
	}
	if len(b.refs) == 0 {
		return fmt.Errorf("uninitialized builder cannot consume events.")
	}
	switch e.Type {
	case addValue:
		ref := b.refs[len(b.refs)-1]
		if err := addValueToSection(ref, e.Name, e.Value); err != nil {
			return err
		}
	case endSection:
		b.refs = b.refs[:len(b.refs)-1]
	case startSection:
		ref := b.refs[len(b.refs)-1]
		if next, err := getSubSection(ref, e.Name); err != nil {
			return err
		} else if !next.IsValid() {
			return fmt.Errorf("encountered invalid value for %v.", e.Name)
		} else {
			b.refs = append(b.refs, next)
		}
	default:
		return fmt.Errorf("unsupported event type %d.", e.Type)
	}
	return nil
}

func getSubSection(section reflect.Value, name string) (sub reflect.Value, err error) {
	if section.Type().Kind() == reflect.Map {
		if section.Type().Key().Kind() != reflect.String {
			err = fmt.Errorf("map key type must be string")
			return
		}
		if section.IsNil() {
			section.Set(reflect.MakeMap(section.Type()))
		}
		switch section.Type().Elem().Kind() {
		case reflect.Ptr:
			ptr := section.MapIndex(reflect.ValueOf(name))
			if !ptr.IsValid() {
				ptr = reflect.New(section.Type().Elem().Elem())
				section.SetMapIndex(reflect.ValueOf(name), ptr)
			} else if ptr.IsNil() {
				ptr.Set(reflect.New(section.Type().Elem()))
			}
			return ptr.Elem(), nil
		case reflect.Interface:
			newmap := reflect.ValueOf(make(map[string]interface{}))
			section.SetMapIndex(reflect.ValueOf(name), newmap)
			return newmap, nil
		default:
			err = fmt.Errorf("cannot add sub-section: map[string]%v is not yet supported.", section.Type().Elem())
			return
		}
	} else if section.Type().Kind() == reflect.Struct {
		var fi = -1
		var squash = false
		for i := 0; i < section.NumField(); i++ {
			tag := section.Type().Field(i).Tag
			if string(tag) == name || tag.Get("zpl") == name {
				fi = i
			} else if (string(tag) == "*" || tag.Get("zpl") == "*") && fi < 0 {
				fi = i
				squash = true
			}
		}
		if fi == -1 {
			err = fmt.Errorf("%v has no field tagged %v", section.Type(), name)
			return
		}
		field := section.Field(fi)
		if field.Type().Kind() == reflect.Map {
			if field.Type().Key().Kind() != reflect.String {
				err = fmt.Errorf("map key type must be string")
				return
			}
			if field.IsNil() {
				field.Set(reflect.MakeMap(field.Type()))
			}
			if !squash {
				sub = field
			} else {
				helper := field
				sub, err = getSubSection(helper, name)
				if err != nil {
					return
				}
			}
		} else if field.Type().Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			sub = field.Elem()
		} else {
			err = fmt.Errorf("cannot unmarshal into %v", field.Type())
		}
	} else {
		err = &InvalidUnmarshalError{Type: section.Type()}
		return
	}
	if !sub.IsValid() && err == nil {
		err = fmt.Errorf("unknown error.")
	}
	return
}

func addValueToSection(section reflect.Value, name string, value string) error {
	switch section.Type().Kind() {
	case reflect.Map:
		key := reflect.ValueOf(name)
		existing := section.MapIndex(key)
		adjusted, err := appendValue(section.Type().Elem(), existing, value)
		if err != nil {
			return err
		}
		if adjusted.IsValid() {
			section.SetMapIndex(key, adjusted)
		}
	case reflect.Ptr, reflect.Struct:
		var fi = -1
		for i := 0; i < section.NumField(); i++ {
			tag := section.Type().Field(i).Tag
			if string(tag) == name || tag.Get("zpl") == name {
				fi = i
			}
		}
		if fi == -1 {
			return fmt.Errorf("%v has no field tagged %v", section.Type(), name)
		}
		existing := section.Field(fi)
		adjusted, err := appendValue(existing.Type(), existing, value)
		if err != nil {
			return err
		}
		if !adjusted.IsValid() && !existing.IsValid() {
			return fmt.Errorf("failed to add value for %v.", name)
		} else if adjusted.IsValid() && adjusted != existing {
			existing.Set(adjusted)
		}
	default:
		return fmt.Errorf("cannot add value: must be a map or a pointer to a struct: %v.", section.Type())
	}
	return nil
}

// Append value to target (optionally a 
func appendValue(typ reflect.Type, target reflect.Value, value string) (result reflect.Value, err error) {
	if target.IsValid() {
		typ = target.Type()
	}
	if typ.Kind() == reflect.Interface {
		typ = reflect.TypeOf([]string{})
	}
	switch typ.Kind() {
	case reflect.Bool:
		if parsed, err := strconv.ParseBool(value); err != nil {
			err = fmt.Errorf("could not parse bool: %v", value)
		} else if target.IsValid() {
			target.SetBool(parsed)
		} else {
			result = reflect.ValueOf(parsed)
		}
	case reflect.Float32, reflect.Float64:
		if parsed, err := strconv.ParseFloat(value, typ.Bits()); err != nil {
			err = fmt.Errorf("could not parse float: %v", value)
		} else if target.IsValid() {
			target.SetFloat(parsed)
		} else {
			switch typ.Kind() {
			case reflect.Float32:
				result = reflect.ValueOf(float32(parsed))
			default:
				result = reflect.ValueOf(parsed)
			}
		}
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		if parsed, err := strconv.ParseInt(value, 10, typ.Bits()); err != nil {
			err = fmt.Errorf("could not parse int: %v", value)
		} else if target.IsValid() {
			target.SetInt(parsed)
		} else {
			switch typ.Kind() {
			case reflect.Int:
				result = reflect.ValueOf(int(parsed))
			case reflect.Int16:
				result = reflect.ValueOf(int16(parsed))
			case reflect.Int32:
				result = reflect.ValueOf(int32(parsed))
			default:
				result = reflect.ValueOf(parsed)
			}
		}
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if parsed, err := strconv.ParseUint(value, 10, typ.Bits()); err != nil {
			err = fmt.Errorf("could not parse unsigned int: %v", value)
		} else if target.IsValid() {
			target.SetUint(parsed)
		} else {
			switch typ.Kind() {
			case reflect.Uint:
				result = reflect.ValueOf(uint(parsed))
			case reflect.Uint16:
				result = reflect.ValueOf(uint16(parsed))
			case reflect.Uint32:
				result = reflect.ValueOf(uint32(parsed))
			default:
				result = reflect.ValueOf(parsed)
			}
		}
	case reflect.Ptr:
		if target.IsNil() {
			target.Set(reflect.New(typ.Elem()))
		}
		var elem reflect.Value
		if elem, err = appendValue(typ.Elem(), target.Elem(), value); err == nil && elem.IsValid() {
			target.Elem().Set(elem)
		}
	case reflect.String:
		result = reflect.ValueOf(value)
	case reflect.Slice:
		var next reflect.Value
		switch typ.Elem().Kind() {
		case reflect.String:
			next = reflect.ValueOf(value)
		}
		if next.IsValid() {
			if !target.IsValid() {
				result = reflect.MakeSlice(typ, 0, 4)
			} else if target.Type().Kind() == reflect.Interface {
				result = reflect.ValueOf(target.Interface())
			} else {
				result = target
			}
			result = reflect.Append(result, next)
		} else {
			err = fmt.Errorf("slice of %v is not yet supported.", typ.Elem())
		}
	default:
		err = fmt.Errorf("cannot set or append to %v", typ)
	}
	return
}
