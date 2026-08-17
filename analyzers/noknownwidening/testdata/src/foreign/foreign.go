// Package foreign stands in for a third-party package that owns an
// empty-interface contract of its own (database/sql/driver.Value).
package foreign

// Value is another package's empty-interface contract.
type Value interface{}
