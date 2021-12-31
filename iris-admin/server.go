package iris_admin

import (
	_ "github.com/GoAdminGroup/go-admin/adapter/iris" // web framework adapter
	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/db"
	_ "github.com/GoAdminGroup/go-admin/modules/db/drivers/sqlite" // sql driver
	"github.com/GoAdminGroup/go-admin/modules/language"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/chartjs"
	_ "github.com/GoAdminGroup/themes/adminlte" // ui theme
	"github.com/Mrs4s/go-cqhttp/iris-admin/models"
	"github.com/Mrs4s/go-cqhttp/iris-admin/tables"
	"github.com/kataras/iris/v12"
	"os"
	"os/signal"
)

func StartServer() {
	app := iris.New()

	eng := engine.Default()

	template.AddComp(chartjs.NewChart())

	cfg := &config.Config{
		Databases: config.DatabaseList{
			"default": {
				File: "./admin.db",
				MaxIdleCon: 50,
				MaxOpenCon: 150,
				Driver:     db.DriverSqlite,
			},
		},
		UrlPrefix: "admin",
		IndexUrl:  "/",
		Debug:     true,
		Language:  language.CN,
		AppID: "lKZy349x0SWF",
		Theme: "adminlte",
		Store: config.Store{
			Path: "./uploads",
			Prefix: "uploads",
		},
		Title: "go-cqhttp",
		Logo: "go-cqhttp",
		MiniLogo: "qocq",
		LoginUrl: "/login",
		AccessLogPath:  "./logs/access.log",
		ErrorLogPath: "./logs/error.log",
		InfoLogPath: "./logs/info.log",
		ColorScheme: "skin-black",
		SessionLifeTime: 7200,
		FileUploadEngine: config.FileUploadEngine{
			Name: "local",
		},
		LoginTitle: "go-cqhttp",
		LoginLogo: "go-cqhttp",
		AuthUserTable: "goadmin_users",
		AssetRootPath: "./public/",
		URLFormat: config.URLFormat{
			Info: "/info/:__prefix",
			Detail:  "/info/:__prefix/detail",
			Create:  "/new/:__prefix",
			Delete: "/delete/:__prefix",
			Export: "/export/:__prefix",
			Edit: "/edit/:__prefix",
			ShowEdit: "/info/:__prefix/edit",
			ShowCreate: "/info/:__prefix/new",
			Update: "/update/:__prefix",
		},
	}
	if err := eng.AddConfig(cfg).
		AddGenerators(tables.Generators).
		AddGenerator("external", tables.GetExternalTable).
		Use(app); err != nil {
		panic(err)
	}

	models.Init(eng.SqliteConnection())
	makeRouter(eng, app) //注册路由

	go func() {
		_ = app.Run(iris.Addr(":8080"))
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	eng.SqliteConnection().Close()
}
