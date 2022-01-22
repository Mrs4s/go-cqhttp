package irisadmin

import (
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	_ "github.com/GoAdminGroup/go-admin/adapter/iris" // web framework adapter
	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/db"
	_ "github.com/GoAdminGroup/go-admin/modules/db/drivers/sqlite" // sql driver
	"github.com/GoAdminGroup/go-admin/modules/language"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/chartjs"
	_ "github.com/GoAdminGroup/themes/adminlte" // ui theme
	"github.com/kataras/iris/v12"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/app/info"
	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/app/qq"
	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/models"
	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/tables"
	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/global/terminal"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/server"
)

func setup() {
	// base.Parse()
	if !base.FastStart && terminal.RunningByDoubleClick() {
		err := terminal.NoMoreDoubleClick()
		if err != nil {
			log.Errorf("遇到错误: %v", err)
			time.Sleep(time.Second * 5)
		}
		return
	}
	switch {
	case base.LittleH:
		base.Help()
	case base.LittleD:
		server.Daemon()
	case base.LittleWD != "":
		base.ResetWorkingDir()
	}
}

// App 网站服务的对象
type App struct {
	Login *qq.Dologin
	Info  *info.Info
}

var appInterface *App

func initApp() {
	appInterface = &App{
		Login: qq.NewDologin(),
		Info:  info.NewInfo(),
	}
}

// StartServer 启动web服务
func StartServer() {
	setup()
	goAdmin()
	app := iris.New()

	eng := engine.Default()

	template.AddComp(chartjs.NewChart())

	cfg := &config.Config{
		Databases: config.DatabaseList{
			"default": {
				File:       "./data/admin.db",
				MaxIdleCon: 50,
				MaxOpenCon: 150,
				Driver:     db.DriverSqlite,
			},
		},
		UrlPrefix: "admin",
		IndexUrl:  "/",
		Debug:     false,
		Language:  language.CN,
		AppID:     "lKZy349x0SWF",
		Theme:     "adminlte",
		Store: config.Store{
			Path:   "./data/uploads",
			Prefix: "uploads",
		},
		Title:           "go-cqhttp",
		Logo:            "go-cqhttp",
		MiniLogo:        "qocq",
		LoginUrl:        "/login",
		AccessLogPath:   "./data/logs/access.log",
		ErrorLogPath:    "./data/logs/error.log",
		InfoLogPath:     "./data/logs/info.log",
		ColorScheme:     "skin-black",
		SessionLifeTime: 7200,
		FileUploadEngine: config.FileUploadEngine{
			Name: "local",
		},
		LoginTitle:    "go-cqhttp",
		LoginLogo:     "go-cqhttp",
		AuthUserTable: "goadmin_users",
		AssetRootPath: "./public/",
		URLFormat: config.URLFormat{
			Info:       "/info/:__prefix",
			Detail:     "/info/:__prefix/detail",
			Create:     "/new/:__prefix",
			Delete:     "/delete/:__prefix",
			Export:     "/export/:__prefix",
			Edit:       "/edit/:__prefix",
			ShowEdit:   "/info/:__prefix/edit",
			ShowCreate: "/info/:__prefix/new",
			Update:     "/update/:__prefix",
		},
	}
	if err := eng.AddConfig(cfg).
		AddGenerators(tables.Generators).
		Use(app); err != nil {
		panic(err)
	}

	models.Init(eng.SqliteConnection())

	initApp()
	makeRouter(eng, app) // 注册路由
	go func() {
		_ = app.Run(iris.Addr(":8080"))
	}()
	go appInterface.Login.DoLoginBackend()
	appInterface.Login.AutoLoginCommon()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	eng.SqliteConnection().Close()
}

func goAdmin() {
	dst := "./data/admin.db"
	if !global.PathExists(dst) {
		_ = os.MkdirAll(filepath.Dir(dst), 0775)
		fs := common.GetStaticFs()
		src, _ := fs.Open("db/admin.db")
		dst, _ := os.Create(dst)
		_, _ = io.Copy(dst, src)
	}
}
