package a

import "reflect"

type Record struct {
	Name string
}

func (Record) Greet() string { return "hi" }

// lookup is a same-named method on a type of our own: the compiler still
// checks it, so it is not a reflect escape hatch.
type lookup struct{}

func (lookup) FieldByName(string) string  { return "" }
func (lookup) MethodByName(string) string { return "" }

// --- invalid ---

func fieldByName(r Record) string {
	return reflect.ValueOf(r).FieldByName("Name").String() // want `noreflect: reflect\.Value\.FieldByName reads a field by string with no compile-time evidence; use a typed accessor or decode into a struct at the boundary`
}

func fieldByNameFunc(r Record) reflect.Value {
	return reflect.ValueOf(r).FieldByNameFunc(func(string) bool { return true }) // want `noreflect: reflect\.Value\.FieldByNameFunc reads a field by string`
}

func methodCall(r Record) []reflect.Value {
	return reflect.ValueOf(r).MethodByName("Greet").Call(nil) // want `noreflect: reflect\.Value\.MethodByName reads a field by string` `noreflect: reflect\.Value\.Call invokes a function with no compile-time evidence about its signature; call it through a typed func value or interface`
}

func callSlice(fn reflect.Value, args []reflect.Value) []reflect.Value {
	return fn.CallSlice(args) // want `noreflect: reflect\.Value\.CallSlice invokes a function with no compile-time evidence`
}

func mapIndex(m map[string]int, key string) reflect.Value {
	return reflect.ValueOf(m).MapIndex(reflect.ValueOf(key)) // want `noreflect: reflect\.Value\.MapIndex reads a field by string`
}

func typeMethodByName(r Record) (reflect.Method, bool) {
	return reflect.TypeOf(r).MethodByName("Greet") // want `noreflect: reflect\.Type\.MethodByName reads a field by string`
}

func typeFieldByName(r Record) (reflect.StructField, bool) {
	return reflect.TypeOf(r).FieldByName("Name") // want `noreflect: reflect\.Type\.FieldByName reads a field by string`
}

// --- valid ---

// The field is named in the type, so the compiler checks the access.
func name(r Record) string { return r.Name }

// reflect.DeepEqual takes and returns typed values; only strict reports it.
func equal(a, b Record) bool { return reflect.DeepEqual(a, b) }

// Kind is not a lookup by name.
func kind(r Record) reflect.Kind { return reflect.TypeOf(r).Kind() }

// Field and Index address a member the compiler can still see the shape of.
func first(r Record) reflect.Value { return reflect.ValueOf(r).Field(0) }

// Methods of the same name on other types are ordinary methods.
func ownLookup(l lookup) string { return l.FieldByName("Name") + l.MethodByName("Greet") }
