// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"bytes"
	"fmt"
	"regexp"
)

var (
	NotFound = fmt.Errorf("not found.")
)

var (
	reskip       = regexp.MustCompile(`^\s*(#.*)?$`)
	reskipinline = regexp.MustCompile(`\s*(#.*)?$`)
	rekeyvalue   = regexp.MustCompile(
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

func Unmarshal(src []byte, dst interface{}) error {
	var builder sink
	switch dst.(type) {
	case sink:
		builder = dst.(sink)
	case interface{}:
		builder = &reflectionBuilder{pointers: []interface{}{dst}}
	default:
		return fmt.Errorf("cannot unmarshal ZPL into %T", dst)
	}
	prevDepth := 0
	for lineno, line := range splitLines(src) {
		//if inline := bytes.IndexByte(line, '#'); inline >= 0 { line = line[:inline] }
		//if skip:=reskipinline.Find(line);skip!=nil{ line=line[:skip[0]] }
		line = bytes.TrimRight(line, " \t\n\r") // TODO: rewhitespace
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
