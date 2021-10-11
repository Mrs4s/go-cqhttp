// Package server 包含HTTP,WebSocket,反向WebSocket请求处理的相关函数与结构体
package server

import "github.com/Mrs4s/go-cqhttp/modules/servers"

// 注册
func init() {
	servers.Register("http", runHTTP)
	servers.Register("ws", runWSServer)
	servers.Register("ws-reverse", runWSClient)
	servers.Register("lambda", runLambda)
}
