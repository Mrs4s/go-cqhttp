package qq

import (
	"fmt"
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/paginator"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/parameter"
	tmpl "github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/icon"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/themes/adminlte/components/infobox"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/jump"
	"github.com/kataras/iris/v12"
	"strconv"
	"time"
)

// 查看当前的qq状态
func (l *Dologin) QqInfo(ctx iris.Context) (types.Panel, error) {
	components := tmpl.Get(config.GetTheme())
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

	lab1 := components.Label().SetContent("快捷操作：").SetType("warning").GetContent()
	rowlab := components.Row().SetContent(tmpl.Default().Box().WithHeadBorder().SetBody(lab1).GetContent()).GetContent()
	link1 := components.Link().
		SetURL("/admin/info/qq_config"). // 设置跳转路由
		SetContent("修改配置信息"). // 设置链接内容
		SetTabTitle("修改配置信息").
		SetClass("btn btn-sm btn-danger").
		GetContent()

	link2 := components.Link().
		SetURL("/admin/qq/checkconfig").
		SetContent("开始登录").
		SetTabTitle("开始登录").
		SetClass("btn btn-sm btn-primary").
		GetContent()

	link3 := components.Link().
		SetURL("/admin/qq/shutdown").
		SetContent("关闭服务").
		SetTabTitle("关闭服务").
		SetClass("btn btn-sm btn-primary").
		GetContent()
	linkinfo := components.Link().
		SetURL("/admin/application/info").
		SetContent("系统信息").
		SetTabTitle("系统信息").
		SetClass("btn btn-sm btn-default").
		GetContent()
	linkLog := components.Link().
		SetURL("/admin/qq/weblog").
		OpenInNewTab().
		SetContent("系统日志").
		SetTabTitle("系统日志").
		SetClass("btn btn-sm btn-default").
		GetContent()
	rown1 := components.Row().SetContent(tmpl.Default().Box().WithHeadBorder().
		SetBody(
			link1 + link2 + link3 + linkinfo + linkLog,
		).GetContent()).GetContent()
	link4 := components.Link().
		SetURL("/admin/qq/friendlist").
		SetContent("好友列表").
		SetClass("btn btn-sm btn-default").
		GetContent()
	link5 := components.Link().
		SetURL("/admin/qq/grouplist").
		SetContent("群组列表").
		SetClass("btn btn-sm btn-default").
		GetContent()
	rown2 := components.Row().SetContent(tmpl.Default().Box().WithHeadBorder().SetBody(link4 + link5).GetContent()).GetContent()

	return types.Panel{
		Content:     row1 + rowlab + rown1 + rown2,
		Title:       "qq状态",
		Description: "当前qq状态信息",
	}, nil
}

// 好友列表
func (l *Dologin) MemberList(ctx iris.Context) (types.Panel, error) {
	err := l.CheckQQlogin(ctx)
	if err != nil {
		return types.Panel{}, nil
	}
	comp := tmpl.Get(config.GetTheme())
	fs := make([]map[string]types.InfoItem, 0, len(l.Cli.FriendList))
	for _, f := range l.Cli.FriendList {
		linkDelete := comp.Link().SetClass("btn btn-sm btn-danger").SetContent("删除").SetURL("/admin/qq/deletefriend?uin=" + strconv.FormatInt(f.Uin, 10)).GetContent()
		linkDetail := comp.Link().SetClass("btn btn-sm btn-primary").SetContent("详情").SetURL("/admin/qq/getfrienddetail?uin=" + strconv.FormatInt(f.Uin, 10)).GetContent()
		fs = append(fs, map[string]types.InfoItem{
			"nickname": {Content: tmpl.HTML(f.Nickname), Value: f.Nickname},
			"remark":   {Content: tmpl.HTML(f.Remark), Value: f.Remark},
			"user_id":  {Content: tmpl.HTML(strconv.FormatInt(f.Uin, 10)), Value: strconv.FormatInt(f.Uin, 10)},
			"console":  {Content: linkDetail + linkDelete},
		})
	}

	param := parameter.GetParam(ctx.Request().URL, 300)
	table := comp.DataTable().
		SetInfoList(common.PageSlice(fs, param.PageInt, param.PageSizeInt)).
		SetThead(types.Thead{
			{Head: "昵称", Field: "nickname"},
			{Head: "备注", Field: "remark"},
			{Head: "QQ号", Field: "user_id"},
			{Head: "操作", Field: "console"},
		})

	body := table.GetContent()

	return types.Panel{
		Content: comp.Box().
			SetBody(body).
			SetNoPadding().
			WithHeadBorder().
			SetFooter(paginator.Get(paginator.Config{
				Size:         len(l.Cli.FriendList),
				PageSizeList: []string{"100", "200", "300", "500"},
				Param:        param,
			}).GetContent()).
			GetContent(),
		Title:       "好友列表",
		Description: tmpl.HTML(fmt.Sprintf("%d的好友列表", l.Cli.Uin)),
	}, nil
}

// 群组列表
func (l *Dologin) GroupList(ctx iris.Context) (types.Panel, error) {
	err := l.CheckQQlogin(ctx)
	if err != nil {
		return types.Panel{}, nil
	}
	fs := make([]map[string]types.InfoItem, 0, len(l.Cli.GroupList))
	comp := tmpl.Get(config.GetTheme())

	for _, g := range l.Cli.GroupList {
		linkLeave := comp.Link().SetClass("btn btn-sm btn-danger").SetContent("退出群").SetURL("/admin/qq/leavegroup?guin=" + strconv.FormatInt(g.Code, 10)).GetContent()
		linkDetail := comp.Link().SetClass("btn btn-sm btn-primary").SetContent("详情").SetURL("/admin/qq/getgroupdetail?guin=" + strconv.FormatInt(g.Code, 10)).GetContent()
		fs = append(fs, map[string]types.InfoItem{
			"name":      {Content: tmpl.HTML(g.Name)},
			"owneruin":  {Content: tmpl.HTML(strconv.FormatInt(g.OwnerUin, 10))},
			"groupuin":  {Content: tmpl.HTML(strconv.FormatInt(g.Uin, 10))},
			"groupcode": {Content: tmpl.HTML(strconv.FormatInt(g.Code, 10))},
			"console":   {Content: linkDetail + linkLeave},
		})
	}
	param := parameter.GetParam(ctx.Request().URL, 100)

	table := comp.DataTable().
		SetInfoList(common.PageSlice(fs, param.PageInt, param.PageSizeInt)).
		SetThead(types.Thead{
			{Head: "群组名", Field: "name"},
			{Head: "群主uin", Field: "owneruin"},
			{Head: "群组uin", Field: "groupuin"},
			{Head: "群组code", Field: "groupcode"},
			{Head: "操作", Field: "console"},
		})

	body := table.GetContent()

	return types.Panel{
		Content: comp.Box().
			SetBody(body).
			SetNoPadding().
			WithHeadBorder().
			SetFooter(paginator.Get(paginator.Config{
				Size:         len(l.Cli.GroupList),
				PageSizeList: []string{"100", "200", "300", "500"},
				Param:        param,
			}).GetContent()).
			GetContent(),
		Title:       "群组列表",
		Description: tmpl.HTML(fmt.Sprintf("%d的群组列表", l.Cli.Uin)),
	}, nil
}

// 获取好友详细资料
func (l *Dologin) GetFriendDetal(ctx iris.Context) (types.Panel, error) {
	err := l.CheckQQlogin(ctx)
	if err != nil {
		return types.Panel{}, nil
	}
	re := ctx.GetReferrer().URL
	if re == "" {
		re = "/admin/qq/friendlist"
	} else if ctx.GetReferrer().Path == "/admin/qq/getfrienddetail" {
		re = "/admin/qq/info"
	}
	uin, err := ctx.URLParamInt64("uin")
	if err != nil {
		return jump.JumpError(common.Msg{
			Msg: "参数错误",
			Url: re,
		}), nil
	}
	info, err := l.Cli.GetSummaryInfo(uin)
	if err != nil {
		return jump.JumpError(common.Msg{
			Msg: fmt.Sprintf("获取uin%d的信息失败：%s", uin, err.Error()),
		}), nil
	}
	comp := tmpl.Get(config.GetTheme())

	table := comp.Table().
		SetHideThead().
		SetStyle("striped").
		SetMinWidth("30").
		SetInfoList([]map[string]types.InfoItem{
			{
				"key": {Content: "uin"},
				"val": {Content: tmpl.HTML(strconv.FormatInt(info.Uin, 10))},
			},
			{
				"key": {Content: "nickname"},
				"val": {Content: tmpl.HTML(info.Nickname)},
			},
			{
				"key": {Content: "city"},
				"val": {Content: tmpl.HTML(info.City)},
			},
			{
				"key": {Content: "mobile"},
				"val": {Content: tmpl.HTML(info.Mobile)},
			},
			{
				"key": {Content: "qid"},
				"val": {Content: tmpl.HTML(info.Qid)},
			},
			{
				"key": {Content: "sign"},
				"val": {Content: tmpl.HTML(info.Sign)},
			},
			{
				"key": {Content: "age"},
				"val": {Content: tmpl.HTML(fmt.Sprintf("%d", info.Age))},
			},
			{
				"key": {Content: "level"},
				"val": {Content: tmpl.HTML(fmt.Sprintf("%d", info.Level))},
			},
			{
				"key": {Content: "login_days"},
				"val": {Content: tmpl.HTML(fmt.Sprintf("%d", info.LoginDays))},
			},
			{
				"key": {Content: "sex"},
				"val": {Content: tmpl.HTML(func() string {
					if info.Sex == 1 {
						return "female"
					} else if info.Sex == 0 {
						return "male"
					}
					// unknown = 0x2
					return "unknown"
				}())},
			},
		}).
		SetThead(types.Thead{
			{Head: "key"},
			{Head: "val"},
		})

	body := table.GetContent()
	linkBack := comp.Link().SetURL(re).SetContent("返回").SetClass("btn btn-sm btn-info btn-flat pull-left").GetContent()
	return types.Panel{
		Content: comp.Box().
			SetBody(body).
			SetFooter(linkBack).
			SetNoPadding().
			WithHeadBorder().
			GetContent(),
		Title:       "好友详细信息",
		Description: tmpl.HTML(fmt.Sprintf("%d的详细资料", l.Cli.Uin)),
	}, nil
}

// 获取群信息
// 获取好友详细资料
func (l *Dologin) GetGroupDetal(ctx iris.Context) (types.Panel, error) {
	err := l.CheckQQlogin(ctx)
	if err != nil {
		return types.Panel{}, nil
	}
	re := ctx.GetReferrer().URL
	if re == "" {
		re = "/admin/qq/grouplist"
	} else if ctx.GetReferrer().Path == "/admin/qq/getgroupdetail" {
		re = "/admin/qq/info"
	}
	guin, err := ctx.URLParamInt64("guin")
	if err != nil {
		return jump.JumpError(common.Msg{
			Msg: "参数错误",
			Url: re,
		}), nil
	}
	group := l.Cli.FindGroup(guin)
	if group == nil {
		group, _ = l.Cli.GetGroupInfo(guin)
	}
	if group == nil {
		gid := strconv.FormatInt(guin, 10)
		info, err := l.Cli.SearchGroupByKeyword(gid)
		if err != nil {
			return jump.JumpError(common.Msg{
				Msg: fmt.Sprintf("获取uin%d的信息失败：%s", guin, err.Error()),
				Url: re,
			}), nil
		}
		for _, g := range info {
			if g.Code == guin {
				group = &client.GroupInfo{
					Code:            g.Code,
					Name:            g.Name,
					Memo:            g.Memo,
					Uin:             0,
					OwnerUin:        0,
					GroupCreateTime: 0,
					GroupLevel:      0,
					MaxMemberCount:  0,
					MemberCount:     0,
					Members:         nil,
				}
			}
		}
	}
	if group == nil {
		return jump.JumpError(common.Msg{
			Msg: fmt.Sprintf("获取uin%d的信息失败", guin),
			Url: re,
		}), nil
	}
	comp := tmpl.Get(config.GetTheme())

	table := comp.Table().
		SetHideThead().
		SetStyle("striped").
		SetMinWidth("30").
		SetInfoList([]map[string]types.InfoItem{
			{
				"key": {Content: "uin"},
				"val": {Content: tmpl.HTML(strconv.FormatInt(group.Uin, 10))},
			},
			{
				"key": {Content: "name"},
				"val": {Content: tmpl.HTML(group.Name)},
			},
			{
				"key": {Content: "memo"},
				"val": {Content: tmpl.HTML(group.Memo)},
			},
			{
				"key": {Content: "code"},
				"val": {Content: tmpl.HTML(strconv.FormatInt(group.Code, 10))},
			},
			{
				"key": {Content: "ownerUin"},
				"val": {Content: tmpl.HTML(strconv.FormatInt(group.OwnerUin, 10))},
			},
			{
				"key": {Content: "GroupCreateTime"},
				"val": {Content: tmpl.HTML(func() string {
					if group.GroupCreateTime == 0 {
						return ""
					}
					timeLayout := "2006-01-02 15:04:05"
					return time.Unix(int64(group.GroupCreateTime), 0).Format(timeLayout)
				}())},
			},
			{
				"key": {Content: "GroupLevel"},
				"val": {Content: tmpl.HTML(fmt.Sprintf("%d", group.GroupLevel))},
			},
			{
				"key": {Content: "MemberCount"},
				"val": {Content: tmpl.HTML(fmt.Sprintf("%d", group.MemberCount))},
			},
			{
				"key": {Content: "MaxMemberCount"},
				"val": {Content: tmpl.HTML(fmt.Sprintf("%d", group.MaxMemberCount))},
			},
		}).
		SetThead(types.Thead{
			{Head: "key"},
			{Head: "val"},
		})

	body := table.GetContent()
	linkBack := comp.Link().SetURL(re).SetContent("返回").SetClass("btn btn-sm btn-info btn-flat pull-left").GetContent()
	return types.Panel{
		Content: comp.Box().
			SetBody(body).
			SetFooter(linkBack).
			SetNoPadding().
			WithHeadBorder().
			GetContent(),
		Title:       "好友详细信息",
		Description: tmpl.HTML(fmt.Sprintf("%d的详细资料", l.Cli.Uin)),
	}, nil
}

// 最近的100条日志
func (l *Dologin) WebLog(ctx *context.Context) (types.Panel, error) {
	comp := tmpl.Get(config.GetTheme())
	re := ctx.Referer()
	if re == "" {
		re = "/admin/qq/info"
	} else if ctx.RefererURL().Path == "/admin/qq/weblog" {
		re = "/admin/qq/info"
	}
	refresh := ctx.QueryDefault("refresh", "true")
	body := comp.Box().SetBody(tmpl.HTML(l.Weblog.Read())).GetContent()
	linkBack := comp.Link().SetURL(re).SetContent("返回").SetClass("btn btn-sm btn-info btn-flat pull-left").GetContent()
	linkStop := comp.Link().SetURL("?refresh=false").SetContent("停止刷新").SetClass("btn btn-sm btn-info btn-flat pull-left").GetContent()
	linkSart := comp.Link().SetURL("?refresh=true").SetContent("启动刷新").SetClass("btn btn-sm btn-info btn-flat pull-left").GetContent()
	return types.Panel{
		Content: comp.Box().
			SetBody(body).
			SetFooter(linkBack + linkStop + linkSart).
			SetNoPadding().
			WithHeadBorder().
			GetContent(),
		Title:       "系统日志",
		Description: "go-cqhttp的最近100条日志。每2秒刷新",
		AutoRefresh: func() bool {
			return refresh == "true"
		}(),
		RefreshInterval: []int{2},
	}, nil
}
