package common

import (
	"embed"
	"github.com/Mrs4s/go-cqhttp/iris-admin/static"
)

func GetStaticFs() embed.FS {
	return static.StaticFs
}
