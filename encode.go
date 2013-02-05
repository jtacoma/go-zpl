// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type writer struct {
	s      string
	indent string
}

func (w *writer) addValue(name string, value string) {
	w.s = fmt.Sprintf("%s%s%s = %s\n", w.s, w.indent, name, value)
}

func (w *writer) startSection(name string) {
	w.s = fmt.Sprintf("%s%s%s\n", w.s, w.indent, name)
	w.indent = fmt.Sprintf("%s    ", w.indent)
}

func (w *writer) endSection() error {
	if len(w.indent) < 4 {
		return fmt.Errorf("zpl: unexpected end of section.")
	}
	w.indent = w.indent[:len(w.indent)-4]
	return nil
}

func marshalProperty(w *writer, name string, value reflect.Value) error {
	switch value.Type().Kind() {
	case reflect.Map:
		if name != "*" {
			w.startSection(name)
		}
		for _, key := range value.MapKeys() {
			v := value.MapIndex(key)
			if err := marshalProperty(w, key.Interface().(string), v); err != nil {
				return err
			}
		}
		if name != "*" {
			if err := w.endSection(); err != nil {
				return err
			}
		}
	case reflect.Struct:
		w.startSection(name)
		marshal(w, value)
		if err := w.endSection(); err != nil {
			return err
		}
	case reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		w.addValue(name, strconv.FormatInt(value.Int(), 10))
	case reflect.Float32, reflect.Float64:
		w.addValue(name, strconv.FormatFloat(value.Float(), 'f', -1, value.Type().Bits()))
	case reflect.Bool:
		if value.Bool() {
			w.addValue(name, "1")
		} else {
			w.addValue(name, "0")
		}
	case reflect.String:
		w.addValue(name, value.String())
	case reflect.Ptr, reflect.Interface:
		marshalProperty(w, name, value.Elem())
	default:
		// Silently fail to marshal what we don't know how to marshal.
	}
	return nil
}

func marshal(w *writer, value reflect.Value) error {
	switch value.Type().Kind() {
	case reflect.Ptr:
		return marshal(w, value.Elem())
	case reflect.Map:
		if value.Type().Key().Kind() == reflect.String {
			for _, key := range value.MapKeys() {
				v := value.MapIndex(key)
				if err := marshalProperty(w, key.String(), v); err != nil {
					return err
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
					return err
				}
			}
		}
	}
	return nil
}
