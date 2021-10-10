// Package servers provide servers register
package servers

import (
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

var svr = make(map[string]func(*coolq.CQBot, yaml.Node))

// Register 注册 Server
func Register(name string, proc func(*coolq.CQBot, yaml.Node)) {
	_, ok := svr[name]
	if ok {
		panic(name + " server has existed")
	}
	svr[name] = proc
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
}
