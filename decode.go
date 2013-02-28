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
	var builder sink
	switch v.(type) {
	case sink:
		builder = v.(sink)
	default:
		builder = newBuilder(v)
	}
	for {
		e, err := d.next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		} else if err = builder.consume(e); err != nil {
			return err
		}
	}
	return nil
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
				d.buffer = append(d.buffer, b...)
				break
			} else if err != nil {
				return
			} else {
				d.buffer = append(d.buffer, b...)
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
		return
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

type modifier interface {
	getSection(name string) (modifier, error)
	addValue(name string, value string) error
}

type builder struct {
	refs []modifier
}

func newBuilder(root interface{}) *builder {
	var ref modifier
	switch root.(type) {
	case map[string]interface{}:
		ref = mapModifier(root.(map[string]interface{}))
	default:
		ref = newRefModifier(root)
	}
	return &builder{refs: []modifier{ref}}
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
		if err := ref.addValue(e.Name, e.Value); err != nil {
			return err
		}
	case endSection:
		b.refs = b.refs[:len(b.refs)-1]
	case startSection:
		ref := b.refs[len(b.refs)-1]
		if next, err := ref.getSection(e.Name); err != nil {
			return err
		} else {
			b.refs = append(b.refs, next)
		}
	default:
		return fmt.Errorf("unsupported event type %d.", e.Type)
	}
	return nil
}

type mapModifier map[string]interface{}

func (m mapModifier) getSection(name string) (modifier, error) {
	var section map[string]interface{}
	if already, ok := m[name]; !ok {
		section = make(map[string]interface{})
		m[name] = section
	} else {
		// TODO: return an error nicely instead of failing a type assertion
		section = already.(map[string]interface{})
	}
	return mapModifier(section), nil
}

func (m mapModifier) addValue(name string, value string) error {
	if already, ok := m[name]; !ok {
		m[name] = []interface{}{value}
	} else {
		switch already.(type) {
		case []string:
			m[name] = append(already.([]string), value)
		case []interface{}:
			m[name] = append(already.([]interface{}), value)
		default:
			return fmt.Errorf("unsupported destination property value type: %T", already)
		}
	}
	return nil
}

type refModifier struct {
	reflect.Value
}

func newRefModifier(v interface{}) refModifier {
	return refModifier{reflect.ValueOf(v).Elem()}
}

func (m refModifier) getSection(name string) (section modifier, err error) {
	if m.Type().Kind() == reflect.Map {
		if m.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key type must be string")
		}
		if m.IsNil() {
			m.Set(reflect.MakeMap(m.Type()))
		}
		switch m.Type().Elem().Kind() {
		case reflect.Ptr:
			ptr := m.MapIndex(reflect.ValueOf(name))
			if !ptr.IsValid() {
				ptr = reflect.New(m.Type().Elem().Elem())
				m.SetMapIndex(reflect.ValueOf(name), ptr)
			} else if ptr.IsNil() {
				ptr.Set(reflect.New(m.Type().Elem()))
			}
			section = refModifier{ptr.Elem()}
		case reflect.Map:
			return nil, fmt.Errorf("map of maps is not yet supported.")
		default:
			return nil, fmt.Errorf("map of %v is not yet supported.", m.Type().Elem())
		}
	} else if m.Type().Kind() == reflect.Struct {
		var fi = -1
		var squash = false
		for i := 0; i < m.NumField(); i++ {
			tag := m.Type().Field(i).Tag
			if string(tag) == name || tag.Get("zpl") == name {
				fi = i
			} else if (string(tag) == "*" || tag.Get("zpl") == "*") && fi < 0 {
				fi = i
				squash = true
			}
		}
		if fi == -1 {
			return nil, fmt.Errorf("%v has no field tagged %v", m.Type(), name)
		}
		field := m.Field(fi)
		if field.Type().Kind() == reflect.Map {
			if field.Type().Key().Kind() != reflect.String {
				return nil, fmt.Errorf("map key type must be string")
			}
			if field.IsNil() {
				field.Set(reflect.MakeMap(field.Type()))
			}
			if !squash {
				section = refModifier{field}
			} else {
				helper := refModifier{field}
				section, err = helper.getSection(name)
				if err != nil {
					return nil, err
				}
			}
		} else if field.Type().Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			section = refModifier{field.Elem()}
		} else {
			return nil, fmt.Errorf("cannot unmarshal into %v", field.Type())
		}
	} else {
		return nil, &InvalidUnmarshalError{Type: m.Type()}
	}
	if section == nil {
		return nil, fmt.Errorf("unknown error.")
	}
	return section, nil
}

func (m refModifier) addValue(name string, value string) error {
	var fi = -1
	for i := 0; i < m.NumField(); i++ {
		tag := m.Type().Field(i).Tag
		if string(tag) == name || tag.Get("zpl") == name {
			fi = i
		}
	}
	if fi == -1 {
		return fmt.Errorf("%v has no field tagged %v", m.Type(), name)
	}
	field := m.Field(fi)
	return m.addValueToField(field, value)
}

func (m refModifier) addValueToField(field reflect.Value, value string) error {
	switch field.Type().Kind() {
	case reflect.Bool:
		if parsed, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("could not parse bool %v", value)
		} else {
			field.SetBool(parsed)
		}
	case reflect.Float32, reflect.Float64:
		if parsed, err := strconv.ParseFloat(value, field.Type().Bits()); err != nil {
			return fmt.Errorf("could not parse float %v", value)
		} else {
			field.SetFloat(parsed)
		}
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		if parsed, err := strconv.ParseInt(value, 10, field.Type().Bits()); err != nil {
			return fmt.Errorf("could not parse int %v", value)
		} else {
			field.SetInt(parsed)
		}
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return m.addValueToField(field.Elem(), value)
	case reflect.String:
		field.SetString(value)
	case reflect.Slice:
		var next reflect.Value
		typ := field.Type()
		switch typ.Elem().Kind() {
		case reflect.String:
			next = reflect.ValueOf(value)
		default:
			return fmt.Errorf("slice of %v is not yet supported.", typ)
		}
		field.Set(reflect.Append(field, next))
	default:
		return fmt.Errorf("cannot set field of type %v", field.Type())
	}
	return nil
}
