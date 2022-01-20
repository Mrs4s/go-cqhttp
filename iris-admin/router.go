package iris_admin

import (
	"fmt"
	adapter "github.com/GoAdminGroup/go-admin/adapter/iris"
	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/kataras/iris/v12"
)

func makeRouter(eng *engine.Engine, app *iris.Application) {
	//eng.HTML("get", "/admin", pages.GetIndex)
	app.Get("/admin", adapter.Content(appInterface.Login.QqInfo))
	app.Get("/", func(ctx iris.Context) {
		//默认转跳地址
		ctx.Redirect("/admin/qq/info") //配置页面
	})
	////自定义中间件
	//app.WrapRouter(func(w http.ResponseWriter, r *http.Request, firstNextIsTheRouter http.HandlerFunc) {
	//	path := r.URL.Path
	//	if strings.HasPrefix(path, "/qq") {
	//		ctx := app.ContextPool.Acquire(w, r)
	//		adapter.Content(login.CheckQQlogin)
	//		app.ContextPool.Release(ctx)
	//		return
	//	}
	//	firstNextIsTheRouter.ServeHTTP(w, r)
	//})
	app.Get("/admin/qq/checkconfig", appInterface.Login.CheckConfig) //登录前调用，校验配置文件等信息。
	app.Post("/qq/do_encrypt_key_input", adapter.Content(appInterface.Login.DoEncryptKeyInput))
	app.Get("/qq/login", appInterface.Login.NomalLogin) //正常登录
	//app.Get("/qq/qrlogin", appInterface.Login.QrloginHtml) //二维码方式登录
	app.Get("/qq/qrlogin", adapter.Content(appInterface.Login.DoQrlogin)) //二维码方式登录
	app.Any("/qq/do_qrlogin", adapter.Content(appInterface.Login.DoQrlogin))
	app.Get("/qq/captcha_input", adapter.Content(appInterface.Login.CaptchaInput))
	app.Get("/qq/loginsuccess", appInterface.Login.LoginSuccess)
	app.Get("/admin/qq/info", adapter.Content(appInterface.Login.QqInfo))
	eng.HTML("any", "/qq/sms_input", appInterface.Login.SmsInput)
	eng.HTML("get", "/admin/qq/encrypt_key_input", appInterface.Login.EncryptPasswordEnterWeb)
	eng.HTML("get", "/admin/qq/session_token_select", appInterface.Login.SessionTokenWeb)
	eng.HTML("get", "/admin/qq/weblog", appInterface.Login.WebLog)
	app.Get("/admin/qq/shutdown", appInterface.Login.Shutdown)
	app.Get("/admin/qq/friendlist", adapter.Content(appInterface.Login.MemberList))
	app.Get("/admin/qq/grouplist", adapter.Content(appInterface.Login.GroupList))
	app.Get("/admin/qq/leavegroup", appInterface.Login.LeaveGroup)
	app.Get("/admin/qq/getgroupdetail", adapter.Content(appInterface.Login.GetGroupDetal))
	app.Get("/admin/qq/deletefriend", appInterface.Login.DeleteFriend)
	app.Get("/admin/qq/getfrienddetail", adapter.Content(appInterface.Login.GetFriendDetal))
	app.Get("/admin/qq/getmsglist", adapter.Content(appInterface.Login.GetMsgList))
	app.Get("/admin/qq/getgroupmsglist", adapter.Content(appInterface.Login.GetGroupMsgList))
	app.Any("/admin/qq/getmsglistforajax", appInterface.Login.GetMsgListAjaxHtml)
	app.Post("/admin/qq/sendmsg", appInterface.Login.SendMsg)
	app.Get("/admin/qq/guildlist", adapter.Content(appInterface.Login.GuildList))
	app.Get("/admin/qq/channellist", adapter.Content(appInterface.Login.ChannelList))
	app.Get("/admin/qq/help", func(ctx iris.Context) {
		p := "html/help.md"
		fs := common.GetStaticFs()
		data, _ := fs.ReadFile(p)
		ctx.Markdown(data)
	})
	app.Get("/admin/qq/faceimg/{name:file}", func(ctx iris.Context) {
		name := ctx.Params().Get("name")
		fs := common.GetStaticFs()
		data, err := fs.ReadFile("qqface/" + name)
		if err != nil {
			return
		}
		ctx.Header("Content-Type", "image/gif")
		ctx.Header("Content-Disposition", fmt.Sprintf(`filename="%s"`, name))
		ctx.Write(data)
	})
	app.HandleDir("/uploads", "./uploads", iris.DirOptions{
		IndexName: "/index.html",
		Gzip:      true,
		ShowList:  false,
	})

}
