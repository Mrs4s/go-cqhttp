package qq

import (
	"fmt"
	"strconv"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/utils/jump"
	"github.com/Mrs4s/go-cqhttp/coolq"
)

// DeleteFriend 删除好友
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
		jump.ErrorForIris(ctx, common.Msg{
			Msg: "参数错误",
			URL: re,
		})
		return
	}
	err = l.Cli.DeleteFriend(uin)
	if err != nil {
		jump.ErrorForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("删除好友错误：%s", err.Error()),
			URL: re,
		})
		return
	}
	jump.SuccessForIris(ctx, common.Msg{
		Msg: fmt.Sprintf("删除%d成功", uin),
		URL: re,
	})
}

// LeaveGroup 退出群
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
	guin, _ := ctx.URLParamInt64("guin")
	if g := l.Cli.FindGroup(guin); g != nil {
		g.Quit()
		jump.SuccessForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("退出群%d成功", g.Uin),
			URL: re,
		})
		return
	}
	jump.ErrorForIris(ctx, common.Msg{
		Msg: fmt.Sprintf("群%d信息不存在", guin),
		URL: re,
	})
}

// SendMsg 发送消息
func (l *Dologin) SendMsg(ctx iris.Context) {
	type data struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		MsgID string `json:"msg_id"`
	}
	uin, _ := ctx.PostValueInt64("uin")
	guid, _ := strconv.ParseUint(ctx.PostValue("guid"), 10, 64)
	cid, _ := strconv.ParseUint(ctx.PostValue("cid"), 10, 64)
	text := ctx.PostValue("text")
	if (uin == 0 && (guid == 0 || cid == 0)) || text == "" {
		_, _ = ctx.JSON(data{
			Code: -1,
			Msg:  "invalid params",
		})
		return
	}
	switch ctx.PostValue("type") {
	case "private":
		mid, err := l.sendPrivateMsg(uin, text)
		if err != nil {
			_, _ = ctx.JSON(data{
				Code: -1,
				Msg:  err.Error(),
			})
			return
		}
		_, _ = ctx.JSON(data{
			Code:  200,
			MsgID: strconv.FormatInt(int64(mid), 10),
		})
		return
	case "group":
		mid, err := l.sendGroupMsg(uin, text)
		if err != nil {
			_, _ = ctx.JSON(data{
				Code: -1,
				Msg:  err.Error(),
			})
			return
		}
		_, _ = ctx.JSON(data{
			Code:  200,
			MsgID: strconv.FormatInt(int64(mid), 10),
		})
		return
	case "channel":
		mid, err := l.sendGuildChannelMsg(guid, cid, text)
		if err != nil {
			_, _ = ctx.JSON(data{
				Code: -1,
				Msg:  err.Error(),
			})
			return
		}
		_, _ = ctx.JSON(data{
			Code:  200,
			MsgID: mid,
		})
		return
	default:
		_, _ = ctx.JSON(data{
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

func (l *Dologin) sendGuildChannelMsg(guildID, channelID uint64, msg string) (string, error) {
	guild := l.Cli.GuildService.FindGuild(guildID)
	if guild == nil {
		log.Errorf("guildid:%d not found", guildID)
		return "", errors.New("GUILD_NOT_FOUND")
	}
	channel := guild.FindChannel(channelID)
	if channel == nil {
		return "", errors.New("CHANNEL_NOT_FOUND")
	}
	if channel.ChannelType != client.ChannelTypeText {
		log.Warnf("无法发送频道信息: 频道类型错误, 不接受文本信息")
		return "", errors.New("CHANNEL_NOT_SUPPORTED_TEXT_MSG")
	}
	fixAt := func(elem []message.IMessageElement) {
		for _, e := range elem {
			if at, ok := e.(*message.AtElement); ok && at.Target != 0 && at.Display == "" {
				mem, _ := l.Cli.GuildService.FetchGuildMemberProfileInfo(guildID, uint64(at.Target))
				if mem != nil {
					at.Display = "@" + mem.Nickname
				} else {
					at.Display = "@" + strconv.FormatInt(at.Target, 10)
				}
			}
		}
	}
	var elem []message.IMessageElement
	if msg == "" {
		log.Warn("频道发送失败: 信息为空.")
		return "", errors.New("EMPTY_MSG_ERROR")
	}
	elem = l.Bot.ConvertStringMessage(msg, coolq.MessageSourceGuildChannel)
	fixAt(elem)
	mid := l.Bot.SendGuildChannelMessage(guildID, channelID, &message.SendingMessage{Elements: elem})
	if mid == "" {
		return "", errors.New("SEND_MSG_API_ERROR")
	}
	log.Infof("发送频道 %v(%v) 子频道 %v(%v) 的消息: %v (%v)", guild.GuildName, guild.GuildId, channel.ChannelName, channel.ChannelId, common.LimitedString(msg), mid)
	return mid, nil
}

// Shutdown 退出程序,用于docker重启
func (l *Dologin) Shutdown(ctx iris.Context) {
	err := l.checkAuth(ctx)
	if err != nil {
		return
	}
	l.Conn <- "shutdown"
	jump.SuccessForIris(ctx, common.Msg{
		Msg:  "服务即将关闭",
		URL:  "/admin/qq/info",
		Wait: 1,
	})
}
