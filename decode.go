// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"fmt"
	"reflect"
	"strconv"
)

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
	E reflect.Value
	T reflect.Type
}

func newRefModifier(v interface{}) *refModifier {
	ref := &refModifier{
		E: reflect.ValueOf(v).Elem(),
	}
	ref.T = ref.E.Type()
	return ref
}

func (m refModifier) getSection(name string) (modifier, error) {
	var fi = -1
	for i := 0; i < m.E.NumField(); i++ {
		tag := m.T.Field(i).Tag
		if string(tag) == name || tag.Get("zpl") == name {
			fi = i
		} else if (string(tag) == "*" || tag.Get("zpl") == "*") && fi < 0 {
			fi = i
		}
	}
	if fi == -1 {
		return nil, fmt.Errorf("unknown name: %v", name)
	}
	field := m.E.Field(fi)
	if field.Type().Kind() == reflect.Map {
		if field.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key type must be string")
		}
		switch field.Type().Elem().Kind() {
		case reflect.Ptr:
			if field.IsNil() {
				field.Set(reflect.MakeMap(field.Type()))
			}
			ptr := field.MapIndex(reflect.ValueOf(name))
			if !ptr.IsValid() {
				ptr = reflect.New(field.Type().Elem().Elem())
				field.SetMapIndex(reflect.ValueOf(name), ptr)
			} else if ptr.IsNil() {
				ptr.Set(reflect.New(field.Type().Elem()))
			}
			return &refModifier{E: ptr.Elem(), T: ptr.Elem().Type()}, nil
		case reflect.Map:
			return nil, fmt.Errorf("map of maps is not yet supported.")
		default:
			return nil, fmt.Errorf("map of %v is not yet supported.", field.Type().Elem())
		}
	} else if field.Type().Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return &refModifier{E: field.Elem(), T: field.Elem().Type()}, nil
	}
	return &refModifier{E: field, T: field.Type()}, nil
}

func (m refModifier) addValue(name string, value string) error {
	var fi = -1
	for i := 0; i < m.E.NumField(); i++ {
		tag := m.T.Field(i).Tag
		if string(tag) == name || tag.Get("zpl") == name {
			fi = i
		}
	}
	if fi == -1 {
		return fmt.Errorf("unknown name: %v", name)
	}
	field := m.E.Field(fi)
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
