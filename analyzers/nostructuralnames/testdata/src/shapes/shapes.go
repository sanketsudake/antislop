package shapes // want `nostructuralnames: name "shapes" describes structure \("shape"\) rather than the domain role; rename it for what it represents`

// Order carries no structural term, so only the package clause is reported.
type Order struct{ ID string }
