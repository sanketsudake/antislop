package terms

// --- invalid ---

type WidgetBox struct{} // want `nostructuralnames: name "WidgetBox" describes structure \("widget"\) rather than the domain role; rename it for what it represents`

func gizmoOf() int { return 0 } // want `nostructuralnames: name "gizmoOf" describes structure \("gizmo"\)`

// Several terms match, so the diagnostic names the first one in the list.
type WidgetGizmo struct{} // want `nostructuralnames: name "WidgetGizmo" describes structure \("widget"\)`

// --- valid ---

// The default term is no longer in the list.
type OrderShape struct{}

var _ = []any{WidgetBox{}, WidgetGizmo{}, OrderShape{}, gizmoOf()}
