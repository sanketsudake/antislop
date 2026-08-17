package anyparams

import "database/sql/driver"

// With skip-declared-any set, narrowing a parameter typed any is left to
// noanyparams, which reports the parameter itself.
func kind(v any) string {
	switch v.(type) {
	case int:
		return "int"
	}
	if _, ok := v.(string); ok {
		return "string"
	}
	return "other"
}

func literal() {
	fn := func(x any) bool { _, ok := x.(int); return ok }
	fn(1)
}

// A closure narrowing the enclosing function's parameter is the same case.
func captured(v any) func() bool {
	return func() bool { _, ok := v.(int); return ok }
}

// Anything that is not a parameter typed any is still reported.
var raw any

func fromVar() bool {
	_, ok := raw.(int) // want `nonarrowany: comma-ok assertion on any`
	return ok
}

// An imported named type is not the parameter noanyparams reports.
func fromDriver(v driver.Value) bool {
	_, ok := v.(int64) // want `nonarrowany: comma-ok assertion on any`
	return ok
}

// A same-package field typed any is reported by noanyfields at the field;
// with skip-declared-any its uses are left to that finding.
type flagDef struct{ DefaultValue any }

func fieldUse(f flagDef) bool {
	_, ok := f.DefaultValue.(bool)
	return ok
}

// Elements of a same-package container of any are reported by
// noanycontainers at the declaration; with skip-declared-any their uses are
// left to that finding.
type call struct{ args map[string]any }

func (c call) str(name string) string {
	s, _ := c.args[name].(string)
	return s
}

func elemOfParam(m map[string]any) bool {
	_, ok := m["k"].(int)
	return ok
}
