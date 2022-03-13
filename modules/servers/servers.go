// Package servers provide servers register
package servers

import (
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

var (
	svr      = make(map[string]func(*coolq.CQBot, yaml.Node))
	nocfgsvr = make(map[string]func(*coolq.CQBot))
)

// Register 注册 Server
func Register(name string, proc func(*coolq.CQBot, yaml.Node)) {
	_, ok := svr[name]
	if ok {
		panic(name + " server has existed")
	}
	svr[name] = proc
}

// RegisterCustom 注册无需 config 的自定义 Server
func RegisterCustom(name string, proc func(*coolq.CQBot)) {
	_, ok := nocfgsvr[name]
	if ok {
		panic(name + " server has existed")
	}
	nocfgsvr[name] = proc
}

// Run 运行所有svr
func Run(bot *coolq.CQBot) {
	for _, l := range base.Servers {
		for name, conf := range l {
			if fn, ok := svr[name]; ok {
				go fn(bot, conf)
			}
		}
	}
	for _, fn := range nocfgsvr {
		go fn(bot)
	}
	base.Servers = nil
}
