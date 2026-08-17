package stubbing

import (
	"time"

	"bou.ke/monkey" // want `nomonkeypatch: import of bou\.ke/monkey patches functions at runtime`
)

var _ = monkey.Patch

var timeNow = time.Now
