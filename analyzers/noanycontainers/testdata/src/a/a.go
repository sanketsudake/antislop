package a

import (
	"database/sql/driver"
	"encoding/json"
	"text/template"
)

type Escape = any

// --- invalid: containers whose key or value is the empty interface -------------

var m = map[string]any{} // want `noanycontainers: map\[string\]any erases the value types`

type Document map[string]any // want `noanycontainers: map\[string\]any erases the value types`

type Record struct {
	Meta map[string]interface{} // want `noanycontainers: map\[string\]interface\{\} erases the value types`
}

func accept(in map[string]any) { _ = in } // want `noanycontainers: map\[string\]any erases the value types`

func produce() map[string]any { return nil } // want `noanycontainers: map\[string\]any erases the value types`

func made() { _ = make(map[string]interface{}) } // want `noanycontainers: map\[string\]interface\{\} erases the value types`

var converted = map[string]any(nil) // want `noanycontainers: map\[string\]any erases the value types`

// An untyped key states nothing about what identifies an entry.
var byKey = map[any]int{} // want `noanycontainers: map\[any\]int uses an untyped key`

// A same-package alias of any is still any.
var escaped = map[string]Escape{} // want `noanycontainers: map\[string\]Escape erases the value types`

// Only the direct key or value type counts, so the inner map is the offender.
var deep map[string]map[string]any // want `noanycontainers: map\[string\]any erases the value types`

// The outermost offending node is reported once: the inner map stays silent.
var nested map[any]map[string]any // want `noanycontainers: map\[any\]map\[string\]any uses an untyped key`

// A map nested in slices is reported at the map.
var lists [][]map[string]any // want `noanycontainers: map\[string\]any erases the value types`

// --- valid --------------------------------------------------------------------

// A type parameter value is not any.
func generic[T any](in map[string]T) map[string]T { return in }

// The imported named type owns its contract.
var values map[string]driver.Value

// An imported named map type is not a map type expression here.
var funcs template.FuncMap

var raw json.RawMessage

// Slices, arrays and channels are only reported with the slices option.
var anySlice []any

var anyArray [3]any

var anyChan chan any

// The direct value type is a slice, so the map is not the offender, and the
// slice is only reported with the slices option.
var mixed map[string][]any

func variadic(args ...any) { _ = args }

// Containers that state what they hold.
type Counts map[string]int

var users = map[string]Record{}
