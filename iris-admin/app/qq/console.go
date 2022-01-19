package qq

import (
	"fmt"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/jump"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strconv"
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

func (l *Dologin) SendMsg(ctx iris.Context) {
	type data struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		MsgId int32  `json:"msg_id"`
	}
	uin, err := ctx.PostValueInt64("uin")
	text := ctx.PostValue("text")
	if err != nil || uin == 0 || text == "" {
		ctx.JSON(data{
			Code: -1,
			Msg:  "invalid params",
		})
		return
	}
	switch ctx.PostValue("type") {
	case "private":
		mid, err := l.sendPrivateMsg(uin, text)
		if err != nil {
			ctx.JSON(data{
				Code: -1,
				Msg:  err.Error(),
			})
			return
		}
		ctx.JSON(data{
			Code:  200,
			MsgId: mid,
		})
		return
	case "group":
		mid, err := l.sendGroupMsg(uin, text)
		if err != nil {
			ctx.JSON(data{
				Code: -1,
				Msg:  err.Error(),
			})
			return
		}
		ctx.JSON(data{
			Code:  200,
			MsgId: mid,
		})
		return
	default:
		ctx.JSON(data{
			Code: -1,
			Msg:  "invalid params",
		})
		return
	}
}

func (l *Dologin) sendGroupMsg(groupID int64, msg string) (int32, error) {
	group := l.Cli.FindGroup(groupID)
	if group == nil {
		return 0, errors.New("GROUP_NOT_FOUND")
	}
	fixAt := func(elem []message.IMessageElement) {
		for _, e := range elem {
			if at, ok := e.(*message.AtElement); ok && at.Target != 0 && at.Display == "" {
				mem := group.FindMember(at.Target)
				if mem != nil {
					at.Display = "@" + mem.DisplayName()
				} else {
					at.Display = "@" + strconv.FormatInt(at.Target, 10)
				}
			}
		}
	}
	var elem []message.IMessageElement
	if msg == "" {
		log.Warn("群消息发送失败: 信息为空.")
		return 0, errors.New("EMPTY_MSG_ERROR")
	}
	elem = l.Bot.ConvertStringMessage(msg, coolq.MessageSourceGroup)

	fixAt(elem)
	mid := l.Bot.SendGroupMessage(groupID, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return mid, errors.New("SEND_MSG_API_ERROR")
	}
	log.Infof("发送群 %v(%v) 的消息: %v (%v)", group.Name, groupID, common.LimitedString(msg), mid)
	return mid, nil
}
func (l *Dologin) sendPrivateMsg(uin int64, msg string) (int32, error) {
	elem := l.Bot.ConvertStringMessage(msg, coolq.MessageSourcePrivate)
	mid := l.Bot.SendPrivateMessage(uin, 0, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return mid, errors.New("SEND_MSG_API_ERROR")
	}
	log.Infof("发送好友 %v(%v)  的消息: %v (%v)", uin, uin, common.LimitedString(msg), mid)
	return mid, nil
}
