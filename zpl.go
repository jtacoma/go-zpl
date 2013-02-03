// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The gozpl package provides methods for consuming and producing data in the
// ZeroMQ Property Language (ZPL) encoding.
//
// Example code in this package will use the name "zpl":
// 
//  import zpl "github.com/jtacoma/gozpl"
//  
// Data presented in the ZeroMQ Property Language, documented fully at
// http://rfc.zeromq.org/spec:4, might look like this:
//
//  threads = 2
//  zpl
//      alias = "ZPL"
//      alias = "ZeroMQ Property Language"
//      indent = "    "
//      base = 10
//  json
//      alias = "JSON"
//      indent = "  "
//
// To load this with gozpl, use the same conventions used in the
// "encodings/json" and "encodings/xml" packages.  First, tag some structs:
//
//  type Config struct {
//      Threads int                `threads`
//      Format  map[string]*Format `zpl:"*"`
//  }
//  type Format struct {
//      Aliases []string `alias`
//      Indent  string   `indent`
//      Base    int      `base`
//  }
//
// Then use zpl.Unmarshal to load a Config from a []byte:
//
//  func main() {
//      var config Config
//      if err := zpl.Unmarshal(bytes_from_zpl_file, &config); err != nil {
//          panic (err)
//      }
//      // ...
//  }
//
package gozpl

import (
	"bytes"
	"fmt"
	"regexp"
)

var (
	reskip     = regexp.MustCompile(`^\s*(#.*)?$`)
	rekeyvalue = regexp.MustCompile(
		`^(?P<indent>(    )*)(?P<key>[a-zA-Z0-9][a-zA-Z0-9/]*)(\s*(?P<hasvalue>=)\s*(?P<value>[^ ].*))?$`)
	rekeyquoted = regexp.MustCompile(
		`^(?P<indent>(    )*)(?P<key>[a-zA-Z0-9][a-zA-Z0-9/]*)(\s*(?P<hasvalue>=)\s*"(?P<value>[^ ].*)")?$`)
)

func splitLines(blob []byte) [][]byte {
	return bytes.FieldsFunc(blob, func(r rune) bool {
		return r == 10 || r == 13
	})
}

type eventType int

const (
	addValue eventType = iota
	endSection
	startSection
)

type parseEvent struct {
	Type  eventType
	Name  string
	Value string
}

type sink interface {
	consume(*parseEvent) error
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
// overflows the target type, Unmarshal fails and returns the error immediately
// without returning any partially decoded data.
//
func Unmarshal(src []byte, dst interface{}) error {
	var builder sink
	switch dst.(type) {
	case sink:
		builder = dst.(sink)
	case interface{}:
		builder = newBuilder(dst)
	default:
		return fmt.Errorf("cannot unmarshal ZPL into %T", dst)
	}
	prevDepth := 0
	for lineno, line := range splitLines(src) {
		// TODO: handle whitespace more correctly
		line = bytes.TrimRight(line, " \t\n\r")
		if len(line) == 0 || reskip.Match(line) {
			continue
		}
		match := rekeyquoted.FindSubmatch(line)
		if match == nil {
			match = rekeyvalue.FindSubmatch(line)
		}
		if match != nil {
			depth := len(match[1]) / 4
			if depth < prevDepth {
				if err := builder.consume(&parseEvent{Type: endSection}); err != nil {
					return err
				}
			}
			prevDepth = depth
			key := string(match[3])
			if len(match[5]) > 0 {
				value := string(match[6])
				if err := builder.consume(&parseEvent{Type: addValue, Name: key, Value: value}); err != nil {
					return err
				}
			} else {
				if err := builder.consume(&parseEvent{Type: startSection, Name: key}); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("line %d: invalid ZPL: %v", lineno, string(line))
		}
	}
	return nil
}
