package static

import (
	"embed"
)

//go:embed html db
var StaticFs embed.FS
