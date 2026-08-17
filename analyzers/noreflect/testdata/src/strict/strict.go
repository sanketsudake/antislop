package strict

import "reflect"

// --- invalid ---

func equal(a, b int) bool {
	return reflect.DeepEqual(a, b) // want `noreflect: use of package reflect \(reflect\.DeepEqual\) bypasses static types; name the concrete types the code works on, or the small interface it needs`
}

func typePosition() bool {
	var t reflect.Type // want `noreflect: use of package reflect \(reflect\.Type\) bypasses static types`
	return t == nil
}

func stillReported(r struct{ Name string }) string {
	// Both rules fire: the package selector and the lookup by name.
	return reflect.ValueOf(r).FieldByName("Name").String() // want `noreflect: use of package reflect \(reflect\.ValueOf\) bypasses static types` `noreflect: reflect\.Value\.FieldByName reads a field by string`
}

// --- valid ---

// Nothing in this function touches package reflect.
func plain(a, b int) bool { return a == b }
