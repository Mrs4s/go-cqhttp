// Package models 数据库操作库
package models

import (
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/jinzhu/gorm"
)

var (
	orm *gorm.DB
	err error
)

// Init 初始化db
func Init(c db.Connection) {
	orm, err = gorm.Open("sqlite3", c.GetDB("default"))
	if err != nil {
		panic("initialize orm failed")
	}
}
