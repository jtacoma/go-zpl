// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zpl

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"regexp"
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
		return "zpl: cannot unmarshal into nil."
	}
	return "zpl: cannot unmarshal into " + e.Type.String() + "."
}

// An UnmarshalFieldError describes describes a ZPL key that could not be matched to a map key or struct field.
//
type UnmarshalFieldError struct {
	Key  string
	Type reflect.Type
}

func (e *UnmarshalFieldError) Error() string {
	return "zpl: no field tagged \"" + e.Key + "\" could be found on " + e.Type.String()
}

// An UnmarshalTypeError describes a ZPL value that was not appropriate for a value of a specific Go type.
//
type UnmarshalTypeError struct {
	Value string       // description of ZPL value - "bool", "array", "number -5"
	Type  reflect.Type // type of Go value it could not be assigned to
}

func (e *UnmarshalTypeError) Error() string {
	return "zpl: cannot unmarshal " + e.Value + " into " + e.Type.String()
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

// A Decoder represents a ZPL parser reading a particular input stream.  The
// parser assumes that its input is encoded in UTF-8.
//
type Decoder struct {
	r         io.Reader
	prevDepth int
	buffer    []byte
	lineno    uint64
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
	if builder, fault = newBuilder(v); fault != nil {
		return fault
	}
	for {
		e, err := d.next()
		if e != nil {
			if err2 := builder.consume(e); err2 != nil && fault == nil {
				fault = err2
				break
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	if fault != nil {
	}
	return fault
}

var (
	rekeyvalue = regexp.MustCompile(
		`^(?P<indent>(    )*)(?P<key>[a-zA-Z0-9][a-zA-Z0-9/]*)(\s*(?P<hasvalue>=)\s*(?P<value>[^ ].*))?$`)
	rekeyquoted = regexp.MustCompile(
		`^(?P<indent>(    )*)(?P<key>[a-zA-Z0-9][a-zA-Z0-9/]*)(\s*(?P<hasvalue>=)\s*"(?P<value>[^ ].*)")?$`)
)

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
		} else if len(line) == 0 || bytes.Trim(line, " \t")[0] == '#' {
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
			Line: uint64(d.lineno),
			msg:  "is neither a comment, a section header, nor a key = value setting.",
		}
	}
	return
}

type builder struct {
	refs []reflect.Value
}

func newBuilder(v interface{}) (*builder, error) {
	if v == nil {
		return nil, &InvalidUnmarshalError{nil}
	}
	var (
		value = reflect.ValueOf(v)
		err   error
	)
	switch value.Kind() {
	case reflect.Ptr:
		value = value.Elem()
		switch value.Kind() {
		case reflect.Map, reflect.Struct:
			// Ok.
		default:
			err = &InvalidUnmarshalError{reflect.TypeOf(v)}
		}
	case reflect.Map:
		switch value.Type().Key().Kind() {
		case reflect.String:
			// Ok.
		default:
			err = &InvalidUnmarshalError{reflect.TypeOf(v)}
		}
	default:
		err = &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
	if err != nil {
		return nil, err
	}
	return &builder{refs: []reflect.Value{value}}, nil
}

func (b *builder) consume(e *parseEvent) error {
	if b == nil {
		panic("zpl: nil builder cannot consume events.")
	}
	if len(b.refs) == 0 {
		panic("zpl: uninitialized builder cannot consume events.")
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
		} else {
			b.refs = append(b.refs, next)
		}
	default:
		panic("zpl: program error: unsupported event type??")
	}
	return nil
}

func getSubSection(section reflect.Value, name string) (sub reflect.Value, err error) {
	if section.Type().Kind() == reflect.Map {
		sub = section.MapIndex(reflect.ValueOf(name))
		if section.Type().Elem().Kind() == reflect.Interface {
			if !sub.IsValid() || sub.IsNil() {
				sub = reflect.ValueOf(make(map[string]interface{}))
				section.SetMapIndex(reflect.ValueOf(name), sub)
			} else {
				sub = reflect.ValueOf(sub.Interface())
			}
			return
		}
		switch section.Type().Elem().Kind() {
		case reflect.Ptr:
			if !sub.IsValid() {
				sub = reflect.New(section.Type().Elem().Elem())
				section.SetMapIndex(reflect.ValueOf(name), sub)
			} else if sub.IsNil() {
				sub.Set(reflect.New(section.Type().Elem()))
			}
			sub = sub.Elem()
			return
		case reflect.Map:
			if section.Type().Elem().Key().Kind() != reflect.String {
				err = &UnmarshalTypeError{
					Value: "subsection \"" + name + "\"",
					Type:  section.Type().Elem(),
				}
			} else if !sub.IsValid() || sub.IsNil() {
				sub = reflect.MakeMap(section.Type().Elem())
				section.SetMapIndex(reflect.ValueOf(name), sub)
			}
			return
		default:
			err = &UnmarshalTypeError{
				Value: "subsection \"" + name + "\"",
				Type:  section.Type().Elem(),
			}
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
			err = &UnmarshalFieldError{
				Key:  name,
				Type: section.Type(),
			}
			return
		}
		field := section.Field(fi)
		if field.Type().Kind() == reflect.Map {
			if field.Type().Key().Kind() != reflect.String {
				err = &UnmarshalTypeError{
					Value: "subsection \"" + name + "\"",
					Type:  field.Type(),
				}
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
			err = errors.New("zpl: cannot unmarshal into " + field.Type().String())
		}
	} else {
		err = &InvalidUnmarshalError{Type: section.Type()}
		return
	}
	if !sub.IsValid() && err == nil {
		err = errors.New("zpl: unknown error.")
	}
	return
}

func addValueToSection(section reflect.Value, name string, value string) error {
	switch section.Type().Kind() {
	case reflect.Map:
		if section.Type().Key().Kind() != reflect.String {
			return &UnmarshalTypeError{
				"value for key \"" + name + "\"",
				section.Type(),
			}
		}
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
			return &UnmarshalFieldError{
				Key:  name,
				Type: section.Type(),
			}
		}
		existing := section.Field(fi)
		adjusted, err := appendValue(existing.Type(), existing, value)
		if err != nil {
			return err
		}
		if !adjusted.IsValid() && !existing.IsValid() {
			return errors.New("zpl: failed to add value for " + name)
		} else if adjusted.IsValid() && adjusted != existing {
			existing.Set(adjusted)
		}
	default:
		return &UnmarshalFieldError{
			Key:  name,
			Type: section.Type(),
		}
	}
	return nil
}

// Append value to target or return a new value of type typ.
func appendValue(typ reflect.Type, target reflect.Value, value string) (result reflect.Value, err error) {
	if target.IsValid() {
		typ = target.Type()
	}
	if typ.Kind() == reflect.Interface {
		typ = reflect.TypeOf([]string{})
	}
	switch typ.Kind() {
	case reflect.Bool:
		if parsed, err2 := strconv.ParseBool(value); err2 != nil {
			err = &UnmarshalTypeError{Value: value, Type: typ}
		} else if target.IsValid() && target.CanSet() {
			target.SetBool(parsed)
		} else {
			result = reflect.ValueOf(parsed)
		}
	case reflect.Float32, reflect.Float64:
		if parsed, err2 := strconv.ParseFloat(value, typ.Bits()); err2 != nil {
			err = &UnmarshalTypeError{Value: value, Type: typ}
		} else if target.IsValid() && target.CanSet() {
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
		if parsed, err2 := strconv.ParseInt(value, 10, typ.Bits()); err2 != nil {
			err = &UnmarshalTypeError{Value: value, Type: typ}
		} else if target.IsValid() && target.CanSet() {
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
		if parsed, err2 := strconv.ParseUint(value, 10, typ.Bits()); err2 != nil {
			err = &UnmarshalTypeError{Value: value, Type: typ}
		} else if target.IsValid() && target.CanSet() {
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
		result = reflect.New(typ.Elem())
		var elem reflect.Value
		if elem, err = appendValue(typ.Elem(), elem, value); err == nil {
			result.Elem().Set(elem)
		}
	case reflect.String:
		result = reflect.ValueOf(value)
	case reflect.Slice:
		var next reflect.Value
		next, err = appendValue(typ.Elem(), next, value)
		if err == nil && next.IsValid() {
			result = target
			if result.IsValid() && result.Type().Kind() == reflect.Interface {
				result = reflect.ValueOf(result.Interface())
			}
			if !result.IsValid() {
				result = reflect.MakeSlice(typ, 0, 4)
			}
			result = reflect.Append(result, next)
		}
	default:
		err = &UnmarshalTypeError{
			Value: value,
			Type:  typ,
		}
	}
	return
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
