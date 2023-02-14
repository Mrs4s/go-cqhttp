// Package main
package main

import (
	"github.com/Mrs4s/go-cqhttp/cmd/gocq"

	_ "github.com/Mrs4s/go-cqhttp/db/leveldb"   // leveldb 数据库支持
	_ "github.com/Mrs4s/go-cqhttp/modules/silk" // silk编码模块
	// 其他模块
	// _ "github.com/Mrs4s/go-cqhttp/db/sqlite3"   // sqlite3 数据库支持
	// _ "github.com/Mrs4s/go-cqhttp/db/mongodb"    // mongodb 数据库支持
	// _ "github.com/Mrs4s/go-cqhttp/modules/pprof" // pprof 性能分析
)

func main() {
	gocq.Main()
}
