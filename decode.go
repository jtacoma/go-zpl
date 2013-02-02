// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"fmt"
	//"reflect"
	//"strconv"
)

type reflectionBuilder struct {
	pointers []interface{}
}

func (b *reflectionBuilder) consume(e *parseEvent) error {
	if b == nil {
		return fmt.Errorf("nil builder cannot consume events.")
	}
	if len(b.pointers) == 0 {
		b.pointers = []interface{}{make(map[string]interface{})}
	}
	switch e.Type {
	case addValue:
		ownerRef := b.pointers[len(b.pointers)-1]
		switch ownerRef.(type) {
		case map[string]interface{}:
			ownerMap := ownerRef.(map[string]interface{})
			if already, ok := ownerMap[e.Name]; !ok {
				println("setting", e.Name, e.Value)
				ownerMap[e.Name] = []interface{}{e.Value}
			} else {
				switch already.(type) {
				case []string:
					println("appending", e.Name, e.Value)
					ownerMap[e.Name] = append(already.([]string), e.Value)
				case []interface{}:
					println("appending", e.Name, e.Value)
					ownerMap[e.Name] = append(already.([]interface{}), e.Value)
				default:
					return fmt.Errorf("unsupported destination property value type: %T", already)
				}
			}
		default:
			return fmt.Errorf("unsupported destination object type: %T", ownerRef)
		}
	case endSection:
		println("pop")
		b.pointers = b.pointers[:len(b.pointers)-1]
	case startSection:
		println("push", e.Name)
		ownerRef := b.pointers[len(b.pointers)-1]
		switch ownerRef.(type) {
		case map[string]interface{}:
			ownerMap := ownerRef.(map[string]interface{})
			if _, ok := ownerMap[e.Name]; !ok {
				ownerMap[e.Name] = make(map[string]interface{})
			}
			b.pointers = append(b.pointers, ownerMap[e.Name])
		default:
			return fmt.Errorf("unsupported destination object type: %T", ownerRef)
		}
	default:
		return fmt.Errorf("unsupported event type %d.", e.Type)
	}
	return nil
}
