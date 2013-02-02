// Copyright 2013 Joshua Tacoma. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gozpl

import (
	"testing"
)

func TestUnmarshalZpl(t *testing.T) {
	raw := `
#   Notice that indentation is always 4 spaces, there are no tabs.
#
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
