package gocq

import (
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/db"
)

// InitDB 初始化数据库
func InitDB() {
	db.Init()
	if err := db.Open(); err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
}
