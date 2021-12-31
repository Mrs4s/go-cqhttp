package main

import (
	_ "github.com/GoAdminGroup/go-admin/adapter/iris"              // web framework adapter
	_ "github.com/GoAdminGroup/go-admin/modules/db/drivers/sqlite" // sql driver
	_ "github.com/GoAdminGroup/themes/adminlte"                    // ui theme
	_ "github.com/Mrs4s/go-cqhttp/db/leveldb"                      // leveldb
	iris_admin "github.com/Mrs4s/go-cqhttp/iris-admin"
	_ "github.com/Mrs4s/go-cqhttp/modules/mime"  // mime检查模块
	_ "github.com/Mrs4s/go-cqhttp/modules/pprof" // pprof 性能分析
	_ "github.com/Mrs4s/go-cqhttp/modules/silk"  // silk编码模块
)

func main() {
	iris_admin.StartServer()
	//gocq.Main()
}
