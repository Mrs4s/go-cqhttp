package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	editType "github.com/GoAdminGroup/go-admin/template/types/table"
)

// GetPostsTable return the model of table posts.
func GetPostsTable(ctx *context.Context) (postsTable table.Table) {

	postsTable = table.NewDefaultTable(table.DefaultConfigWithDriver("sqlite"))

	info := postsTable.GetInfo()
	info.AddField("ID", "id", db.Int).FieldSortable()
	info.AddField("Title", "title", db.Varchar)
	info.AddField("AuthorID", "author_id", db.Int).FieldDisplay(func(value types.FieldModel) interface{} {
		return template.Default().
			Link().
			SetURL("/admin/info/authors/detail?__goadmin_detail_pk=" + value.Value).
			SetContent(template.HTML(value.Value)).
			OpenInNewTab().
			SetTabTitle(template.HTML("Author Detail(" + value.Value + ")")).
			GetContent()
	})
	info.AddField("AuthorName", "name", db.Varchar).FieldDisplay(func(value types.FieldModel) interface{} {
		first, _ := value.Row["authors_goadmin_join_first_name"].(string)
		last, _ := value.Row["authors_goadmin_join_last_name"].(string)
		return first + " " + last
	})
	info.AddField("AuthorFirstName", "first_name", db.Varchar).FieldJoin(types.Join{
		Field:     "author_id",
		JoinField: "id",
		Table:     "authors",
	}).FieldHide()
	info.AddField("AuthorLastName", "last_name", db.Varchar).FieldJoin(types.Join{
		Field:     "author_id",
		JoinField: "id",
		Table:     "authors",
	}).FieldHide()
	info.AddField("Description", "description", db.Varchar)
	info.AddField("Content", "content", db.Varchar).FieldEditAble(editType.Textarea)
	info.AddField("Date", "date", db.Varchar)

	info.SetTable("posts").SetTitle("Posts").SetDescription("Posts")

	formList := postsTable.GetForm()
	formList.AddField("ID", "id", db.Int, form.Default).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("Title", "title", db.Varchar, form.Text)
	formList.AddField("Description", "description", db.Varchar, form.Text)
	formList.AddField("Content", "content", db.Varchar, form.RichText).FieldEnableFileUpload()
	formList.AddField("Date", "date", db.Varchar, form.Datetime)

	formList.EnableAjax("Success", "Fail")

	formList.SetTable("posts").SetTitle("Posts").SetDescription("Posts")

	return
}
