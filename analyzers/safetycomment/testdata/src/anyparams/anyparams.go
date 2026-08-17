package anyparams

// With skip-declared-any set, an unchecked assertion on a parameter typed any
// is left to noanyparams, which reports the parameter itself.
func mustInt(v any) int { return v.(int) }

var raw any

func fromVar() int {
	return raw.(int) // want `safetycomment: unchecked type assertion has no SAFETY: justification`
}

type flagDef struct{ DefaultValue any }

func fieldUse(f flagDef) bool { return f.DefaultValue.(bool) }

func elemOfParam(m map[string]any) int { return m["k"].(int) }
