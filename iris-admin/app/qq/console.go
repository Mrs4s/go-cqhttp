package qq

import (
	"fmt"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/jump"
	"github.com/kataras/iris/v12"
)

// 删除好友
func (l *Dologin) DeleteFriend(ctx iris.Context) {
	err := l.CheckQQlogin(ctx)
	if err != nil {
		return
	}
	re := ctx.GetReferrer().URL
	if re == "" {
		re = "/admin/qq/friendlist"
	} else if ctx.GetReferrer().Path == "/admin/qq/deletefriend" {
		re = "/admin/qq/info"
	}
	uin, err := ctx.URLParamInt64("uin")
	if err != nil {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: "参数错误",
			Url: re,
		})
		return
	}
	err = l.Cli.DeleteFriend(uin)
	if err != nil {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("删除好友错误：%s", err.Error()),
			Url: re,
		})
		return
	}
	jump.JumpSuccessForIris(ctx, common.Msg{
		Msg: fmt.Sprintf("删除%d成功", uin),
		Url: re,
	})
}

// 退出群
func (l *Dologin) LeaveGroup(ctx iris.Context) {
	err := l.CheckQQlogin(ctx)
	if err != nil {
		return
	}
	re := ctx.GetReferrer().URL
	if re == "" {
		re = "/admin/qq/grouplist"
	} else if ctx.GetReferrer().Path == "/admin/qq/leavegroup" {
		re = "/admin/qq/info"
	}
	guin, err := ctx.URLParamInt64("guin")
	if g := l.Cli.FindGroup(guin); g != nil {
		g.Quit()
		jump.JumpSuccessForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("退出群%d成功", g.Uin),
			Url: re,
		})
		return
	}
	jump.JumpErrorForIris(ctx, common.Msg{
		Msg: fmt.Sprintf("群%d信息不存在", guin),
		Url: re,
	})
}
