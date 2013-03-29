// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zpl

import (
	"testing"
)

type marshalCase struct {
	Output string
	Value  interface{}
}

type marshalMock struct {
	Int   *int   `int`
	Int32 *int32 `zpl:"int32"`
}

func TestMarshal_Map(t *testing.T) {
	var i int = 1
	var i32 int32 = 32
	tests := []marshalCase{
		{"version = 1\n", map[string]int{"version": 1}},
		{"version = 1\n", map[string]*int{"version": &i}},
		{"version = 1\n", map[string]interface{}{"version": int(1)}},
		{"version = 1\n", map[string]interface{}{"version": int16(1)}},
		{"version = 1\n", map[string]interface{}{"version": int32(1)}},
		{"version = 1\n", map[string]interface{}{"version": int64(1)}},
		{"version = 1\n", map[string]interface{}{"version": uint(1)}},
		{"version = 1\n", map[string]interface{}{"version": uint16(1)}},
		{"version = 1\n", map[string]interface{}{"version": uint32(1)}},
		{"version = 1\n", map[string]interface{}{"version": uint64(1)}},
		{"version = 1\n", map[string]interface{}{"version": float32(1)}},
		{"version = 1\n", map[string]interface{}{"version": float64(1)}},
		{"ok = 1\n", map[string]interface{}{"ok": true}},
		{"ok = 0\n", map[string]interface{}{"ok": false}},
		{"int = 1\n", marshalMock{Int: &i}},
		{"int = 1\n", &marshalMock{Int: &i}},
		{"int32 = 32\n", &marshalMock{Int32: &i32}},
		{"root\n    c0 = testing\n", map[string]interface{}{"root": map[string]string{"c0": "testing"}}},
		{"c0 = test\n", map[string]map[string]string{"*": map[string]string{"c0": "test"}}},
		{"root\n    int = 1\n", map[string]interface{}{"root": &marshalMock{Int: &i}}},
	}
	for _, c := range tests {
		bytes, err := Marshal(c.Value)
		if err != nil {
			t.Errorf(err.Error())
		}
		if string(bytes) != c.Output {
			t.Errorf("unexpected result: %s", string(bytes))
		}
	}
}
