// Package yaml is a stub of gopkg.in/yaml.v3 for the fixtures.
package yaml

import "io"

// Unmarshal decodes in into out.
func Unmarshal(in []byte, out any) error { return nil }

// Decoder reads a YAML document from a stream.
type Decoder struct{}

// NewDecoder returns a Decoder reading from r.
func NewDecoder(r io.Reader) *Decoder { return &Decoder{} }

// Decode decodes the next document into out.
func (d *Decoder) Decode(out any) error { return nil }
