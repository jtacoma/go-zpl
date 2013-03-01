// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zpl

import (
	"testing"
)

var (
	raw0 = []byte(`
#   Notice that indentation is always 4 spaces, there are no tabs.
#
version = 0.1

context
    iothreads = 1
    verbose = 1

main
    type = zmq_queue
    frontend
        option
            hwm = 1000
            swap = 25000000
            subscribe = "#2"
        bind = tcp://eth0:5555
    backend
        bind = tcp://eth0:5556
        bind = inproc://device

auxiliary
    type = foo
    socket0
    socket1`)
	raw1 = []byte(`version = 1
# The first line is not a comment.  What happens?
words
    cat
        kind = mammal
        alias = feline
    dog
        kind = mammal
        alias = canine`)
	bad0 = []byte(`
# This is an example of an invalid ZPL document.
invalid line with spaces`)
	bad1 = []byte(`
# This is an example of an invalid ZPL document.
    key = overly indented value`)
)

func TestUnmarshal_Map(t *testing.T) {
	conf := make(map[string]interface{})
	err := Unmarshal(raw0, conf)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	tmp, ok := conf["context"]
	if !ok {
		t.Fatalf("context not found.")
	}
	context := tmp.(map[string]interface{})
	tmp, ok = context["iothreads"]
	if !ok {
		t.Fatalf("iothreads not found.")
	}
	iothreads := tmp.([]string)
	if len(iothreads) != 1 {
		t.Fatalf("len(iothreads) = %d", len(iothreads))
	}
	if iothreads[0] != "1" {
		t.Fatalf("context/iothreads[0] = %v", iothreads[0])
	}
	tmp, ok = context["verbose"]
	verbose := tmp.([]string)
	if verbose[0] != "1" {
		t.Fatalf("context/verbose[0] = %v", verbose[0])
	}
	main := conf["main"].(map[string]interface{})
	frontend := main["frontend"].(map[string]interface{})
	option := frontend["option"].(map[string]interface{})
	subscribe := option["subscribe"].([]string)
	if subscribe[0] != "#2" {
		t.Fatalf("main/frontend/subscribe[0] = %v (length %d)", subscribe[0], len(subscribe[0]))
	}
	backend := main["backend"].(map[string]interface{})
	ibind, ok := backend["bind"]
	if !ok {
		t.Fatalf("main/backend/bind = nil")
	}
	bind := ibind.([]string)
	if bind[0] != "tcp://eth0:5556" {
		t.Fatalf("main/backend/bind[0] = %v", bind[0])
	}
	if bind[1] != "inproc://device" {
		t.Fatalf("main/backend/bind[1] = %v", bind[0])
	}
}

type ZdcfRoot struct {
	Context *ZdcfContext           `context`
	Devices map[string]*ZdcfDevice `*`
	Version float32                `version`
}

type ZdcfContext struct {
	IoThreads int  `iothreads`
	Verbose   bool `verbose`
}

type ZdcfDevice struct {
	Type    string                 `type`
	Sockets map[string]*ZdcfSocket `*`
}

type ZdcfSocket struct {
	Type    string       `type`
	Options *ZdcfOptions `option`
	Bind    []string     `bind`
	Connect []string     `connect`
}

type ZdcfOptions struct {
	Hwm       *int     `zpl:"hwm"`
	Swap      *int64   `swap`
	Subscribe []string `zpl:"subscribe"`
}

type Dictionary struct {
	Version float32          `version`
	Words   map[string]*Word `words`
}

type Word struct {
	Kind    string   `kind`
	Aliases []string `alias`
}

func TestUnmarshal_Reflect(t *testing.T) {
	var conf ZdcfRoot
	err := Unmarshal(raw0, &conf)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if conf.Version != 0.1 {
		t.Fatalf("version = %v", conf.Version)
	}
	if conf.Context.IoThreads != 1 {
		t.Fatalf("context/iothreads = %v", conf.Context.IoThreads)
	}
	if conf.Context.Verbose != true {
		t.Fatalf("context/verbose = %v", conf.Context.Verbose)
	}
	if *conf.Devices["main"].Sockets["frontend"].Options.Hwm != 1000 {
		t.Fatalf("main/frontend/hwm = %v",
			*conf.Devices["main"].Sockets["frontend"].Options.Hwm)
	}
	if conf.Devices["main"].Sockets["frontend"].Options.Subscribe[0] != "#2" {
		t.Fatalf("main/frontend/subscribe[0] = %v",
			conf.Devices["main"].Sockets["frontend"].Options.Subscribe[0])
	}
	if conf.Devices["main"].Sockets["backend"].Bind[0] != "tcp://eth0:5556" {
		t.Fatalf("main/backend/bind[0] = %v", conf.Devices["main"].Sockets["backend"].Bind[0])
	}
	if conf.Devices["main"].Sockets["backend"].Bind[1] != "inproc://device" {
		t.Fatalf("main/backend/bind[1] = %v", conf.Devices["main"].Sockets["backend"].Bind[1])
	}
	var dict Dictionary
	err = Unmarshal(raw1, &dict)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if _, ok := dict.Words["words"]; ok {
		t.Fatalf("words/words exists")
	}
	if dict.Words["cat"].Aliases[0] != "feline" {
		t.Fatalf("words/cat/alias[0] = %v", dict.Words["cat"].Aliases[0])
	}
}

func TestUnmarshal_Bad(t *testing.T) {
	var conf ZdcfRoot
	err := Unmarshal(bad0, &conf)
	if err == nil {
		t.Fatalf("expected error unmarshalling bad0, got none.")
	} else {
		switch err.(type) {
		case *SyntaxError:
			synerr := err.(*SyntaxError)
			if synerr.Line != 3 {
				t.Fatalf("expected syntax error on line 3, got line %d.", synerr.Line)
			}
		default:
			t.Fatalf("expected syntax error, got %T.", err)
		}
	}
	err = Unmarshal(bad1, &conf)
	if err == nil {
		t.Fatalf("expected error unmarshalling bad1, got none.")
	}
}
