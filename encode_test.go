// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"testing"
)

func TestMarshal_Map(t *testing.T) {
	tests := map[string]interface{}{
		"version = 1\n":            map[string]int{"version": 1},
		"root\n    c0 = testing\n": map[string]interface{}{"root": map[string]string{"c0": "testing"}},
		"c0 = test\n":              map[string]map[string]string{"*": map[string]string{"c0": "test"}},
	}
	for text, data := range tests {
		bytes, err := Marshal(data)
		if err != nil {
			t.Errorf(err.Error())
		}
		if string(bytes) != text {
			t.Errorf("unexpected result: %s", string(bytes))
		}
	}
}
