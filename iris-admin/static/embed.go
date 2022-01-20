package static

import (
	"embed"
)

//go:embed html db js qqface
var StaticFs embed.FS
