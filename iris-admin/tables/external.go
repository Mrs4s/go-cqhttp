package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/parameter"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

// GetExternalTable return the model from external data source.
func GetExternalTable(ctx *context.Context) (externalTable table.Table) {

	externalTable = table.NewDefaultTable(table.DefaultConfig())

	info := externalTable.GetInfo()
	info.AddField("ID", "id", db.Int).FieldSortable()
	info.AddField("Title", "title", db.Varchar)

	info.SetTable("external").
		SetTitle("Externals").
		SetDescription("Externals").
		SetGetDataFn(func(param parameter.Parameters) ([]map[string]interface{}, int) {
			return []map[string]interface{}{
				{
					"id":    10,
					"title": "this is a title",
				}, {
					"id":    11,
					"title": "this is a title2",
				}, {
					"id":    12,
					"title": "this is a title3",
				}, {
					"id":    13,
					"title": "this is a title4",
				},
			}, 10
		})

	formList := externalTable.GetForm()
	formList.AddField("ID", "id", db.Int, form.Default).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("Title", "title", db.Varchar, form.Text)

	formList.SetTable("external").SetTitle("Externals").SetDescription("Externals")

	detail := externalTable.GetDetail()

	detail.SetTable("external").
		SetTitle("Externals").
		SetDescription("Externals").
		SetGetDataFn(func(param parameter.Parameters) ([]map[string]interface{}, int) {
			return []map[string]interface{}{
				{
					"id":    10,
					"title": "this is a title",
				},
			}, 1
		})

	return
}
