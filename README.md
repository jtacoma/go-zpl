# zpl
--
    import "github.com/jtacoma/go-zpl"

[![Build Status](https://drone.io/github.com/jtacoma/go-zpl/status.png)](https://drone.io/github.com/jtacoma/go-zpl/latest) [![Build Status](https://travis-ci.org/jtacoma/go-zpl.png)](https://travis-ci.org/jtacoma/go-zpl)

The go-zpl package provides methods for consuming and producing data
in the ZeroMQ Property Language (ZPL).

ZPL is defined here: http://rfc.zeromq.org/spec:4.

    package main

    import "github.com/jtacoma/go-zpl"

    var data = `
    endpoints
        worker1
            addr = 192.168.0.10
            port = 8080
        worker2
            addr = 192.168.0.11
            port = 9999
    `

    type Endpoint struct {
        Address string `zpl:"addr"`
        Port    int    `zpl:"port"`
    }

    type Config struct {
        Endpoints map[string]*Endpoint `zpl:"endpoints"`
    }

    var config Config

    func main() {
        if err := zpl.Unmarshal([]byte(data), &config); err != nil {
            panic(err.Error())
        }
        // ...
        println(config.Endpoints["worker1"].Port)
        // ...
    }

This package mimics Go's standard "encodings" packages so should be
easy to get started with.  Since this package is new, there may be
some loose ends: please submit bug reports to:

https://github.com/jtacoma/go-zpl/issues

The errors include line numbers but not yet much else.  For example,
it doesn't mention that you've got an underscore in name, a number
sign in a value, or a tab character where a multiple of 4 spaces was
expected.

While ZPL shares some structural qualities with JSON there are some
major differences worth noting.  First, the root must be either an
object or a map.  Of course, in a configuration file format, this
makes perfect sense.  Second, two or more ZPL files can be
effectively merged by simply concatenating them together.  Given that
configuration files are often split into different scopes (e.g.
system, application, user), this feature is also good news.

## License

Use of this source code is governed by a BSD-style license that can be found in
the LICENSE file.
