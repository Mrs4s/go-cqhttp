package tables

import (
	"fmt"
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	form2 "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/icon"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/action"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	selection "github.com/GoAdminGroup/go-admin/template/types/form/select"
	editType "github.com/GoAdminGroup/go-admin/template/types/table"
)

// GetUserTable return the model of table user.
func GetUserTable(ctx *context.Context) (userTable table.Table) {

	userTable = table.NewDefaultTable(table.Config{
		Driver:     db.DriverSqlite,
		CanAdd:     true,
		Editable:   true,
		Deletable:  true,
		Exportable: true,
		Connection: table.DefaultConnectionName,
		PrimaryKey: table.PrimaryKey{
			Type: db.Int,
			Name: table.DefaultPrimaryKeyName,
		},
	})

	info := userTable.GetInfo().SetFilterFormLayout(form.LayoutThreeCol)
	info.AddField("ID", "id", db.Int).FieldSortable()
	info.AddField("Name", "name", db.Varchar).FieldEditAble(editType.Text).
		FieldFilterable(types.FilterType{Operator: types.FilterOperatorLike})
	info.AddField("Gender", "gender", db.Tinyint).FieldDisplay(func(model types.FieldModel) interface{} {
		if model.Value == "0" {
			return "men"
		}
		if model.Value == "1" {
			return "women"
		}
		return "unknown"
	}).FieldEditAble(editType.Switch).FieldEditOptions(types.FieldOptions{
		{Value: "0", Text: "👨"},
		{Value: "1", Text: "👩"},
	}).FieldFilterable(types.FilterType{FormType: form.SelectSingle}).FieldFilterOptions(types.FieldOptions{
		{Value: "0", Text: "men"},
		{Value: "1", Text: "women"},
	})
	info.AddColumn("personality", func(value types.FieldModel) interface{} {
		return "handsome"
	})
	info.AddColumnButtons("see more", types.GetColumnButton("more", icon.Info,
		action.PopUp("/see/more/example", "Detail", func(ctx *context.Context) (success bool, msg string, data interface{}) {
			return true, "ok", "<h1>Detail</h1><p>balabala</p><p>this feature will be released in v1.2.7</p>"
		})))
	info.AddField("Phone", "phone", db.Varchar).FieldFilterable()
	info.AddField("City", "city", db.Varchar).FieldFilterable()
	info.AddField("Avatar", "avatar", db.Varchar).FieldDisplay(func(value types.FieldModel) interface{} {
		return template.Default().Image().
			SetSrc(`//quick.go-admin.cn/demo/assets/dist/img/gopher_avatar.png`).
			SetHeight("120").SetWidth("120").WithModal().GetContent()
	})
	info.AddField("CreatedAt", "created_at", db.Timestamp).
		FieldFilterable(types.FilterType{FormType: form.DatetimeRange})
	info.AddField("UpdatedAt", "updated_at", db.Timestamp).FieldEditAble(editType.Datetime)

	info.AddActionButton("google", action.Jump("https://google.com"))
	info.AddActionButton("audit", action.Ajax("/admin/audit",
		func(ctx *context.Context) (success bool, msg string, data interface{}) {
			return true, "success", ""
		}))
	info.AddActionButton("Preview", action.PopUp("/admin/preview", "Preview",
		func(ctx *context.Context) (success bool, msg string, data interface{}) {
			return true, "", "<h2>hello world</h2>"
		}))
	info.AddButton("google", icon.Google, action.Jump("https://google.com"))
	info.AddButton("popup", icon.Terminal, action.PopUp("/admin/popup", "Popup Example",
		func(ctx *context.Context) (success bool, msg string, data interface{}) {
			return true, "", "<h2>hello world</h2>"
		}))
	info.AddButton("iframe", icon.Tv, action.PopUpWithIframe("/admin/iframe", "Iframe Example",
		action.IframeData{Src: "/admin/info/profile/new"}, "900px", "480px"))
	info.AddButton("ajax", icon.Android, action.Ajax("/admin/ajax",
		func(ctx *context.Context) (success bool, msg string, data interface{}) {
			return true, "success", ""
		}))
	info.AddSelectBox("gender", types.FieldOptions{
		{Value: "0", Text: "men"},
		{Value: "1", Text: "women"},
	}, action.FieldFilter("gender"))

	info.SetTable("users").SetTitle("Users").SetDescription("Users")

	formList := userTable.GetForm()
	formList.AddField("ID", "id", db.Int, form.Default).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("Ip", "ip", db.Varchar, form.Text)
	formList.AddField("Name", "name", db.Varchar, form.Text)
	formList.AddField("Gender", "gender", db.Tinyint, form.Radio).
		FieldOptions(types.FieldOptions{
			{Text: "men", Value: "0"},
			{Text: "women", Value: "1"},
		}).FieldDefault("0")
	formList.AddField("Phone", "phone", db.Varchar, form.Text)
	formList.AddField("Country", "country", db.Tinyint, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "China", Value: "0"},
			{Text: "America", Value: "1"},
			{Text: "England", Value: "2"},
			{Text: "Canada", Value: "3"},
		}).FieldDefault("0").FieldOnChooseAjax("city", "/choose/country",
		func(ctx *context.Context) (bool, string, interface{}) {
			country := ctx.FormValue("value")
			var data = make(selection.Options, 0)
			switch country {
			case "0":
				data = selection.Options{
					{Text: "Beijing", ID: "beijing"},
					{Text: "ShangHai", ID: "shangHai"},
					{Text: "GuangZhou", ID: "guangZhou"},
					{Text: "ShenZhen", ID: "shenZhen"},
				}
			case "1":
				data = selection.Options{
					{Text: "Los Angeles", ID: "los angeles"},
					{Text: "Washington, dc", ID: "washington, dc"},
					{Text: "New York", ID: "new york"},
					{Text: "Las Vegas", ID: "las vegas"},
				}
			case "2":
				data = selection.Options{
					{Text: "London", ID: "london"},
					{Text: "Cambridge", ID: "cambridge"},
					{Text: "Manchester", ID: "manchester"},
					{Text: "Liverpool", ID: "liverpool"},
				}
			case "3":
				data = selection.Options{
					{Text: "Vancouver", ID: "vancouver"},
					{Text: "Toronto", ID: "toronto"},
				}
			default:
				data = selection.Options{
					{Text: "Beijing", ID: "beijing"},
					{Text: "ShangHai", ID: "shangHai"},
					{Text: "GuangZhou", ID: "guangZhou"},
					{Text: "ShenZhen", ID: "shenZhen"},
				}
			}
			return true, "ok", data
		})
	formList.AddField("City", "city", db.Varchar, form.SelectSingle).
		FieldOptionInitFn(func(val types.FieldModel) types.FieldOptions {
			return types.FieldOptions{
				{Value: val.Value, Text: val.Value, Selected: true},
			}
		})
	formList.AddField("Custom Field", "role", db.Varchar, form.Text).
		FieldPostFilterFn(func(value types.PostFieldModel) interface{} {
			fmt.Println("user custom field", value)
			return ""
		})

	formList.AddField("UpdatedAt", "updated_at", db.Timestamp, form.Default).FieldNotAllowAdd()
	formList.AddField("CreatedAt", "created_at", db.Timestamp, form.Default).FieldNotAllowAdd()

	userTable.GetForm().SetTabGroups(types.
		NewTabGroups("id", "ip", "name", "gender", "country", "city").
		AddGroup("phone", "role", "created_at", "updated_at")).
		SetTabHeaders("profile1", "profile2")

	formList.SetTable("users").SetTitle("Users").SetDescription("Users")

	formList.SetPostHook(func(values form2.Values) error {
		fmt.Println("userTable.GetForm().PostHook", values)
		return nil
	})

	return
}
