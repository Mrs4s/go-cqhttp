package common

import (
	"embed"

	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/static"
)

// GetStaticFs 获取静态资源打包对象
func GetStaticFs() embed.FS {
	return static.StaticFs
}
