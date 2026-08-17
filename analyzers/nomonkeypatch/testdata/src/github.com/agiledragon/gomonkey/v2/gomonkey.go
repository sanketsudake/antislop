// Package gomonkey is a stub of github.com/agiledragon/gomonkey/v2 for the fixtures.
package gomonkey

// Patches records the applied patches.
type Patches struct{}

// ApplyFunc replaces target with double at run time.
func ApplyFunc(target, double any) *Patches { return &Patches{} }
