package login

import (
	"fmt"
	tmpl "github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/icon"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/themes/adminlte/components/infobox"
	"github.com/kataras/iris/v12"
	"strconv"
)

// 查看当前的qq状态
func (l *Dologin) QqInfo(ctx iris.Context) (types.Panel, error) {
	components := tmpl.Default()
	colComp := components.Col()
	/**************************
	 * Info Box
	/**************************/

	infobox1 := infobox.New().
		SetText("QQ账号").
		SetColor("aqua").
		SetNumber(tmpl.HTML(strconv.FormatInt(l.Cli.Uin, 10))).
		SetIcon("ion-ios-gear-outline").
		GetContent()
	status := func(l *Dologin) string {
		if l.Cli.Online {
			return "QQ在线"
		}
		return "QQ离线"
	}(l)
	infobox2 := infobox.New().
		SetText("QQ状态").
		SetColor("red").
		SetNumber(tmpl.HTML(status)).
		SetIcon(icon.GooglePlus).
		GetContent()
	serverStatus := func(l *Dologin) string {
		if l.Status {
			return "server已启动"
		}
		return "server服务未启动"
	}(l)
	infobox3 := infobox.New().
		SetText("Server状态").
		SetColor("green").
		SetNumber(tmpl.HTML(serverStatus)).
		SetIcon("ion-ios-cart-outline").
		GetContent()

	infobox4 := infobox.New().
		SetText("错误信息").
		SetColor("yellow").
		SetNumber(tmpl.HTML(fmt.Sprintf("code:%d ,msg:%s", l.ErrMsg.Code, l.ErrMsg.Msg))).
		SetIcon("ion-ios-people-outline"). // svg is ok
		GetContent()

	var size = types.SizeMD(3).SM(6).XS(12)
	infoboxCol1 := colComp.SetSize(size).SetContent(infobox1).GetContent()
	infoboxCol2 := colComp.SetSize(size).SetContent(infobox2).GetContent()
	infoboxCol3 := colComp.SetSize(size).SetContent(infobox3).GetContent()
	infoboxCol4 := colComp.SetSize(size).SetContent(infobox4).GetContent()
	row1 := components.Row().SetContent(infoboxCol1 + infoboxCol2 + infoboxCol3 + infoboxCol4).GetContent()

	lab1 := tmpl.Default().Label().SetContent("操作：").SetType("warning").GetContent()
	link1 := tmpl.Default().Link().
		SetURL("/admin/info/qq_config"). // 设置跳转路由
		SetContent("修改配置信息"). // 设置链接内容
		SetTabTitle("修改配置信息").
		SetClass("btn btn-sm btn-danger").
		GetContent()

	link2 := tmpl.Default().Link().
		SetURL("/admin/qq/checkconfig").
		SetContent("开始登录").
		SetTabTitle("开始登录").
		SetClass("btn btn-sm btn-primary").
		GetContent()
	rown := components.Row().SetContent(tmpl.Default().Box().WithHeadBorder().SetHeader(lab1).SetBody(link1 + link2).GetContent()).GetContent()
	return types.Panel{
		Content:     row1 + rown,
		Title:       "qq状态",
		Description: "当前qq状态信息",
	}, nil
}
