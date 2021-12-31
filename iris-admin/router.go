package iris_admin

import (
	adapter "github.com/GoAdminGroup/go-admin/adapter/iris"
	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/Mrs4s/go-cqhttp/iris-admin/app/login"
	"github.com/Mrs4s/go-cqhttp/iris-admin/pages"
	"github.com/kataras/iris/v12"
)

func makeRouter(eng *engine.Engine, app *iris.Application) {
	eng.HTML("GET", "/admin", pages.DashboardPage)
	eng.HTML("GET", "/admin/form", pages.GetFormContent)
	eng.HTML("GET", "/admin/table", pages.GetTableContent)
	app.Get("/", func(ctx iris.Context) {
		ctx.Redirect("/admin")
	})
	app.Get("/test", adapter.Content(login.CheckQQlogin))
	app.HandleDir("/uploads", "./uploads", iris.DirOptions{
		IndexName: "/index.html",
		Gzip:      true,
		ShowList:  false,
	})

}
