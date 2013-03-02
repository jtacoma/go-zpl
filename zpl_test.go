// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zpl

import "fmt"

func ExampleUnmarshal() {
	z := "salutation = Hello,\nsalutation = See you tomorrow,"
	m := make(map[string][]string)
	Unmarshal([]byte(z), m)
	for _, s := range m["salutation"] {
		fmt.Println(s, "World!")
	}
	// Output:
	// Hello, World!
	// See you tomorrow, World!
}
