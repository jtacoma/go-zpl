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
    verbose = 1      #   Ask for a trace

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
	conf := NewSection()
	err := Unmarshal([]byte(raw), conf)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if conf == nil {
		t.Fatalf("unmarshal returned two nils.")
	}
	context, ok := conf.Sections["context"]
	if !ok {
		t.Fatalf("context not found.")
	}
	iothreads, ok := context.Properties["iothreads"]
	if !ok {
		t.Fatalf("iothreads not found.")
	}
	if len(iothreads) != 1 {
		t.Fatalf("len(iothreads) = %d", len(iothreads))
	}
	if iothreads[0].(string) != "1" {
		t.Fatalf("context/iothreads[0] = %v", iothreads[0])
	}
	if context.Properties["verbose"][0].(string) != "1" {
		t.Fatalf("context/verbose[0] = %v", context.Properties["verbose"][0])
	}
	if conf.Sections["main"].Sections["frontend"].Properties["bind"][0].(string) != "tcp://eth0:5555" {
		t.Fatalf("main/frontend/bind[0] = %v", conf.Sections["main"].Sections["frontend"].Properties["bind"][0])
	}
	if conf.Sections["main"].Sections["backend"].Properties["bind"][1].(string) != "inproc://device" {
		t.Fatalf("main/backend/bind[1] = %v", conf.Sections["main"].Sections["backend"].Properties["bind"][1])
	}
}
