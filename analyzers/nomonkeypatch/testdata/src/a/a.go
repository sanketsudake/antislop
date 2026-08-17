package a

import (
	"time"

	"bou.ke/monkey"                               // want `nomonkeypatch: import of bou\.ke/monkey patches functions at runtime; inject the dependency through an interface or a function parameter instead`
	gomonkey "github.com/agiledragon/gomonkey/v2" // want `nomonkeypatch: import of github\.com/agiledragon/gomonkey/v2 patches functions at runtime`
)

var (
	_ = monkey.Patch
	_ = gomonkey.ApplyFunc
)

// timeNow is the package-level function variable a_test.go reassigns.
var timeNow = time.Now

// verbose is not a function, so a test that sets it is not patching a call.
var verbose = false

// server takes its clock through a field: that is the seam this rule asks for.
type server struct {
	now func() time.Time
}

func (s server) at() time.Time { return s.now() }

// --- valid ---

// Assignment outside a test is ordinary package initialisation.
func init() { timeNow = time.Now }
