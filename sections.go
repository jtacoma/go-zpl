// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"fmt"
	"strconv"
)

type Section struct {
	Properties map[string][]interface{}
	Sections   map[string]*Section
}

func NewSection() *Section {
	return &Section{
		Properties: make(map[string][]interface{}),
		Sections:   make(map[string]*Section),
	}
}

type sectionBuilder struct {
	sections []*Section
}

func (b *sectionBuilder) consume(e *parseEvent) error {
	if b == nil {
		return fmt.Errorf("nil builder cannot consume events.")
	}
	if len(b.sections) == 0 {
		b.sections = []*Section{NewSection()}
	}
	switch e.Type {
	case addValue:
		s := b.sections[len(b.sections)-1]
		already := s.Properties[e.Name]
		s.Properties[e.Name] = append(already, e.Value)
	case endSection:
		b.sections = b.sections[:len(b.sections)-1]
	case startSection:
		section := NewSection()
		b.sections[len(b.sections)-1].Sections[e.Name] = section
		b.sections = append(b.sections, section)
	default:
		return fmt.Errorf("unsupported event type %d.", e.Type)
	}
	return nil
}

func (s *Section) GetBool(name string) (value bool, err error) {
	if values, ok := s.Properties[name]; !ok {
		err = NotFound
	} else if len(values) != 1 {
		err = fmt.Errorf("expected exactly one %v, found %d.", name, len(values))
	} else {
		switch values[0].(type) {
		case string:
			if value, err = strconv.ParseBool(values[0].(string)); err != nil {
				err = fmt.Errorf("failed to parse %v: %s", name, err)
			}
		default:
			err = fmt.Errorf("unsupported data type for %v: %T", name, values[0])
		}
	}
	return
}

func (s *Section) GetFloat32(name string) (value float32, err error) {
	if values, ok := s.Properties[name]; !ok {
		err = NotFound
	} else if len(values) != 1 {
		err = fmt.Errorf("expected exactly one %v, found %d.", name, len(values))
	} else {
		switch values[0].(type) {
		case string:
			var parsed float64
			if parsed, err = strconv.ParseFloat(values[0].(string), 32); err != nil {
				err = fmt.Errorf("failed to parse %v: %s", name, err)
			}
			value = float32(parsed)
		default:
			err = fmt.Errorf("unsupported data type for %v: %T", name, values[0])
		}
	}
	return
}

func (s *Section) GetInt(name string) (value int, err error) {
	if values, ok := s.Properties[name]; !ok {
		err = NotFound
	} else if len(values) != 1 {
		err = fmt.Errorf("expected exactly one %v, found %d.", name, len(values))
	} else {
		switch values[0].(type) {
		case string:
			var parsed int64
			if parsed, err = strconv.ParseInt(values[0].(string), 10, 32); err != nil {
				err = fmt.Errorf("failed to parse %v: %s", name, err)
			}
			value = int(parsed)
		default:
			err = fmt.Errorf("unsupported data type for %v: %T", name, values[0])
		}
	}
	return
}

func (s *Section) GetString(name string) (value string, err error) {
	if values, ok := s.Properties[name]; !ok {
		err = NotFound
	} else if len(values) != 1 {
		err = fmt.Errorf("expected exactly one %v, found %d.", name, len(values))
	} else {
		switch values[0].(type) {
		case string:
			value = values[0].(string)
		default:
			err = fmt.Errorf("unsupported data type for %v: %T", name, values[0])
		}
	}
	return
}
