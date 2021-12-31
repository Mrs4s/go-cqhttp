package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"path/filepath"
	"strings"
)

func GetProfileTable(ctx *context.Context) table.Table {

	profile := table.NewDefaultTable(table.DefaultConfigWithDriver("sqlite"))

	info := profile.GetInfo().HideFilterArea()
	info.AddField("ID", "id", db.Int).FieldFilterable()
	info.AddField("UUID", "uuid", db.Varchar).FieldCopyable()
	info.AddField("Pass", "pass", db.Tinyint).FieldBool("1", "0")
	info.AddField("Photos", "photos", db.Varchar).FieldCarousel(func(value string) []string {
		return strings.Split(value, ",")
	}, 150, 100)
	info.AddField("Finish State", "finish_state", db.Tinyint).
		FieldDisplay(func(value types.FieldModel) interface{} {
			if value.Value == "0" {
				return "Step 1"
			}
			if value.Value == "1" {
				return "Step 2"
			}
			if value.Value == "2" {
				return "Step 3"
			}
			return "Unknown"
		}).
		FieldDot(map[string]types.FieldDotColor{
			"Step 1": types.FieldDotColorDanger,
			"Step 2": types.FieldDotColorInfo,
			"Step 3": types.FieldDotColorPrimary,
		}, types.FieldDotColorDanger)
	info.AddField("Progress", "finish_progress", db.Int).FieldProgressBar()
	info.AddField("Resume", "resume", db.Varchar).
		FieldDisplay(func(value types.FieldModel) interface{} {
			return filepath.Base(value.Value)
		}).
		FieldDownLoadable("http://yinyanghu.github.io/files/")
	info.AddField("FileSize", "resume_size", db.Int).FieldFileSize()

	info.SetTable("profile").SetTitle("Profile").SetDescription("Profile")

	formList := profile.GetForm()
	formList.AddField("UUID", "uuid", db.Varchar, form.Text)
	formList.AddField("Photos", "photos", db.Varchar, form.Text)
	formList.AddField("Resume", "resume", db.Varchar, form.Text)
	formList.AddField("FileSize", "resume_size", db.Int, form.Number)
	formList.AddField("Finish State", "finish_state", db.Tinyint, form.Number)
	formList.AddField("Progress", "finish_progress", db.Int, form.Number)
	formList.AddField("Pass", "pass", db.Tinyint, form.Number)

	formList.SetTable("profile").SetTitle("Profile").SetDescription("Profile")

	return profile
}
