// Package static 静态资源库
package static

import (
	"embed"
)

// StaticFs 静态资源打包
//go:embed html db js qqface
var StaticFs embed.FS
