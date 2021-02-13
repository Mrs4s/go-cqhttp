package coolq

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mrs4s/go-cqhttp/global"
	log "github.com/sirupsen/logrus"
)

var Version = "unknown"

// https://cqhttp.cc/docs/4.15/#/API?id=get_login_info-%E8%8E%B7%E5%8F%96%E7%99%BB%E5%BD%95%E5%8F%B7%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetLoginInfo() MSG {
	return OK(MSG{"user_id": bot.Client.Uin, "nickname": bot.Client.Nickname})
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_friend_list-%E8%8E%B7%E5%8F%96%E5%A5%BD%E5%8F%8B%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetFriendList() MSG {
	fs := make([]MSG, 0)
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
func (bot *CQBot) CQGetGroupList(noCache bool) MSG {
	gs := make([]MSG, 0)
	if noCache {
		_ = bot.Client.ReloadGroupList()
	}
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
func (bot *CQBot) CQGetGroupInfo(groupId int64, noCache bool) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		gid := strconv.FormatInt(groupId, 10)
		info, err := bot.Client.SearchGroupByKeyword(gid)
		if err != nil {
			return Failed(100, "GROUP_SEARCH_ERROR", "群聊搜索失败")
		}
		for _, g := range info {
			if g.Code == groupId {
				return OK(MSG{
					"group_id":         g.Code,
					"group_name":       g.Name,
					"max_member_count": 0,
					"member_count":     0,
				})
			}
		}
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在失败")
	}
	if noCache {
		var err error
		group, err = bot.Client.GetGroupInfo(groupId)
		if err != nil {
			return Failed(100, "GET_GROUP_INFO_API_ERROR", err.Error())
		}
	}
	return OK(MSG{
		"group_id":         group.Code,
		"group_name":       group.Name,
		"max_member_count": group.MaxMemberCount,
		"member_count":     group.MemberCount,
	})
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_group_member_list-%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%88%90%E5%91%98%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetGroupMemberList(groupId int64, noCache bool) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	if noCache {
		t, err := bot.Client.GetGroupMembers(group)
		if err != nil {
			log.Warnf("刷新群 %v 成员列表失败: %v", groupId, err)
			return Failed(100, "GET_MEMBERS_API_ERROR", err.Error())
		}
		group.Members = t
	}
	members := make([]MSG, 0)
	for _, m := range group.Members {
		members = append(members, convertGroupMemberInfo(groupId, m))
	}
	return OK(members)
}

// https://cqhttp.cc/docs/4.15/#/API?id=get_group_member_info-%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%88%90%E5%91%98%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetGroupMemberInfo(groupId, userId int64) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	member := group.FindMember(userId)
	if member == nil {
		return Failed(100, "MEMBER_NOT_FOUND", "群员不存在")
	}
	return OK(convertGroupMemberInfo(groupId, member))
}

func (bot *CQBot) CQGetGroupFileSystemInfo(groupId int64) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupId)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupId, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(fs)
}

func (bot *CQBot) CQGetGroupRootFiles(groupId int64) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupId)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupId, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	files, folders, err := fs.Root()
	if err != nil {
		log.Errorf("获取群 %v 根目录文件失败: %v", groupId, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(MSG{
		"files":   files,
		"folders": folders,
	})
}

func (bot *CQBot) CQGetGroupFilesByFolderId(groupId int64, folderId string) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupId)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupId, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	files, folders, err := fs.GetFilesByFolder(folderId)
	if err != nil {
		log.Errorf("获取群 %v 根目录 %v 子文件失败: %v", groupId, folderId, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(MSG{
		"files":   files,
		"folders": folders,
	})
}

func (bot *CQBot) CQGetGroupFileUrl(groupId int64, fileId string, busId int32) MSG {
	url := bot.Client.GetGroupFileUrl(groupId, fileId, busId)
	if url == "" {
		return Failed(100, "FILE_SYSTEM_API_ERROR")
	}
	return OK(MSG{
		"url": url,
	})
}

func (bot *CQBot) CQUploadGroupFile(groupId int64, file, name, folder string) MSG {
	if !global.PathExists(file) {
		log.Errorf("上传群文件 %v 失败: 文件不存在", file)
		return Failed(100, "FILE_NOT_FOUND", "文件不存在")
	}
	fs, err := bot.Client.GetGroupFileSystem(groupId)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupId, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	if folder == "" {
		folder = "/"
	}
	if err = fs.UploadFile(file, name, folder); err != nil {
		log.Errorf("上传群 %v 文件 %v 失败: %v", groupId, file, err)
		return Failed(100, "FILE_SYSTEM_UPLOAD_API_ERROR", err.Error())
	}
	return OK(nil)
}

func (bot *CQBot) CQGetWordSlices(content string) MSG {
	slices, err := bot.Client.GetWordSegmentation(content)
	if err != nil {
		return Failed(100, "WORD_SEGMENTATION_API_ERROR", err.Error())
	}
	for i := 0; i < len(slices); i++ {
		slices[i] = strings.ReplaceAll(slices[i], "\u0000", "")
	}
	return OK(MSG{"slices": slices})
}

// https://cqhttp.cc/docs/4.15/#/API?id=send_group_msg-%E5%8F%91%E9%80%81%E7%BE%A4%E6%B6%88%E6%81%AF
func (bot *CQBot) CQSendGroupMessage(groupId int64, i interface{}, autoEscape bool) MSG {
	if bot.Client.FindGroup(groupId) == nil {
		log.Warnf("群消息发送失败: 群 %v 不存在", groupId)
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	var str string
	fixAt := func(elem []message.IMessageElement) {
		for _, e := range elem {
			if at, ok := e.(*message.AtElement); ok && at.Target != 0 {
				at.Display = "@" + func() string {
					mem := bot.Client.FindGroup(groupId).FindMember(at.Target)
					if mem != nil {
						return mem.DisplayName()
					}
					return strconv.FormatInt(at.Target, 10)
				}()
			}
		}
	}
	if m, ok := i.(gjson.Result); ok {
		if m.Type == gjson.JSON {
			elem := bot.ConvertObjectMessage(m, true)
			fixAt(elem)
			mid := bot.SendGroupMessage(groupId, &message.SendingMessage{Elements: elem})
			if mid == -1 {
				return Failed(100, "SEND_MSG_API_ERROR", "请参考输出")
			}
			log.Infof("发送群 %v(%v)  的消息: %v (%v)", groupId, groupId, limitedString(m.String()), mid)
			return OK(MSG{"message_id": mid})
		}
		str = func() string {
			if m.Str != "" {
				return m.Str
			}
			return m.Raw
		}()
	} else if s, ok := i.(string); ok {
		str = s
	}
	if str == "" {
		log.Warnf("群消息发送失败: 信息为空. MSG: %v", i)
		return Failed(100, "EMPTY_MSG_ERROR", "消息为空")
	}
	var elem []message.IMessageElement
	if autoEscape {
		elem = append(elem, message.NewText(str))
	} else {
		elem = bot.ConvertStringMessage(str, true)
	}
	fixAt(elem)
	mid := bot.SendGroupMessage(groupId, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return Failed(100, "SEND_MSG_API_ERROR", "请参考输出")
	}
	log.Infof("发送群 %v(%v)  的消息: %v (%v)", groupId, groupId, limitedString(str), mid)
	return OK(MSG{"message_id": mid})
}

func (bot *CQBot) CQSendGroupForwardMessage(groupId int64, m gjson.Result) MSG {
	if m.Type != gjson.JSON {
		return Failed(100)
	}
	var sendNodes []*message.ForwardNode
	ts := time.Now().Add(-time.Minute * 5)
	hasCustom := func() bool {
		for _, item := range m.Array() {
			if item.Get("data.uin").Exists() {
				return true
			}
		}
		return false
	}()
	var convert func(e gjson.Result) []*message.ForwardNode
	convert = func(e gjson.Result) (nodes []*message.ForwardNode) {
		if e.Get("type").Str != "node" {
			return nil
		}
		ts.Add(time.Second)
		if e.Get("data.id").Exists() {
			i, _ := strconv.Atoi(e.Get("data.id").String())
			m := bot.GetMessage(int32(i))
			if m != nil {
				sender := m["sender"].(message.Sender)
				nodes = append(nodes, &message.ForwardNode{
					SenderId:   sender.Uin,
					SenderName: (&sender).DisplayName(),
					Time: func() int32 {
						msgTime := m["time"].(int32)
						if hasCustom && msgTime == 0 {
							return int32(ts.Unix())
						}
						return msgTime
					}(),
					Message: bot.ConvertStringMessage(m["message"].(string), true),
				})
				return
			}
			log.Warnf("警告: 引用消息 %v 错误或数据库未开启.", e.Get("data.id").Str)
			return
		}
		uin, _ := strconv.ParseInt(e.Get("data.uin").Str, 10, 64)
		msgTime, err := strconv.ParseInt(e.Get("data.time").Str, 10, 64)
		if err != nil {
			msgTime = ts.Unix()
		}
		name := e.Get("data.name").Str
		c := e.Get("data.content")
		if c.IsArray() {
			flag := false
			c.ForEach(func(_, value gjson.Result) bool {
				if value.Get("type").String() == "node" {
					flag = true
					return false
				}
				return true
			})
			if flag {
				var taowa []*message.ForwardNode
				for _, item := range c.Array() {
					taowa = append(taowa, convert(item)...)
				}
				nodes = append(nodes, &message.ForwardNode{
					SenderId:   uin,
					SenderName: name,
					Time:       int32(msgTime),
					Message:    []message.IMessageElement{bot.Client.UploadGroupForwardMessage(groupId, &message.ForwardMessage{Nodes: taowa})},
				})
				return
			}
		}
		content := bot.ConvertObjectMessage(e.Get("data.content"), true)
		if uin != 0 && name != "" && len(content) > 0 {
			var newElem []message.IMessageElement
			for _, elem := range content {
				if img, ok := elem.(*LocalImageElement); ok {
					gm, err := bot.UploadLocalImageAsGroup(groupId, img)
					if err != nil {
						log.Warnf("警告：群 %v 图片上传失败: %v", groupId, err)
						continue
					}
					newElem = append(newElem, gm)
					continue
				}
				if video, ok := elem.(*LocalVideoElement); ok {
					gm, err := bot.UploadLocalVideo(groupId, video)
					if err != nil {
						log.Warnf("警告：群 %v 视频上传失败: %v", groupId, err)
						continue
					}
					newElem = append(newElem, gm)
					continue
				}
				newElem = append(newElem, elem)
			}
			nodes = append(nodes, &message.ForwardNode{
				SenderId:   uin,
				SenderName: name,
				Time:       int32(msgTime),
				Message:    newElem,
			})
			return
		}
		log.Warnf("警告: 非法 Forward node 将跳过")
		return
	}
	if m.IsArray() {
		for _, item := range m.Array() {
			sendNodes = append(sendNodes, convert(item)...)
		}
	} else {
		sendNodes = convert(m)
	}
	if len(sendNodes) > 0 {
		gm := bot.Client.SendGroupForwardMessage(groupId, &message.ForwardMessage{Nodes: sendNodes})
		return OK(MSG{
			"message_id": bot.InsertGroupMessage(gm),
		})
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=send_private_msg-%E5%8F%91%E9%80%81%E7%A7%81%E8%81%8A%E6%B6%88%E6%81%AF
func (bot *CQBot) CQSendPrivateMessage(userId int64, i interface{}, autoEscape bool) MSG {
	var str string
	if m, ok := i.(gjson.Result); ok {
		if m.Type == gjson.JSON {
			elem := bot.ConvertObjectMessage(m, false)
			mid := bot.SendPrivateMessage(userId, &message.SendingMessage{Elements: elem})
			if mid == -1 {
				return Failed(100, "SEND_MSG_API_ERROR", "请参考输出")
			}
			log.Infof("发送好友 %v(%v)  的消息: %v (%v)", userId, userId, limitedString(m.String()), mid)
			return OK(MSG{"message_id": mid})
		}
		str = func() string {
			if m.Str != "" {
				return m.Str
			}
			return m.Raw
		}()
	} else if s, ok := i.(string); ok {
		str = s
	}
	if str == "" {
		return Failed(100, "EMPTY_MSG_ERROR", "消息为空")
	}
	var elem []message.IMessageElement
	if autoEscape {
		elem = append(elem, message.NewText(str))
	} else {
		elem = bot.ConvertStringMessage(str, false)
	}
	mid := bot.SendPrivateMessage(userId, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return Failed(100, "SEND_MSG_API_ERROR", "请参考输出")
	}
	log.Infof("发送好友 %v(%v)  的消息: %v (%v)", userId, userId, limitedString(str), mid)
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
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_special_title-%E8%AE%BE%E7%BD%AE%E7%BE%A4%E7%BB%84%E4%B8%93%E5%B1%9E%E5%A4%B4%E8%A1%94
func (bot *CQBot) CQSetGroupSpecialTitle(groupId, userId int64, title string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		if m := g.FindMember(userId); m != nil {
			m.EditSpecialTitle(title)
			return OK(nil)
		}
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

func (bot *CQBot) CQSetGroupName(groupId int64, name string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.UpdateName(name)
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

func (bot *CQBot) CQSetGroupMemo(groupId int64, msg string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.UpdateMemo(msg)
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_kick-%E7%BE%A4%E7%BB%84%E8%B8%A2%E4%BA%BA
func (bot *CQBot) CQSetGroupKick(groupId, userId int64, msg string, block bool) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		if m := g.FindMember(userId); m != nil {
			m.Kick(msg, block)
			return OK(nil)
		}
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_ban-%E7%BE%A4%E7%BB%84%E5%8D%95%E4%BA%BA%E7%A6%81%E8%A8%80
func (bot *CQBot) CQSetGroupBan(groupId, userId int64, duration uint32) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		if m := g.FindMember(userId); m != nil {
			m.Mute(duration)
			return OK(nil)
		}
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_whole_ban-%E7%BE%A4%E7%BB%84%E5%85%A8%E5%91%98%E7%A6%81%E8%A8%80
func (bot *CQBot) CQSetGroupWholeBan(groupId int64, enable bool) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.MuteAll(enable)
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_leave-%E9%80%80%E5%87%BA%E7%BE%A4%E7%BB%84
func (bot *CQBot) CQSetGroupLeave(groupId int64) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.Quit()
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

func (bot *CQBot) CQGetAtAllRemain(groupId int64) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		i, err := bot.Client.GetAtAllRemain(groupId)
		if err != nil {
			return Failed(100, "GROUP_REMAIN_API_ERROR", err.Error())
		}
		return OK(i)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_friend_add_request-%E5%A4%84%E7%90%86%E5%8A%A0%E5%A5%BD%E5%8F%8B%E8%AF%B7%E6%B1%82
func (bot *CQBot) CQProcessFriendRequest(flag string, approve bool) MSG {
	req, ok := bot.friendReqCache.Load(flag)
	if !ok {
		return Failed(100, "FLAG_NOT_FOUND", "FLAG不存在")
	}
	if approve {
		req.(*client.NewFriendRequest).Accept()
	} else {
		req.(*client.NewFriendRequest).Reject()
	}
	return OK(nil)
}

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_add_request-%E5%A4%84%E7%90%86%E5%8A%A0%E7%BE%A4%E8%AF%B7%E6%B1%82%EF%BC%8F%E9%82%80%E8%AF%B7
func (bot *CQBot) CQProcessGroupRequest(flag, subType, reason string, approve bool) MSG {
	msgs, err := bot.Client.GetGroupSystemMessages()
	if err != nil {
		log.Errorf("获取群系统消息失败: %v", err)
		return Failed(100, "SYSTEM_MSG_API_ERROR", err.Error())
	}
	if subType == "add" {
		for _, req := range msgs.JoinRequests {
			if strconv.FormatInt(req.RequestId, 10) == flag {
				if req.Checked {
					log.Errorf("处理群系统消息失败: 无法操作已处理的消息.")
					return Failed(100, "FLAG_HAS_BEEN_CHECKED", "消息已被处理")
				}
				if approve {
					req.Accept()
				} else {
					req.Reject(false, reason)
				}
				return OK(nil)
			}
		}
	} else {
		for _, req := range msgs.InvitedRequests {
			if strconv.FormatInt(req.RequestId, 10) == flag {
				if req.Checked {
					log.Errorf("处理群系统消息失败: 无法操作已处理的消息.")
					return Failed(100, "FLAG_HAS_BEEN_CHECKED", "消息已被处理")
				}
				if approve {
					req.Accept()
				} else {
					req.Reject(false, reason)
				}
				return OK(nil)
			}
		}
	}
	log.Errorf("处理群系统消息失败: 消息 %v 不存在.", flag)
	return Failed(100, "FLAG_NOT_FOUND", "FLAG不存在")
}

// https://cqhttp.cc/docs/4.15/#/API?id=delete_msg-%E6%92%A4%E5%9B%9E%E6%B6%88%E6%81%AF
func (bot *CQBot) CQDeleteMessage(messageId int32) MSG {
	msg := bot.GetMessage(messageId)
	if msg == nil {
		return Failed(100, "MESSAGE_NOT_FOUND", "消息不存在")
	}
	if _, ok := msg["group"]; ok {
		if err := bot.Client.RecallGroupMessage(msg["group"].(int64), msg["message-id"].(int32), msg["internal-id"].(int32)); err != nil {
			log.Warnf("撤回 %v 失败: %v", messageId, err)
			return Failed(100, "RECALL_API_ERROR", err.Error())
		}
	} else {
		if msg["sender"].(message.Sender).Uin != bot.Client.Uin {
			log.Warnf("撤回 %v 失败: 好友会话无法撤回对方消息.", messageId)
			return Failed(100, "CANNOT_RECALL_FRIEND_MSG", "无法撤回对方消息")
		}
		if err := bot.Client.RecallPrivateMessage(msg["target"].(int64), int64(msg["time"].(int32)), msg["message-id"].(int32), msg["internal-id"].(int32)); err != nil {
			log.Warnf("撤回 %v 失败: %v", messageId, err)
			return Failed(100, "RECALL_API_ERROR", err.Error())
		}
	}
	return OK(nil)
}

// https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_admin-%E7%BE%A4%E7%BB%84%E8%AE%BE%E7%BD%AE%E7%AE%A1%E7%90%86%E5%91%98
func (bot *CQBot) CQSetGroupAdmin(groupId, userId int64, enable bool) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil || group.OwnerUin != bot.Client.Uin {
		return Failed(100, "PERMISSION_DENIED", "群不存在或权限不足")
	}
	mem := group.FindMember(userId)
	if mem == nil {
		return Failed(100, "GROUP_MEMBER_NOT_FOUND", "群成员不存在")
	}
	mem.SetAdmin(enable)
	t, err := bot.Client.GetGroupMembers(group)
	if err != nil {
		log.Warnf("刷新群 %v 成员列表失败: %v", groupId, err)
		return Failed(100, "GET_MEMBERS_API_ERROR", err.Error())
	}
	group.Members = t
	return OK(nil)
}

func (bot *CQBot) CQGetVipInfo(userId int64) MSG {
	vip, err := bot.Client.GetVipInfo(userId)
	if err != nil {
		return Failed(100, "VIP_API_ERROR", err.Error())
	}
	msg := MSG{
		"user_id":          vip.Uin,
		"nickname":         vip.Name,
		"level":            vip.Level,
		"level_speed":      vip.LevelSpeed,
		"vip_level":        vip.VipLevel,
		"vip_growth_speed": vip.VipGrowthSpeed,
		"vip_growth_total": vip.VipGrowthTotal,
	}
	return OK(msg)
}

// https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_group_honor_info-%E8%8E%B7%E5%8F%96%E7%BE%A4%E8%8D%A3%E8%AA%89%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetGroupHonorInfo(groupId int64, t string) MSG {
	msg := MSG{"group_id": groupId}
	convertMem := func(memList []client.HonorMemberInfo) (ret []MSG) {
		for _, mem := range memList {
			ret = append(ret, MSG{
				"user_id":     mem.Uin,
				"nickname":    mem.Name,
				"avatar":      mem.Avatar,
				"description": mem.Desc,
			})
		}
		return
	}
	if t == "talkative" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupId, client.Talkative); err == nil {
			if honor.CurrentTalkative.Uin != 0 {
				msg["current_talkative"] = MSG{
					"user_id":   honor.CurrentTalkative.Uin,
					"nickname":  honor.CurrentTalkative.Name,
					"avatar":    honor.CurrentTalkative.Avatar,
					"day_count": honor.CurrentTalkative.DayCount,
				}
			}
			msg["talkative_list"] = convertMem(honor.TalkativeList)
		}
	}

	if t == "performer" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupId, client.Performer); err == nil {
			msg["performer_lis"] = convertMem(honor.ActorList)
		}
	}

	if t == "legend" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupId, client.Legend); err == nil {
			msg["legend_list"] = convertMem(honor.LegendList)
		}
	}

	if t == "strong_newbie" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupId, client.StrongNewbie); err == nil {
			msg["strong_newbie_list"] = convertMem(honor.StrongNewbieList)
		}
	}

	if t == "emotion" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupId, client.Emotion); err == nil {
			msg["emotion_list"] = convertMem(honor.EmotionList)
		}
	}

	return OK(msg)
}

// https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_stranger_info-%E8%8E%B7%E5%8F%96%E9%99%8C%E7%94%9F%E4%BA%BA%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetStrangerInfo(userId int64) MSG {
	info, err := bot.Client.GetSummaryInfo(userId)
	if err != nil {
		return Failed(100, "SUMMARY_API_ERROR", err.Error())
	}
	return OK(MSG{
		"user_id":  info.Uin,
		"nickname": info.Nickname,
		"qid":      info.Qid,
		"sex": func() string {
			if info.Sex == 1 {
				return "female"
			} else if info.Sex == 0 {
				return "male"
			}
			// unknown = 0x2
			return "unknown"
		}(),
		"age":        info.Age,
		"level":      info.Level,
		"login_days": info.LoginDays,
	})
}

// https://cqhttp.cc/docs/4.15/#/API?id=-handle_quick_operation-%E5%AF%B9%E4%BA%8B%E4%BB%B6%E6%89%A7%E8%A1%8C%E5%BF%AB%E9%80%9F%E6%93%8D%E4%BD%9C
// https://github.com/richardchien/coolq-http-api/blob/master/src/cqhttp/plugins/web/http.cpp#L376
func (bot *CQBot) CQHandleQuickOperation(context, operation gjson.Result) MSG {
	postType := context.Get("post_type").Str
	switch postType {
	case "message":
		msgType := context.Get("message_type").Str
		reply := operation.Get("reply")
		if reply.Exists() {
			autoEscape := global.EnsureBool(operation.Get("auto_escape"), false)

			at := false
			if operation.Get("at_sender").Exists() {
				at = operation.Get("at_sender").Bool()
			}

			if at && reply.IsArray() {
				modified, err := sjson.Set(
					reply.Raw,
					"-1",
					MSG{
						"type": "at",
						"data": MSG{
							"qq": context.Get("sender.user_id").Int(),
						},
					},
				)
				if err != nil {
					return Failed(-1, "处理 at_sender 字段时出现错误", err.Error())
				}

				reply = gjson.Parse(modified)
			} else if at && reply.Type == gjson.String {
				reply = gjson.Parse(fmt.Sprintf(
					"\"%s[CQ:at,qq=%d]\"",
					reply.String(),
					context.Get("sender.user_id").Int(),
				))
			}

			if msgType == "group" {
				bot.CQSendGroupMessage(context.Get("group_id").Int(), reply, autoEscape)
			}
			if msgType == "private" {
				bot.CQSendPrivateMessage(context.Get("user_id").Int(), reply, autoEscape)
			}
		}
		if msgType == "group" {
			anonymous := context.Get("anonymous")
			isAnonymous := anonymous.Type == gjson.Null
			if operation.Get("delete").Bool() {
				bot.CQDeleteMessage(int32(context.Get("message_id").Int()))
			}
			if operation.Get("kick").Bool() && !isAnonymous {
				bot.CQSetGroupKick(context.Get("group_id").Int(), context.Get("user_id").Int(), "", operation.Get("reject_add_request").Bool())
			}
			if operation.Get("ban").Bool() {
				var duration uint32 = 30 * 60
				if operation.Get("ban_duration").Exists() {
					duration = uint32(operation.Get("ban_duration").Uint())
				}
				// unsupported anonymous ban yet
				if !isAnonymous {
					bot.CQSetGroupBan(context.Get("group_id").Int(), context.Get("user_id").Int(), duration)
				}
			}
		}
	case "request":
		reqType := context.Get("request_type").Str
		if operation.Get("approve").Exists() {
			if reqType == "friend" {
				bot.CQProcessFriendRequest(context.Get("flag").Str, operation.Get("approve").Bool())
			}
			if reqType == "group" {
				bot.CQProcessGroupRequest(context.Get("flag").Str, context.Get("sub_type").Str, operation.Get("reason").Str, operation.Get("approve").Bool())
			}
		}
	}
	return OK(nil)
}

func (bot *CQBot) CQGetImage(file string) MSG {
	if !global.PathExists(path.Join(global.ImagePath, file)) {
		return Failed(100)
	}
	if b, err := ioutil.ReadFile(path.Join(global.ImagePath, file)); err == nil {
		r := binary.NewReader(b)
		r.ReadBytes(16)
		msg := MSG{
			"size":     r.ReadInt32(),
			"filename": r.ReadString(),
			"url":      r.ReadString(),
		}
		local := path.Join(global.CachePath, file+"."+path.Ext(msg["filename"].(string)))
		if !global.PathExists(local) {
			if data, err := global.GetBytes(msg["url"].(string)); err == nil {
				_ = ioutil.WriteFile(local, data, 0644)
			}
		}
		msg["file"] = local
		return OK(msg)
	} else {
		return Failed(100, "LOAD_FILE_ERROR", err.Error())
	}
}

func (bot *CQBot) CQDownloadFile(url string, headers map[string]string, threadCount int) MSG {
	hash := md5.Sum([]byte(url))
	file := path.Join(global.CachePath, hex.EncodeToString(hash[:])+".cache")
	if global.PathExists(file) {
		if err := os.Remove(file); err != nil {
			log.Warnf("删除缓存文件 %v 时出现错误: %v", file, err)
			return Failed(100, "DELETE_FILE_ERROR", err.Error())
		}
	}
	if err := global.DownloadFileMultiThreading(url, file, 0, threadCount, headers); err != nil {
		log.Warnf("下载链接 %v 时出现错误: %v", url, err)
		return Failed(100, "DOWNLOAD_FILE_ERROR", err.Error())
	}
	abs, _ := filepath.Abs(file)
	return OK(MSG{
		"file": abs,
	})
}

func (bot *CQBot) CQGetForwardMessage(resId string) MSG {
	m := bot.Client.GetForwardMessage(resId)
	if m == nil {
		return Failed(100, "MSG_NOT_FOUND", "消息不存在")
	}
	r := make([]MSG, 0)
	for _, n := range m.Nodes {
		bot.checkMedia(n.Message)
		r = append(r, MSG{
			"sender": MSG{
				"user_id":  n.SenderId,
				"nickname": n.SenderName,
			},
			"time":    n.Time,
			"content": ToFormattedMessage(n.Message, 0, false),
		})
	}
	return OK(MSG{
		"messages": r,
	})
}

func (bot *CQBot) CQGetMessage(messageId int32) MSG {
	msg := bot.GetMessage(messageId)
	if msg == nil {
		return Failed(100, "MSG_NOT_FOUND", "消息不存在")
	}
	sender := msg["sender"].(message.Sender)
	gid, isGroup := msg["group"]
	raw := msg["message"].(string)
	return OK(MSG{
		"message_id":  messageId,
		"real_id":     msg["message-id"],
		"message_seq": msg["message-id"],
		"group":       isGroup,
		"group_id":    gid,
		"message_type": func() string {
			if isGroup {
				return "group"
			}
			return "private"
		}(),
		"sender": MSG{
			"user_id":  sender.Uin,
			"nickname": sender.Nickname,
		},
		"time":        msg["time"],
		"raw_message": raw,
		"message": ToFormattedMessage(bot.ConvertStringMessage(raw, isGroup), func() int64 {
			if isGroup {
				return gid.(int64)
			}
			return sender.Uin
		}(), false),
	})
}

func (bot *CQBot) CQGetGroupSystemMessages() MSG {
	msg, err := bot.Client.GetGroupSystemMessages()
	if err != nil {
		log.Warnf("获取群系统消息失败: %v", err)
		return Failed(100, "SYSTEM_MSG_API_ERROR", err.Error())
	}
	return OK(msg)
}

func (bot *CQBot) CQGetGroupMessageHistory(groupId int64, seq int64) MSG {
	if g := bot.Client.FindGroup(groupId); g == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	if seq == 0 {
		g, err := bot.Client.GetGroupInfo(groupId)
		if err != nil {
			return Failed(100, "GROUP_INFO_API_ERROR", err.Error())
		}
		seq = g.LastMsgSeq
	}
	msg, err := bot.Client.GetGroupMessages(groupId, int64(math.Max(float64(seq-19), 1)), seq)
	if err != nil {
		log.Warnf("获取群历史消息失败: %v", err)
		return Failed(100, "MESSAGES_API_ERROR", err.Error())
	}
	var ms []MSG
	for _, m := range msg {
		id := m.Id
		bot.checkMedia(m.Elements)
		if bot.db != nil {
			id = bot.InsertGroupMessage(m)
		}
		t := bot.formatGroupMessage(m)
		t["message_id"] = id
		ms = append(ms, t)
	}
	return OK(MSG{
		"messages": ms,
	})
}

func (bot *CQBot) CQGetOnlineClients(noCache bool) MSG {
	if noCache {
		if err := bot.Client.RefreshStatus(); err != nil {
			log.Warnf("刷新客户端状态时出现问题 %v", err)
			return Failed(100, "REFRESH_STATUS_ERROR", err.Error())
		}
	}
	var d []MSG
	for _, oc := range bot.Client.OnlineClients {
		d = append(d, MSG{
			"app_id":      oc.AppId,
			"device_name": oc.DeviceName,
			"device_kind": oc.DeviceKind,
		})
	}
	return OK(MSG{
		"clients": d,
	})
}

func (bot *CQBot) CQCanSendImage() MSG {
	return OK(MSG{"yes": true})
}

func (bot *CQBot) CQCanSendRecord() MSG {
	return OK(MSG{"yes": true})
}

func (bot *CQBot) CQOcrImage(imageId string) MSG {
	img, err := bot.makeImageOrVideoElem(map[string]string{"file": imageId}, false, true)
	if err != nil {
		log.Warnf("load image error: %v", err)
		return Failed(100, "LOAD_FILE_ERROR", err.Error())
	}
	rsp, err := bot.Client.ImageOcr(img)
	if err != nil {
		log.Warnf("ocr image error: %v", err)
		return Failed(100, "OCR_API_ERROR", err.Error())
	}
	return OK(rsp)
}

func (bot *CQBot) CQReloadEventFilter() MSG {
	global.BootFilter()
	return OK(nil)
}

func (bot *CQBot) CQSetGroupPortrait(groupId int64, file, cache string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		img, err := global.FindFile(file, cache, global.ImagePath)
		if err != nil {
			log.Warnf("set group portrait error: %v", err)
			return Failed(100, "LOAD_FILE_ERROR", err.Error())
		}
		g.UpdateGroupHeadPortrait(img)
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

func (bot *CQBot) CQSetGroupAnonymousBan(groupId int64, flag string, duration int32) MSG {
	if flag == "" {
		return Failed(100, "INVALID_FLAG", "无效的flag")
	}
	if g := bot.Client.FindGroup(groupId); g != nil {
		s := strings.SplitN(flag, "|", 2)
		if len(s) != 2 {
			return Failed(100, "INVALID_FLAG", "无效的flag")
		}
		id := s[0]
		nick := s[1]
		if err := g.MuteAnonymous(id, nick, duration); err != nil {
			log.Warnf("anonymous ban error: %v", err)
			return Failed(100, "CALL_API_ERROR", err.Error())
		}
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_status-%E8%8E%B7%E5%8F%96%E8%BF%90%E8%A1%8C%E7%8A%B6%E6%80%81
func (bot *CQBot) CQGetStatus() MSG {
	return OK(MSG{
		"app_initialized": true,
		"app_enabled":     true,
		"plugins_good":    nil,
		"app_good":        true,
		"online":          bot.Client.Online,
		"good":            bot.Client.Online,
		"stat":            bot.Client.GetStatistics(),
	})
}

// CQSetEssenceMessage 设置精华消息
func (bot *CQBot) CQSetEssenceMessage(messageID int32) MSG {
	msg := bot.GetMessage(messageID)
	if msg == nil {
		return Failed(100, "MESSAGE_NOT_FOUND", "消息不存在")
	}
	if _, ok := msg["group"]; ok {
		if err := bot.Client.SetEssenceMessage(msg["group"].(int64), msg["message-id"].(int32), msg["internal-id"].(int32)); err != nil {
			log.Warnf("设置精华消息 %v 失败: %v", messageID, err)
			return Failed(100, "SET_ESSENCE_MSG_ERROR", err.Error())
		}
	} else {
		log.Warnf("设置精华消息 %v 失败: 非群聊", messageID)
		return Failed(100, "SET_ESSENCE_MSG_ERROR", "非群聊")
	}
	return OK(nil)
}

// CQDeleteEssenceMessage 移出精华消息
func (bot *CQBot) CQDeleteEssenceMessage(messageID int32) MSG {
	msg := bot.GetMessage(messageID)
	if msg == nil {
		return Failed(100, "MESSAGE_NOT_FOUND", "消息不存在")
	}
	if _, ok := msg["group"]; ok {
		if err := bot.Client.DeleteEssenceMessage(msg["group"].(int64), msg["message-id"].(int32), msg["internal-id"].(int32)); err != nil {
			log.Warnf("移出精华消息 %v 失败: %v", messageID, err)
			return Failed(100, "DEL_ESSENCE_MSG_ERROR", err.Error())
		}
	} else {
		log.Warnf("移出精华消息 %v 失败: 非群聊", messageID)
		return Failed(100, "DEL_ESSENCE_MSG_ERROR", "非群聊")
	}
	return OK(nil)
}

// CQGetEssenceMessageList 获取精华消息列表
func (bot *CQBot) CQGetEssenceMessageList(groupCode int64) MSG {
	g := bot.Client.FindGroup(groupCode)
	if g == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	msgList, err := bot.Client.GetGroupEssenceMsgList(groupCode)
	if err != nil {
		return Failed(100, "GET_ESSENCE_LIST_FOUND", err.Error())
	}
	list := make([]MSG, 0)
	for _, m := range msgList {
		var msg = MSG{
			"sender_nick":   m.SenderNick,
			"sender_time":   m.SenderTime,
			"operator_time": m.AddDigestTime,
			"operator_nick": m.AddDigestNick,
		}
		msg["sender_id"], _ = strconv.ParseUint(m.SenderUin, 10, 64)
		msg["operator_id"], _ = strconv.ParseUint(m.AddDigestUin, 10, 64)
		msg["message_id"] = ToGlobalId(groupCode, int32(m.MessageID))
		list = append(list, msg)
	}
	return OK(list)
}

func (bot *CQBot) CQCheckUrlSafely(url string) MSG {
	return OK(MSG{
		"level": bot.Client.CheckUrlSafely(url),
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
		"version":                    Version,
		"protocol": func() int {
			switch client.SystemDeviceInfo.Protocol {
			case client.IPad:
				return 0
			case client.AndroidPhone:
				return 1
			case client.AndroidWatch:
				return 2
			case client.MacOS:
				return 3
			default:
				return -1
			}
		}(),
	})
}

func OK(data interface{}) MSG {
	return MSG{"data": data, "retcode": 0, "status": "ok"}
}

func Failed(code int, msg ...string) MSG {
	m := ""
	w := ""
	if len(msg) > 0 {
		m = msg[0]
	}
	if len(msg) > 1 {
		w = msg[1]
	}
	return MSG{"data": nil, "retcode": code, "msg": m, "wording": w, "status": "failed"}
}

func convertGroupMemberInfo(groupId int64, m *client.GroupMemberInfo) MSG {
	return MSG{
		"group_id": groupId,
		"user_id":  m.Uin,
		"nickname": m.Nickname,
		"card":     m.CardName,
		"sex": func() string {
			if m.Gender == 1 {
				return "female"
			} else if m.Gender == 0 {
				return "male"
			}
			// unknown = 0xff
			return "unknown"
		}(),
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

func limitedString(str string) string {
	if strings.Count(str, "") <= 10 {
		return str
	}
	limited := []rune(str)
	limited = limited[:10]
	return string(limited) + " ..."
}
