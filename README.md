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

## Usage

#### func  Marshal

```go
func Marshal(v interface{}) ([]byte, error)
```
Marshal returns the ZPL encoding of v.

Marshal traverses the value v recursively, using the following type-dependent
default encodings:

Boolean values encode as ints (0 for false or 1 for true).

Floating point and integer values encode as base-10 numbers.

String values encode as strings. Invalid character sequences will cause Marshal
to return an UnsupportedValueError. Line breaks are invalid.

Array and slice values encode as repetitions of the same property.

Struct values encode as ZPL sections. Each exported struct field becomes a
property in the section unless the field's tag is "-". The "zpl" key in the
struct field's tag value is the key name. Examples:

    // Field is ignored by this package.
    Field int `zpl:"-"`
    // Field appears in ZPL as property "name".
    Field int `zpl:"name"`

The key name will be used if it's a non-empty string consisting of only
alphanumeric ([A-Za-z0-9]) characters.

Map values encode as ZPL sections unless their tag is "*", in which case they
will be collapsed into their parent. There can be only one "*"-tagged map in any
marshalled struct. The map's key type must be string; the map keys are used
directly as property and sub-section names.

Pointer values encode as the value pointed to.

Interface values encode as the value contained in the interface.

Channel, complex, and function values cannot be encoded in ZPL. Attempting to
encode such a value causes Marshal to return an UnsupportedTypeError.

ZPL cannot represent cyclic data structures and Marshal does not handle them.
Passing cyclic structures to Marshal will result in an infinite recursion.

#### func  Unmarshal

```go
func Unmarshal(src []byte, dst interface{}) error
```
Unmarshal parses the ZPL-encoded data and stores the result in the value pointed
to by v.

Unmarshal allocates maps, slices, and pointers as necessary while following
these rules:

To unmarshal ZPL into a pointer, Unmarshal unmarshals the ZPL into the value
pointed at by the pointer. If the pointer is nil, Unmarshal allocates a new
value for it to point to.

To unmarshal ZPL into an interface value, Unmarshal unmarshals the ZPL into the
concrete value contained in the interface value. If the interface value is nil,
that is, has no concrete value stored in it, Unmarshal stores a
map[string]interface{} in the interface value.

If a ZPL value is not appropriate for a given target type, or if a ZPL number
overflows the target type, Unmarshal returns the error after processing the
remaining data.

#### type Decoder

```go
type Decoder struct {
}
```

A Decoder represents a ZPL parser reading a particular input stream. The parser
assumes that its input is encoded in UTF-8.

#### func  NewDecoder

```go
func NewDecoder(r io.Reader) *Decoder
```
NewDecoder creates a new ZPL parser that reads from r.

The decoder introduces its own buffering and may read data from r beyond the ZPL
values requested.

#### func (*Decoder) Decode

```go
func (d *Decoder) Decode(v interface{}) error
```
Decode reads the next ZPL-encoded value from its input and stores it in the
value pointed to by v.

See the documentation for Unmarshal for details about the conversion of ZPL into
a Go value.

#### type Encoder

```go
type Encoder struct {
}
```

An Encoder write ZPL to an output stream.

#### func  NewEncoder

```go
func NewEncoder(w io.Writer) *Encoder
```
NewEncoder returns a new encoder that writes to w.

#### func (*Encoder) Encode

```go
func (w *Encoder) Encode(v interface{}) error
```
Encode writes the ZPL encoding of v to the connection.

See the documentation for Marshal for details about the conversion of Go values
to ZPL.

#### type InvalidUnmarshalError

```go
type InvalidUnmarshalError struct {
	Type reflect.Type
}
```

An InvalidUnmarshalError describes an invalid argument passed to Unmarshal. (The
argument to Unmarshal must be a non-nil pointer.)

#### func (*InvalidUnmarshalError) Error

```go
func (e *InvalidUnmarshalError) Error() string
```

#### type SyntaxError

```go
type SyntaxError struct {
	Line uint64 // error occurred on this line
}
```

A SyntaxError is a description of a ZPL syntax error.

#### func (*SyntaxError) Error

```go
func (e *SyntaxError) Error() string
```

#### type UnmarshalFieldError

```go
type UnmarshalFieldError struct {
	Key  string
	Type reflect.Type
}
```

An UnmarshalFieldError describes describes a ZPL key that could not be matched
to a map key or struct field.

#### func (*UnmarshalFieldError) Error

```go
func (e *UnmarshalFieldError) Error() string
```

#### type UnmarshalTypeError

```go
type UnmarshalTypeError struct {
	Value string       // description of ZPL value - "bool", "array", "number -5"
	Type  reflect.Type // type of Go value it could not be assigned to
}
```

An UnmarshalTypeError describes a ZPL value that was not appropriate for a value
of a specific Go type.

#### func (*UnmarshalTypeError) Error

```go
func (e *UnmarshalTypeError) Error() string
```

## License

Use of this source code is governed by a BSD-style license that can be found in
the LICENSE file.
