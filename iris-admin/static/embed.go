package static

import (
	"embed"
)

//go:embed html db js
var StaticFs embed.FS
