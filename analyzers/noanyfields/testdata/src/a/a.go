package a

import (
	"database/sql/driver"
	"encoding/json"
)

type Payload = any

type Blob interface{}

// --- invalid: struct fields typed empty interface -----------------------------

type Event struct {
	Payload any         // want `noanyfields: field "Payload" has type any`
	Meta    interface{} // want `noanyfields: field "Meta" has type interface\{\}`
	Data    Payload     // want `noanyfields: field "Data" has type Payload`
	Extra   Blob        // want `noanyfields: field "Extra" has type Blob`
	lo, hi  any         // want `noanyfields: fields "lo", "hi" have type any`
}

// An embedded field of an owned empty interface type carries no evidence either.
type Envelope struct {
	Payload // want `noanyfields: field "Payload" has type Payload`
}

// Anonymous structs are struct types too, in every position.
var anon struct {
	Value any // want `noanyfields: field "Value" has type any`
}

func param(in struct {
	Value any // want `noanyfields: field "Value" has type any`
}) {
	_ = in
}

func result() (out struct {
	Value any // want `noanyfields: field "Value" has type any`
}) {
	return
}

var lit = struct {
	Value any // want `noanyfields: field "Value" has type any`
}{Value: 1}

// --- valid --------------------------------------------------------------------

// A type parameter field is not any.
type Slot[T any] struct{ v T }

func (s Slot[T]) Value() T { return s.v }

// The imported named type owns its contract, embedded or not.
type Row struct {
	V driver.Value
	driver.Value
	Raw json.RawMessage
}

// A func-typed field is noanyparams' domain.
type Options struct {
	OnEvent func(e any)
}

// A container-typed field is noanycontainers' domain.
type Index struct {
	Meta map[string]any
	List []any
}

// Fields that state what they hold.
type User struct {
	ID   string
	Tags []string
}
