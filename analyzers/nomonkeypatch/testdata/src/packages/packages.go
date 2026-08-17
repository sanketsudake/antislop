package packages

import (
	"bou.ke/monkey"                               // want `nomonkeypatch: import of bou\.ke/monkey patches functions at runtime`
	gomonkey "github.com/agiledragon/gomonkey/v2" // the packages option replaced the list, so this import is not reported
)

var (
	_ = monkey.Patch
	_ = gomonkey.ApplyFunc
)
