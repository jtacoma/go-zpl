// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"testing"
)

var (
	raw = `
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
        bind = inproc://device`
)

func TestUnmarshal_Map(t *testing.T) {
	conf := make(map[string]interface{})
	err := Unmarshal([]byte(raw), conf)
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
	iothreads := tmp.([]interface{})
	if len(iothreads) != 1 {
		t.Fatalf("len(iothreads) = %d", len(iothreads))
	}
	if iothreads[0].(string) != "1" {
		t.Fatalf("context/iothreads[0] = %v", iothreads[0])
	}
	tmp, ok = context["verbose"]
	verbose := tmp.([]interface{})
	if verbose[0].(string) != "1" {
		t.Fatalf("context/verbose[0] = %v", verbose[0])
	}
	main := conf["main"].(map[string]interface{})
	frontend := main["frontend"].(map[string]interface{})
	option := frontend["option"].(map[string]interface{})
	subscribe := option["subscribe"].([]interface{})
	if subscribe[0] != "#2" {
		t.Fatalf("main/frontend/subscribe[0] = %v (length %d)", subscribe[0], len(subscribe[0].(string)))
	}
	backend := main["backend"].(map[string]interface{})
	bind := backend["bind"].([]interface{})
	if bind[0] != "tcp://eth0:5556" {
		t.Fatalf("main/backend/bind[0] = %v", bind[0])
	}
	if bind[1] != "inproc://device" {
		t.Fatalf("main/backend/bind[1] = %v", bind[0])
	}
}

type ZdcfRoot struct {
	Context ZdcfContext            `context`
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
	Options *ZdcfOptions `option`
	Bind    []string     `bind`
	Connect []string     `connect`
}

type ZdcfOptions struct {
	Hwm       int      `zpl:"hwm"`
	Swap      int64    `swap`
	Subscribe []string `zpl:"subscribe"`
}

func TestUnmarshal_Reflect(t *testing.T) {
	var conf ZdcfRoot
	err := Unmarshal([]byte(raw), &conf)
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
}
