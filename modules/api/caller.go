// Package api implements the API route for servers.
package api

import (
	"github.com/tidwall/gjson"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/pkg/onebot"
)

//go:generate go run ./../../cmd/api-generator -pkg api -path=./../../coolq/api.go,./../../coolq/api_v12.go -o api.go

// Getter 参数获取
type Getter interface {
	Get(string) gjson.Result
}

// Handler 中间件
type Handler func(action string, spe *onebot.Spec, p Getter) global.MSG

// Caller api route caller
type Caller struct {
	bot      *coolq.CQBot
	handlers []Handler
}

// Call specific API
func (c *Caller) Call(action string, spec *onebot.Spec, p Getter) global.MSG {
	for _, fn := range c.handlers {
		if ret := fn(action, spec, p); ret != nil {
			return ret
		}
	}
	return c.call(action, spec, p)
}

// Use add handlers to the API caller
func (c *Caller) Use(middlewares ...Handler) {
	c.handlers = append(c.handlers, middlewares...)
}

// NewCaller create a new API caller
func NewCaller(bot *coolq.CQBot) *Caller {
	return &Caller{
		bot:      bot,
		handlers: make([]Handler, 0),
	}
}
