package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/auth"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/models"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

var user models.UserModel

func GetQQConfigTable(ctx *context.Context) table.Table {
	// 获取登录用户模型
	user = auth.Auth(ctx)
	qqconfig := table.NewDefaultTable(table.DefaultConfigWithDriver("sqlite"))
	info := qqconfig.GetInfo()
	isAdmin := false
	for _, v := range user.Roles {
		if v.Id == 1 {
			isAdmin = true
		}
	}
	info.HideNewButton()
	info.HideDeleteButton()
	info.HideExportButton()
	info.HideFilterButton()
	info.HideRowSelector()
	if !isAdmin {
		info.HideEditButton()
	}

	info.AddField("Id", "id", db.Int).FieldHide()
	info.AddField("参数key", "key", db.Varchar)
	info.AddField("配置说明", "name", db.Varchar)
	info.AddField("配置的值", "value", db.Text)
	info.AddField("创建时间", "created_at", db.Timestamp).
		FieldSortable()
	info.AddField("更新时间", "updated_at", db.Timestamp).
		FieldSortable()
	info.SetTable("qq_config").SetTitle("qq config").SetDescription("qq机器人的配置信息")
	info.SetSortAsc().SetSortField("id").SetDefaultPageSize(50)
	formList := qqconfig.GetForm()
	formList.AddField("参数key", "key", db.Varchar, form.Text).FieldDisplayButCanNotEditWhenUpdate()
	formList.AddField("配置说明", "name", db.Text, form.Text).FieldDisplayButCanNotEditWhenUpdate()
	formList.AddField("配置的值", "value", db.Text, form.Text)
	formList.AddField("创建时间", "created_at", db.Timestamp, form.Datetime).
		FieldHide().FieldNowWhenInsert()
	formList.AddField("更新时间", "updated_at", db.Timestamp, form.Datetime).
		FieldHide().FieldNowWhenUpdate()
	formList.SetTable("qq_config").SetTitle("qq config").SetDescription("qq机器人的配置信息")
	return qqconfig
}
