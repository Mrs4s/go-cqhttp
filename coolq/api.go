package coolq

import (
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/global"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
)

// https://cqhttp.cc/docs/4.15/#/API?id=get_login_info-%E8%8E%B7%E5%8F%96%E7%99%BB%E5%BD%95%E5%8F%B7%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetLoginInfo() MSG {
	return OK(MSG{"user_id": bot.Client.Uin, "nickname": bot.Client.Nickname})
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_friend_list-%E8%8E%B7%E5%8F%96%E5%A5%BD%E5%8F%8B%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetFriendList() MSG {
	var fs []MSG
	for _, f := range bot.Client.FriendList {
		fs = append(fs, MSG{
			"nickname": f.Nickname,
			"remark":   f.Remark,
			"user_id":  f.Uin,
		})
	}
	return OK(fs)
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_group_list-%E8%8E%B7%E5%8F%96%E7%BE%A4%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetGroupList() MSG {
	var gs []MSG
	for _, g := range bot.Client.GroupList {
		gs = append(gs, MSG{
			"group_id":         g.Code,
			"group_name":       g.Name,
			"max_member_count": g.MaxMemberCount,
			"member_count":     g.MemberCount,
		})
	}
	return OK(gs)
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_group_info-%E8%8E%B7%E5%8F%96%E7%BE%A4%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetGroupInfo(groupId int64) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		return Failed(100)
	}
	return OK(MSG{
		"group_id":         group.Code,
		"group_name":       group.Name,
		"max_member_count": group.MaxMemberCount,
		"member_count":     group.MemberCount,
	})
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_group_member_list-%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%88%90%E5%91%98%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetGroupMemberList(groupId int64) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		return Failed(100)
	}
	var members []MSG
	for _, m := range group.Members {
		members = append(members, convertGroupMemberInfo(groupId, m))
	}
	return OK(members)
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_group_member_info-%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%88%90%E5%91%98%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetGroupMemberInfo(groupId, userId int64, noCache bool) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		return Failed(100)
	}
	if noCache {
		t, err := bot.Client.GetGroupMembers(group)
		if err != nil {
			log.Warnf("刷新群 %v 成员列表失败: %v", groupId, err)
			return Failed(100)
		}
		group.Members = t
	}
	member := group.FindMember(userId)
	if member == nil {
		return Failed(102)
	}
	return OK(convertGroupMemberInfo(groupId, member))
}

// https://cqhttp.cc/docs/4.15/#/API?id=send_group_msg-%E5%8F%91%E9%80%81%E7%BE%A4%E6%B6%88%E6%81%AF
func (bot *CQBot) CQSendGroupMessage(groupId int64, msg string) MSG {
	if msg == "" {
		return Failed(100)
	}
	elem := bot.ConvertStringMessage(msg, true)
	mid := bot.SendGroupMessage(groupId, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return Failed(100)
	}
	return OK(MSG{"message_id": mid})
}

// https://cqhttp.cc/docs/4.15/#/API?id=send_private_msg-%E5%8F%91%E9%80%81%E7%A7%81%E8%81%8A%E6%B6%88%E6%81%AF
func (bot *CQBot) CQSendPrivateMessage(userId int64, msg string) MSG {
	if msg == "" {
		return Failed(100)
	}
	elem := bot.ConvertStringMessage(msg, false)
	mid := bot.SendPrivateMessage(userId, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return Failed(100)
	}
	return OK(MSG{"message_id": mid})
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_card-%E8%AE%BE%E7%BD%AE%E7%BE%A4%E5%90%8D%E7%89%87%EF%BC%88%E7%BE%A4%E5%A4%87%E6%B3%A8%EF%BC%89
func (bot *CQBot) CQSetGroupCard(groupId, userId int64, card string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		if m := g.FindMember(userId); m != nil {
			m.EditCard(card)
			return OK(nil)
		}
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_special_title-%E8%AE%BE%E7%BD%AE%E7%BE%A4%E7%BB%84%E4%B8%93%E5%B1%9E%E5%A4%B4%E8%A1%94
func (bot *CQBot) CQSetGroupSpecialTitle(groupId, userId int64, title string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		if m := g.FindMember(userId); m != nil {
			m.EditSpecialTitle(title)
			return OK(nil)
		}
	}
	return Failed(100)
}

func (bot *CQBot) CQSetGroupName(groupId int64, name string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.UpdateName(name)
		return OK(nil)
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_kick-%E7%BE%A4%E7%BB%84%E8%B8%A2%E4%BA%BA
func (bot *CQBot) CQSetGroupKick(groupId, userId int64, msg string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		if m := g.FindMember(userId); m != nil {
			m.Kick(msg)
			return OK(nil)
		}
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_ban-%E7%BE%A4%E7%BB%84%E5%8D%95%E4%BA%BA%E7%A6%81%E8%A8%80
func (bot *CQBot) CQSetGroupBan(groupId, userId int64, duration uint32) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		if m := g.FindMember(userId); m != nil {
			m.Mute(duration)
			return OK(nil)
		}
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_whole_ban-%E7%BE%A4%E7%BB%84%E5%85%A8%E5%91%98%E7%A6%81%E8%A8%80
func (bot *CQBot) CQSetGroupWholeBan(groupId int64, enable bool) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.MuteAll(enable)
		return OK(nil)
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_friend_add_request-%E5%A4%84%E7%90%86%E5%8A%A0%E5%A5%BD%E5%8F%8B%E8%AF%B7%E6%B1%82
func (bot *CQBot) CQProcessFriendRequest(flag string, approve bool) MSG {
	req, ok := bot.friendReqCache.Load(flag)
	if !ok {
		return Failed(100)
	}
	if approve {
		req.(*client.NewFriendRequest).Accept()
	} else {
		req.(*client.NewFriendRequest).Reject()
	}
	return OK(nil)
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_add_request-%E5%A4%84%E7%90%86%E5%8A%A0%E7%BE%A4%E8%AF%B7%E6%B1%82%EF%BC%8F%E9%82%80%E8%AF%B7
func (bot *CQBot) CQProcessGroupRequest(flag, subType string, approve bool) MSG {
	if subType == "add" {
		req, ok := bot.joinReqCache.Load(flag)
		if !ok {
			return Failed(100)
		}
		bot.joinReqCache.Delete(flag)
		if approve {
			req.(*client.UserJoinGroupRequest).Accept()
		} else {
			req.(*client.UserJoinGroupRequest).Reject()
		}
		return OK(nil)
	}
	req, ok := bot.invitedReqCache.Load(flag)
	if ok {
		bot.invitedReqCache.Delete(flag)
		if approve {
			req.(*client.GroupInvitedRequest).Accept()
		} else {
			req.(*client.GroupInvitedRequest).Reject()
		}
		return OK(nil)
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=delete_msg-%E6%92%A4%E5%9B%9E%E6%B6%88%E6%81%AF
func (bot *CQBot) CQDeleteMessage(messageId int32) MSG {
	msg := bot.GetGroupMessage(messageId)
	if msg == nil {
		return Failed(100)
	}
	bot.Client.RecallGroupMessage(msg["group"].(int64), msg["message-id"].(int32), msg["internal-id"].(int32))
	return OK(nil)
}

func (bot *CQBot) CQGetImage(file string) MSG {
	if !global.PathExists(path.Join(global.IMAGE_PATH, file)) {
		return Failed(100)
	}
	if b, err := ioutil.ReadFile(path.Join(global.IMAGE_PATH, file)); err == nil {
		r := binary.NewReader(b)
		r.ReadBytes(16)
		return OK(MSG{
			"size":     r.ReadInt32(),
			"filename": r.ReadString(),
			"url":      r.ReadString(),
		})
	}
	return Failed(100)
}

func (bot *CQBot) CQGetGroupMessage(messageId int32) MSG {
	msg := bot.GetGroupMessage(messageId)
	if msg == nil {
		return Failed(100)
	}
	sender := msg["sender"].(message.Sender)
	return OK(MSG{
		"message_id": messageId,
		"real_id":    msg["message-id"],
		"sender": MSG{
			"user_id":  sender.Uin,
			"nickname": sender.Nickname,
		},
		"time":    msg["time"],
		"content": msg["message"],
	})
}

func (bot *CQBot) CQCanSendImage() MSG {
	return OK(MSG{"yes": true})
}

func (bot *CQBot) CQCanSendRecord() MSG {
	return OK(MSG{"yes": false})
}

func (bot *CQBot) CQGetStatus() MSG {
	return OK(MSG{
		"app_initialized": true,
		"app_enabled":     true,
		"plugins_good":    nil,
		"app_good":        true,
		"online":          bot.Client.Online,
		"good":            true,
	})
}

func (bot *CQBot) CQGetVersionInfo() MSG {
	wd, _ := os.Getwd()
	return OK(MSG{
		"coolq_directory":            wd,
		"coolq_edition":              "pro",
		"go-cqhttp":                  true,
		"plugin_version":             "4.15.0",
		"plugin_build_number":        99,
		"plugin_build_configuration": "release",
		"runtime_version":            runtime.Version(),
		"runtime_os":                 runtime.GOOS,
	})
}

func OK(data interface{}) MSG {
	return MSG{"data": data, "retcode": 0, "status": "ok"}
}

func Failed(code int) MSG {
	return MSG{"data": nil, "retcode": code, "status": "failed"}
}

func convertGroupMemberInfo(groupId int64, m *client.GroupMemberInfo) MSG {
	return MSG{
		"group_id":       groupId,
		"user_id":        m.Uin,
		"nickname":       m.Nickname,
		"card":           m.CardName,
		"sex":            "unknown",
		"age":            0,
		"area":           "",
		"join_time":      m.JoinTime,
		"last_sent_time": m.LastSpeakTime,
		"level":          strconv.FormatInt(int64(m.Level), 10),
		"role": func() string {
			switch m.Permission {
			case client.Owner:
				return "owner"
			case client.Administrator:
				return "admin"
			default:
				return "member"
			}
		}(),
		"unfriendly":        false,
		"title":             m.SpecialTitle,
		"title_expire_time": m.SpecialTitleExpireTime,
		"card_changeable":   false,
	}
}
