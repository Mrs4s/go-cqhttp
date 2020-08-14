package coolq

import (
	"encoding/hex"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/global"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"
)

var format = "string"

func SetMessageFormat(f string) {
	format = f
}

func ToFormattedMessage(e []message.IMessageElement, code int64, raw ...bool) (r interface{}) {
	if format == "string" {
		r = ToStringMessage(e, code, raw...)
	} else if format == "array" {
		r = ToArrayMessage(e, code, raw...)
	}
	return
}

func (bot *CQBot) privateMessageEvent(c *client.QQClient, m *message.PrivateMessage) {
	bot.checkMedia(m.Elements)
	cqm := ToStringMessage(m.Elements, 0, true)
	log.Infof("收到好友 %v(%v) 的消息: %v", m.Sender.DisplayName(), m.Sender.Uin, cqm)
	fm := MSG{
		"post_type":    "message",
		"message_type": "private",
		"sub_type":     "friend",
		"message_id":   ToGlobalId(m.Sender.Uin, m.Id),
		"user_id":      m.Sender.Uin,
		"message":      ToFormattedMessage(m.Elements, 0, false),
		"raw_message":  cqm,
		"font":         0,
		"self_id":      c.Uin,
		"time":         time.Now().Unix(),
		"sender": MSG{
			"user_id":  m.Sender.Uin,
			"nickname": m.Sender.Nickname,
			"sex":      "unknown",
			"age":      0,
		},
	}
	bot.dispatchEventMessage(fm)
}

func (bot *CQBot) groupMessageEvent(c *client.QQClient, m *message.GroupMessage) {
	bot.checkMedia(m.Elements)
	for _, elem := range m.Elements {
		if file, ok := elem.(*message.GroupFileElement); ok {
			log.Infof("群 %v(%v) 内 %v(%v) 上传了文件: %v", m.GroupName, m.GroupCode, m.Sender.DisplayName(), m.Sender.Uin, file.Name)
			bot.dispatchEventMessage(MSG{
				"post_type":   "notice",
				"notice_type": "group_upload",
				"group_id":    m.GroupCode,
				"user_id":     m.Sender.Uin,
				"file": MSG{
					"id":    file.Path,
					"name":  file.Name,
					"size":  file.Size,
					"busid": file.Busid,
					"url":   c.GetGroupFileUrl(m.GroupCode, file.Path, file.Busid),
				},
				"self_id": c.Uin,
				"time":    time.Now().Unix(),
			})
			return
		}
	}
	cqm := ToStringMessage(m.Elements, m.GroupCode, true)
	id := m.Id
	if bot.db != nil {
		id = bot.InsertGroupMessage(m)
	}
	log.Infof("收到群 %v(%v) 内 %v(%v) 的消息: %v (%v)", m.GroupName, m.GroupCode, m.Sender.DisplayName(), m.Sender.Uin, cqm, id)
	gm := MSG{
		"anonymous":    nil,
		"font":         0,
		"group_id":     m.GroupCode,
		"message":      ToFormattedMessage(m.Elements, m.GroupCode, false),
		"message_id":   id,
		"message_type": "group",
		"post_type":    "message",
		"raw_message":  cqm,
		"self_id":      c.Uin,
		"sender": MSG{
			"age":     0,
			"area":    "",
			"level":   "",
			"sex":     "unknown",
			"user_id": m.Sender.Uin,
		},
		"sub_type": "normal",
		"time":     time.Now().Unix(),
		"user_id":  m.Sender.Uin,
	}
	if m.Sender.IsAnonymous() {
		gm["anonymous"] = MSG{
			"flag": "",
			"id":   0,
			"name": m.Sender.Nickname,
		}
		gm["sender"].(MSG)["nickname"] = "匿名消息"
		gm["sub_type"] = "anonymous"
	} else {
		mem := c.FindGroup(m.GroupCode).FindMember(m.Sender.Uin)
		ms := gm["sender"].(MSG)
		ms["role"] = func() string {
			switch mem.Permission {
			case client.Owner:
				return "owner"
			case client.Administrator:
				return "admin"
			default:
				return "member"
			}
		}()
		ms["nickname"] = mem.Nickname
		ms["card"] = mem.CardName
		ms["title"] = mem.SpecialTitle
	}
	bot.dispatchEventMessage(gm)
}

func (bot *CQBot) tempMessageEvent(c *client.QQClient, m *message.TempMessage) {
	bot.checkMedia(m.Elements)
	cqm := ToStringMessage(m.Elements, 0, true)
	bot.tempMsgCache.Store(m.Sender.Uin, m.GroupCode)
	log.Infof("收到来自群 %v(%v) 内 %v(%v) 的临时会话消息: %v", m.GroupName, m.GroupCode, m.Sender.DisplayName(), m.Sender.Uin, cqm)
	tm := MSG{
		"post_type":    "message",
		"message_type": "private",
		"sub_type":     "group",
		"message_id":   m.Id,
		"user_id":      m.Sender.Uin,
		"message":      ToFormattedMessage(m.Elements, 0, false),
		"raw_message":  cqm,
		"font":         0,
		"self_id":      c.Uin,
		"time":         time.Now().Unix(),
		"sender": MSG{
			"user_id":  m.Sender.Uin,
			"nickname": m.Sender.Nickname,
			"sex":      "unknown",
			"age":      0,
		},
	}
	bot.dispatchEventMessage(tm)
}

func (bot *CQBot) groupMutedEvent(c *client.QQClient, e *client.GroupMuteEvent) {
	g := c.FindGroup(e.GroupCode)
	if e.Time > 0 {
		log.Infof("群 %v 内 %v 被 %v 禁言了 %v秒.",
			formatGroupName(g), formatMemberName(g.FindMember(e.TargetUin)), formatMemberName(g.FindMember(e.OperatorUin)), e.Time)
	} else {
		log.Infof("群 %v 内 %v 被 %v 解除禁言.",
			formatGroupName(g), formatMemberName(g.FindMember(e.TargetUin)), formatMemberName(g.FindMember(e.OperatorUin)))
	}
	bot.dispatchEventMessage(MSG{
		"post_type":   "notice",
		"duration":    e.Time,
		"group_id":    e.GroupCode,
		"notice_type": "group_ban",
		"operator_id": e.OperatorUin,
		"self_id":     c.Uin,
		"user_id":     e.TargetUin,
		"time":        time.Now().Unix(),
		"sub_type": func() string {
			if e.Time > 0 {
				return "ban"
			}
			return "lift_ban"
		}(),
	})
}

func (bot *CQBot) groupRecallEvent(c *client.QQClient, e *client.GroupMessageRecalledEvent) {
	g := c.FindGroup(e.GroupCode)
	gid := ToGlobalId(e.GroupCode, e.MessageId)
	log.Infof("群 %v 内 %v 撤回了 %v 的消息: %v.",
		formatGroupName(g), formatMemberName(g.FindMember(e.OperatorUin)), formatMemberName(g.FindMember(e.AuthorUin)), gid)
	bot.dispatchEventMessage(MSG{
		"post_type":   "notice",
		"group_id":    e.GroupCode,
		"notice_type": "group_recall",
		"self_id":     c.Uin,
		"user_id":     e.AuthorUin,
		"operator_id": e.OperatorUin,
		"time":        e.Time,
		"message_id":  gid,
	})
}

func (bot *CQBot) friendRecallEvent(c *client.QQClient, e *client.FriendMessageRecalledEvent) {
	f := c.FindFriend(e.FriendUin)
	gid := ToGlobalId(e.FriendUin, e.MessageId)
	log.Infof("好友 %v(%v) 撤回了消息: %v", f.Nickname, f.Uin, gid)
	bot.dispatchEventMessage(MSG{
		"post_type":   "notice",
		"notice_type": "friend_recall",
		"self_id":     c.Uin,
		"user_id":     f.Uin,
		"time":        e.Time,
		"message_id":  gid,
	})
}

func (bot *CQBot) joinGroupEvent(c *client.QQClient, group *client.GroupInfo) {
	log.Infof("Bot进入了群 %v.", formatGroupName(group))
	bot.dispatchEventMessage(bot.groupIncrease(group.Code, 0, c.Uin))
}

func (bot *CQBot) leaveGroupEvent(c *client.QQClient, e *client.GroupLeaveEvent) {
	if e.Operator != nil {
		log.Infof("Bot被 %v T出了群 %v.", formatMemberName(e.Operator), formatGroupName(e.Group))
	} else {
		log.Infof("Bot退出了群 %v.", formatGroupName(e.Group))
	}
	bot.dispatchEventMessage(bot.groupDecrease(e.Group.Code, c.Uin, e.Operator))
}

func (bot *CQBot) memberPermissionChangedEvent(c *client.QQClient, e *client.MemberPermissionChangedEvent) {
	st := func() string {
		if e.NewPermission == client.Administrator {
			return "set"
		}
		return "unset"
	}()
	bot.dispatchEventMessage(MSG{
		"post_type":   "notice",
		"notice_type": "group_admin",
		"sub_type":    st,
		"group_id":    e.Group.Code,
		"user_id":     e.Member.Uin,
		"time":        time.Now().Unix(),
		"self_id":     c.Uin,
	})
}

func (bot *CQBot) memberJoinEvent(c *client.QQClient, e *client.MemberJoinGroupEvent) {
	log.Infof("新成员 %v 进入了群 %v.", formatMemberName(e.Member), formatGroupName(e.Group))
	bot.dispatchEventMessage(bot.groupIncrease(e.Group.Code, 0, e.Member.Uin))
}

func (bot *CQBot) memberLeaveEvent(c *client.QQClient, e *client.MemberLeaveGroupEvent) {
	if e.Operator != nil {
		log.Infof("成员 %v 被 %v T出了群 %v.", formatMemberName(e.Member), formatMemberName(e.Operator), formatGroupName(e.Group))
	} else {
		log.Infof("成员 %v 离开了群 %v.", formatMemberName(e.Member), formatGroupName(e.Group))
	}
	bot.dispatchEventMessage(bot.groupDecrease(e.Group.Code, e.Member.Uin, e.Operator))
}

func (bot *CQBot) friendRequestEvent(c *client.QQClient, e *client.NewFriendRequest) {
	log.Infof("收到来自 %v(%v) 的好友请求: %v", e.RequesterNick, e.RequesterUin, e.Message)
	flag := strconv.FormatInt(e.RequestId, 10)
	bot.friendReqCache.Store(flag, e)
	bot.dispatchEventMessage(MSG{
		"post_type":    "request",
		"request_type": "friend",
		"user_id":      e.RequesterUin,
		"comment":      e.Message,
		"flag":         flag,
		"time":         time.Now().Unix(),
		"self_id":      c.Uin,
	})
}

func (bot *CQBot) friendAddedEvent(c *client.QQClient, e *client.NewFriendEvent) {
	log.Infof("添加了新好友: %v(%v)", e.Friend.Nickname, e.Friend.Uin)
	bot.tempMsgCache.Delete(e.Friend.Uin)
	bot.dispatchEventMessage(MSG{
		"post_type":   "notice",
		"notice_type": "friend_add",
		"self_id":     c.Uin,
		"user_id":     e.Friend.Uin,
		"time":        time.Now().Unix(),
	})
}

func (bot *CQBot) groupInvitedEvent(c *client.QQClient, e *client.GroupInvitedRequest) {
	log.Infof("收到来自群 %v(%v) 内用户 %v(%v) 的加群邀请.", e.GroupName, e.GroupCode, e.InvitorNick, e.InvitorUin)
	flag := strconv.FormatInt(e.RequestId, 10)
	bot.invitedReqCache.Store(flag, e)
	bot.dispatchEventMessage(MSG{
		"post_type":    "request",
		"request_type": "group",
		"sub_type":     "invite",
		"group_id":     e.GroupCode,
		"user_id":      e.InvitorUin,
		"comment":      "",
		"flag":         flag,
		"time":         time.Now().Unix(),
		"self_id":      c.Uin,
	})
}

func (bot *CQBot) groupJoinReqEvent(c *client.QQClient, e *client.UserJoinGroupRequest) {
	log.Infof("群 %v(%v) 收到来自用户 %v(%v) 的加群请求.", e.GroupName, e.GroupCode, e.RequesterNick, e.RequesterUin)
	flag := strconv.FormatInt(e.RequestId, 10)
	bot.joinReqCache.Store(flag, e)
	bot.dispatchEventMessage(MSG{
		"post_type":    "request",
		"request_type": "group",
		"sub_type":     "add",
		"group_id":     e.GroupCode,
		"user_id":      e.RequesterUin,
		"comment":      e.Message,
		"flag":         flag,
		"time":         time.Now().Unix(),
		"self_id":      c.Uin,
	})
}

func (bot *CQBot) groupIncrease(groupCode, operatorUin, userUin int64) MSG {
	return MSG{
		"post_type":   "notice",
		"notice_type": "group_increase",
		"group_id":    groupCode,
		"operator_id": operatorUin,
		"self_id":     bot.Client.Uin,
		"sub_type":    "approve",
		"time":        time.Now().Unix(),
		"user_id":     userUin,
	}
}

func (bot *CQBot) groupDecrease(groupCode, userUin int64, operator *client.GroupMemberInfo) MSG {
	return MSG{
		"post_type":   "notice",
		"notice_type": "group_decrease",
		"group_id":    groupCode,
		"operator_id": func() int64 {
			if operator != nil {
				return operator.Uin
			}
			return userUin
		}(),
		"self_id": bot.Client.Uin,
		"sub_type": func() string {
			if operator != nil {
				if userUin == bot.Client.Uin {
					return "kick_me"
				}
				return "kick"
			}
			return "leave"
		}(),
		"time":    time.Now().Unix(),
		"user_id": userUin,
	}
}

func (bot *CQBot) checkMedia(e []message.IMessageElement) {
	for _, elem := range e {
		switch i := elem.(type) {
		case *message.ImageElement:
			filename := hex.EncodeToString(i.Md5) + ".image"
			if !global.PathExists(path.Join(global.IMAGE_PATH, filename)) {
				_ = ioutil.WriteFile(path.Join(global.IMAGE_PATH, filename), binary.NewWriterF(func(w *binary.Writer) {
					w.Write(i.Md5)
					w.WriteUInt32(uint32(i.Size))
					w.WriteString(i.Filename)
					w.WriteString(i.Url)
				}), 0644)
			}
			i.Filename = filename
		case *message.VoiceElement:
			i.Name = strings.ReplaceAll(i.Name, "{", "")
			i.Name = strings.ReplaceAll(i.Name, "}", "")
			if !global.PathExists(path.Join(global.VOICE_PATH, i.Name)) {
				b, err := global.GetBytes(i.Url)
				if err != nil {
					log.Warnf("语音文件 %v 下载失败: %v", i.Name, err)
					continue
				}
				_ = ioutil.WriteFile(path.Join(global.VOICE_PATH, i.Name), b, 0644)
			}
		case *message.ShortVideoElement:
			filename := hex.EncodeToString(i.Md5) + ".video"
			if !global.PathExists(path.Join(global.VIDEO_PATH, filename)) {
				_ = ioutil.WriteFile(path.Join(global.VIDEO_PATH, filename), binary.NewWriterF(func(w *binary.Writer) {
					w.Write(i.Md5)
					w.WriteUInt32(uint32(i.Size))
					w.WriteString(i.Name)
					w.Write(i.Uuid)
				}), 0644)
			}
			i.Name = filename
			i.Url = bot.Client.GetShortVideoUrl(i.Uuid, i.Md5)
		}
	}
}
