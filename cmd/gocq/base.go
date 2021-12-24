package gocq

import (
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/server"
)

// InitBase 解析 flags 与配置文件到 base
func InitBase() {
	base.Parse()
	base.Init()
	switch {
	case base.LittleH:
		base.Help()
	case base.LittleD:
		server.Daemon()
	case base.LittleWD != "":
		base.ResetWorkingDir()
	}
}
