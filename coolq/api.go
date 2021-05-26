package coolq

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/Mrs4s/go-cqhttp/global"
)

// Version go-cqhttp的版本信息，在编译时使用ldflags进行覆盖
var Version = "unknown"

func init() {
	if Version != "unknown" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if ok {
		Version = info.Main.Version
	}
}

// CQGetLoginInfo 获取登录号信息
//
// https://git.io/Jtz1I
func (bot *CQBot) CQGetLoginInfo() MSG {
	return OK(MSG{"user_id": bot.Client.Uin, "nickname": bot.Client.Nickname})
}

// CQGetQiDianAccountInfo 获取企点账号信息
func (bot *CQBot) CQGetQiDianAccountInfo() MSG {
	if bot.Client.QiDian == nil {
		return Failed(100, "QIDIAN_PROTOCOL_REQUEST", "请使用企点协议")
	}
	return OK(MSG{
		"master_id":   bot.Client.QiDian.MasterUin,
		"ext_name":    bot.Client.QiDian.ExtName,
		"create_time": bot.Client.QiDian.CreateTime,
	})
}

// CQGetFriendList 获取好友列表
//
// https://git.io/Jtz1L
func (bot *CQBot) CQGetFriendList() MSG {
	fs := make([]MSG, 0, len(bot.Client.FriendList))
	for _, f := range bot.Client.FriendList {
		fs = append(fs, MSG{
			"nickname": f.Nickname,
			"remark":   f.Remark,
			"user_id":  f.Uin,
		})
	}
	return OK(fs)
}

// CQDeleteFriend 删除好友
//
//
func (bot *CQBot) CQDeleteFriend(uin int64) MSG {
	if bot.Client.FindFriend(uin) == nil {
		return Failed(100, "FRIEND_NOT_FOUND", "好友不存在")
	}
	if err := bot.Client.DeleteFriend(uin); err != nil {
		log.Errorf("删除好友时出现错误: %v", err)
		return Failed(100, "DELETE_API_ERROR", err.Error())
	}
	return OK(nil)
}

// CQGetGroupList 获取群列表
//
// https://git.io/Jtz1t
func (bot *CQBot) CQGetGroupList(noCache bool) MSG {
	gs := make([]MSG, 0, len(bot.Client.GroupList))
	if noCache {
		_ = bot.Client.ReloadGroupList()
	}
	for _, g := range bot.Client.GroupList {
		gs = append(gs, MSG{
			"group_id":          g.Code,
			"group_name":        g.Name,
			"group_memo":        g.Memo,
			"group_create_time": g.GroupCreateTime,
			"group_level":       g.GroupLevel,
			"max_member_count":  g.MaxMemberCount,
			"member_count":      g.MemberCount,
		})
	}
	return OK(gs)
}

// CQGetGroupInfo 获取群信息
//
// https://git.io/Jtz1O
func (bot *CQBot) CQGetGroupInfo(groupID int64, noCache bool) MSG {
	group := bot.Client.FindGroup(groupID)
	if group == nil || noCache {
		group, _ = bot.Client.GetGroupInfo(groupID)
	}
	if group == nil {
		gid := strconv.FormatInt(groupID, 10)
		info, err := bot.Client.SearchGroupByKeyword(gid)
		if err != nil {
			return Failed(100, "GROUP_SEARCH_ERROR", "群聊搜索失败")
		}
		for _, g := range info {
			if g.Code == groupID {
				return OK(MSG{
					"group_id":          g.Code,
					"group_name":        g.Name,
					"group_memo":        g.Memo,
					"group_create_time": 0,
					"group_level":       0,
					"max_member_count":  0,
					"member_count":      0,
				})
			}
		}
	} else {
		return OK(MSG{
			"group_id":          group.Code,
			"group_name":        group.Name,
			"group_memo":        group.Memo,
			"group_create_time": group.GroupCreateTime,
			"group_level":       group.GroupLevel,
			"max_member_count":  group.MaxMemberCount,
			"member_count":      group.MemberCount,
		})
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQGetGroupMemberList 获取群成员列表
//
// https://git.io/Jtz13
func (bot *CQBot) CQGetGroupMemberList(groupID int64, noCache bool) MSG {
	group := bot.Client.FindGroup(groupID)
	if group == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	if noCache {
		t, err := bot.Client.GetGroupMembers(group)
		if err != nil {
			log.Warnf("刷新群 %v 成员列表失败: %v", groupID, err)
			return Failed(100, "GET_MEMBERS_API_ERROR", err.Error())
		}
		group.Members = t
	}
	members := make([]MSG, 0, len(group.Members))
	for _, m := range group.Members {
		members = append(members, convertGroupMemberInfo(groupID, m))
	}
	return OK(members)
}

// CQGetGroupMemberInfo 获取群成员信息
//
// https://git.io/Jtz1s
func (bot *CQBot) CQGetGroupMemberInfo(groupID, userID int64, noCache bool) MSG {
	group := bot.Client.FindGroup(groupID)
	if group == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	var member *client.GroupMemberInfo
	if noCache {
		var err error
		member, err = bot.Client.GetMemberInfo(groupID, userID)
		if err != nil {
			log.Warnf("刷新群 %v 中成员 %v 失败: %v", groupID, userID, err)
			return Failed(100, "GET_MEMBER_INFO_API_ERROR", err.Error())
		}
	} else {
		member = group.FindMember(userID)
	}
	if member == nil {
		return Failed(100, "MEMBER_NOT_FOUND", "群员不存在")
	}
	return OK(convertGroupMemberInfo(groupID, member))
}

// CQGetGroupFileSystemInfo 扩展API-获取群文件系统信息
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%96%87%E4%BB%B6%E7%B3%BB%E7%BB%9F%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetGroupFileSystemInfo(groupID int64) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupID)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(fs)
}

// CQGetGroupRootFiles 扩展API-获取群根目录文件列表
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%A0%B9%E7%9B%AE%E5%BD%95%E6%96%87%E4%BB%B6%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetGroupRootFiles(groupID int64) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupID)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	files, folders, err := fs.Root()
	if err != nil {
		log.Errorf("获取群 %v 根目录文件失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(MSG{
		"files":   files,
		"folders": folders,
	})
}

// CQGetGroupFilesByFolderID 扩展API-获取群子目录文件列表
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%BE%A4%E5%AD%90%E7%9B%AE%E5%BD%95%E6%96%87%E4%BB%B6%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetGroupFilesByFolderID(groupID int64, folderID string) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupID)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	files, folders, err := fs.GetFilesByFolder(folderID)
	if err != nil {
		log.Errorf("获取群 %v 根目录 %v 子文件失败: %v", groupID, folderID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(MSG{
		"files":   files,
		"folders": folders,
	})
}

// CQGetGroupFileURL 扩展API-获取群文件资源链接
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%96%87%E4%BB%B6%E8%B5%84%E6%BA%90%E9%93%BE%E6%8E%A5
func (bot *CQBot) CQGetGroupFileURL(groupID int64, fileID string, busID int32) MSG {
	url := bot.Client.GetGroupFileUrl(groupID, fileID, busID)
	if url == "" {
		return Failed(100, "FILE_SYSTEM_API_ERROR")
	}
	return OK(MSG{
		"url": url,
	})
}

// CQUploadGroupFile 扩展API-上传群文件
//
// https://docs.go-cqhttp.org/api/#%E4%B8%8A%E4%BC%A0%E7%BE%A4%E6%96%87%E4%BB%B6
func (bot *CQBot) CQUploadGroupFile(groupID int64, file, name, folder string) MSG {
	if !global.PathExists(file) {
		log.Errorf("上传群文件 %v 失败: 文件不存在", file)
		return Failed(100, "FILE_NOT_FOUND", "文件不存在")
	}
	fs, err := bot.Client.GetGroupFileSystem(groupID)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	if folder == "" {
		folder = "/"
	}
	if err = fs.UploadFile(file, name, folder); err != nil {
		log.Errorf("上传群 %v 文件 %v 失败: %v", groupID, file, err)
		return Failed(100, "FILE_SYSTEM_UPLOAD_API_ERROR", err.Error())
	}
	return OK(nil)
}

// CQGroupFileCreateFolder 拓展API-创建群文件文件夹
//
//
func (bot *CQBot) CQGroupFileCreateFolder(groupID int64, parentID, name string) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupID)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	if err = fs.CreateFolder(parentID, name); err != nil {
		log.Errorf("创建群 %v 文件夹失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(nil)
}

// CQGroupFileDeleteFolder 拓展API-删除群文件文件夹
//
//
func (bot *CQBot) CQGroupFileDeleteFolder(groupID int64, id string) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupID)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	if err = fs.DeleteFolder(id); err != nil {
		log.Errorf("删除群 %v 文件夹 %v 时出现文件: %v", groupID, id, err)
		return Failed(200, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	return OK(nil)
}

// CQGroupFileDeleteFile 拓展API-删除群文件
//
//
func (bot *CQBot) CQGroupFileDeleteFile(groupID int64, parentID, id string, busID int32) MSG {
	fs, err := bot.Client.GetGroupFileSystem(groupID)
	if err != nil {
		log.Errorf("获取群 %v 文件系统信息失败: %v", groupID, err)
		return Failed(100, "FILE_SYSTEM_API_ERROR", err.Error())
	}
	if res := fs.DeleteFile(parentID, id, busID); res != "" {
		log.Errorf("删除群 %v 文件 %v 时出现文件: %v", groupID, id, res)
		return Failed(200, "FILE_SYSTEM_API_ERROR", res)
	}
	return OK(nil)
}

// CQGetWordSlices 隐藏API-获取中文分词
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E4%B8%AD%E6%96%87%E5%88%86%E8%AF%8D-%E9%9A%90%E8%97%8F-api
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

// CQSendGroupMessage 发送群消息
//
// https://git.io/Jtz1c
func (bot *CQBot) CQSendGroupMessage(groupID int64, i interface{}, autoEscape bool) MSG {
	var str string
	group := bot.Client.FindGroup(groupID)
	if group == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	fixAt := func(elem []message.IMessageElement) {
		for _, e := range elem {
			if at, ok := e.(*message.AtElement); ok && at.Target != 0 && at.Display == "" {
				at.Display = "@" + func() string {
					mem := group.FindMember(at.Target)
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
			mid := bot.SendGroupMessage(groupID, &message.SendingMessage{Elements: elem})
			if mid == -1 {
				return Failed(100, "SEND_MSG_API_ERROR", "请参考 go-cqhttp 端输出")
			}
			log.Infof("发送群 %v(%v) 的消息: %v (%v)", group.Name, groupID, limitedString(ToStringMessage(elem, groupID)), mid)
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
	mid := bot.SendGroupMessage(groupID, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return Failed(100, "SEND_MSG_API_ERROR", "请参考 go-cqhttp 端输出")
	}
	log.Infof("发送群 %v(%v) 的消息: %v (%v)", group.Name, groupID, limitedString(str), mid)
	return OK(MSG{"message_id": mid})
}

// CQSendGroupForwardMessage 扩展API-发送合并转发(群)
//
// https://docs.go-cqhttp.org/api/#%E5%8F%91%E9%80%81%E5%90%88%E5%B9%B6%E8%BD%AC%E5%8F%91-%E7%BE%A4
func (bot *CQBot) CQSendGroupForwardMessage(groupID int64, m gjson.Result) MSG {
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
					Message:    []message.IMessageElement{bot.Client.UploadGroupForwardMessage(groupID, &message.ForwardMessage{Nodes: taowa})},
				})
				return
			}
		}
		content := bot.ConvertObjectMessage(e.Get("data.content"), true)
		if uin != 0 && name != "" && len(content) > 0 {
			var newElem []message.IMessageElement
			for _, elem := range content {
				if img, ok := elem.(*LocalImageElement); ok {
					gm, err := bot.UploadLocalImageAsGroup(groupID, img)
					if err != nil {
						log.Warnf("警告：群 %v 图片上传失败: %v", groupID, err)
						continue
					}
					newElem = append(newElem, gm)
					continue
				}
				if video, ok := elem.(*LocalVideoElement); ok {
					gm, err := bot.UploadLocalVideo(groupID, video)
					if err != nil {
						log.Warnf("警告：群 %v 视频上传失败: %v", groupID, err)
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
		ret := bot.Client.SendGroupForwardMessage(groupID, &message.ForwardMessage{Nodes: sendNodes})
		if ret == nil || ret.Id == -1 {
			log.Warnf("合并转发(群)消息发送失败: 账号可能被风控.")
			return Failed(100, "SEND_MSG_API_ERROR", "请参考 go-cqhttp 端输出")
		}
		return OK(MSG{
			"message_id": bot.InsertGroupMessage(ret),
		})
	}
	return Failed(100)
}

// CQSendPrivateMessage 发送私聊消息
//
// https://git.io/Jtz1l
func (bot *CQBot) CQSendPrivateMessage(userID int64, groupID int64, i interface{}, autoEscape bool) MSG {
	var str string
	if m, ok := i.(gjson.Result); ok {
		if m.Type == gjson.JSON {
			elem := bot.ConvertObjectMessage(m, false)
			mid := bot.SendPrivateMessage(userID, groupID, &message.SendingMessage{Elements: elem})
			if mid == -1 {
				return Failed(100, "SEND_MSG_API_ERROR", "请参考 go-cqhttp 端输出")
			}
			log.Infof("发送好友 %v(%v)  的消息: %v (%v)", userID, userID, limitedString(m.String()), mid)
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
	mid := bot.SendPrivateMessage(userID, groupID, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return Failed(100, "SEND_MSG_API_ERROR", "请参考 go-cqhttp 端输出")
	}
	log.Infof("发送好友 %v(%v)  的消息: %v (%v)", userID, userID, limitedString(str), mid)
	return OK(MSG{"message_id": mid})
}

// CQSetGroupCard 设置群名片(群备注)
//
// https://git.io/Jtz1B
func (bot *CQBot) CQSetGroupCard(groupID, userID int64, card string) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		if m := g.FindMember(userID); m != nil {
			m.EditCard(card)
			return OK(nil)
		}
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQSetGroupSpecialTitle 设置群组专属头衔
//
// https://git.io/Jtz10
func (bot *CQBot) CQSetGroupSpecialTitle(groupID, userID int64, title string) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		if m := g.FindMember(userID); m != nil {
			m.EditSpecialTitle(title)
			return OK(nil)
		}
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQSetGroupName 设置群名
//
// https://git.io/Jtz12
func (bot *CQBot) CQSetGroupName(groupID int64, name string) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		g.UpdateName(name)
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQSetGroupMemo 扩展API-发送群公告
//
// https://docs.go-cqhttp.org/api/#%E5%8F%91%E9%80%81%E7%BE%A4%E5%85%AC%E5%91%8A
func (bot *CQBot) CQSetGroupMemo(groupID int64, msg string, img string) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		if g.SelfPermission() == client.Member {
			return Failed(100, "PERMISSION_DENIED", "权限不足")
		}
		if img != "" {
			data, err := global.FindFile(img, "", global.ImagePath)
			if err != nil {
				return Failed(100, "IMAGE_NOT_FOUND", "图片未找到")
			}
			err = bot.Client.AddGroupNoticeWithPic(groupID, msg, data)
			if err != nil {
				return Failed(100, "SEND_NOTICE_ERROR", err.Error())
			}
		} else {
			err := bot.Client.AddGroupNoticeSimple(groupID, msg)
			if err != nil {
				return Failed(100, "SEND_NOTICE_ERROR", err.Error())
			}
		}
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQSetGroupKick 群组踢人
//
// https://git.io/Jtz1V
func (bot *CQBot) CQSetGroupKick(groupID, userID int64, msg string, block bool) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		if m := g.FindMember(userID); m != nil {
			err := m.Kick(msg, block)
			if err != nil {
				return Failed(100, "NOT_MANAGEABLE", "机器人权限不足")
			}
			return OK(nil)
		}
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQSetGroupBan 群组单人禁言
//
// https://git.io/Jtz1w
func (bot *CQBot) CQSetGroupBan(groupID, userID int64, duration uint32) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		if m := g.FindMember(userID); m != nil {
			err := m.Mute(duration)
			if err != nil {
				if duration > 2592000 {
					return Failed(100, "DURATION_IS_NOT_IN_RANGE", "非法的禁言时长")
				}
				return Failed(100, "NOT_MANAGEABLE", "机器人权限不足")
			}
			return OK(nil)
		}
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQSetGroupWholeBan 群组全员禁言
//
// https://git.io/Jtz1o
func (bot *CQBot) CQSetGroupWholeBan(groupID int64, enable bool) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		g.MuteAll(enable)
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQSetGroupLeave 退出群组
//
// https://git.io/Jtz1K
func (bot *CQBot) CQSetGroupLeave(groupID int64) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		g.Quit()
		return OK(nil)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQGetAtAllRemain 扩展API-获取群 @全体成员 剩余次数
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%BE%A4-%E5%85%A8%E4%BD%93%E6%88%90%E5%91%98-%E5%89%A9%E4%BD%99%E6%AC%A1%E6%95%B0
func (bot *CQBot) CQGetAtAllRemain(groupID int64) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
		i, err := bot.Client.GetAtAllRemain(groupID)
		if err != nil {
			return Failed(100, "GROUP_REMAIN_API_ERROR", err.Error())
		}
		return OK(i)
	}
	return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
}

// CQProcessFriendRequest 处理加好友请求
//
// https://git.io/Jtz11
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

// CQProcessGroupRequest 处理加群请求／邀请
//
// https://git.io/Jtz1D
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

// CQDeleteMessage 撤回消息
//
// https:// git.io/Jtz1y
func (bot *CQBot) CQDeleteMessage(messageID int32) MSG {
	msg := bot.GetMessage(messageID)
	if msg == nil {
		return Failed(100, "MESSAGE_NOT_FOUND", "消息不存在")
	}
	if _, ok := msg["group"]; ok {
		if msg["internal-id"] == nil {
			// TODO 撤回临时对话消息
			log.Warnf("撤回 %v 失败: 无法撤回临时对话消息", messageID)
			return Failed(100, "CANNOT_RECALL_TEMP_MSG", "无法撤回临时对话消息")
		}
		if err := bot.Client.RecallGroupMessage(msg["group"].(int64), msg["message-id"].(int32), msg["internal-id"].(int32)); err != nil {
			log.Warnf("撤回 %v 失败: %v", messageID, err)
			return Failed(100, "RECALL_API_ERROR", err.Error())
		}
	} else {
		if msg["sender"].(message.Sender).Uin != bot.Client.Uin {
			log.Warnf("撤回 %v 失败: 好友会话无法撤回对方消息.", messageID)
			return Failed(100, "CANNOT_RECALL_FRIEND_MSG", "无法撤回对方消息")
		}
		if err := bot.Client.RecallPrivateMessage(msg["target"].(int64), int64(msg["time"].(int32)), msg["message-id"].(int32), msg["internal-id"].(int32)); err != nil {
			log.Warnf("撤回 %v 失败: %v", messageID, err)
			return Failed(100, "RECALL_API_ERROR", err.Error())
		}
	}
	return OK(nil)
}

// CQSetGroupAdmin 群组设置管理员
//
// https://git.io/Jtz1S
func (bot *CQBot) CQSetGroupAdmin(groupID, userID int64, enable bool) MSG {
	group := bot.Client.FindGroup(groupID)
	if group == nil || group.OwnerUin != bot.Client.Uin {
		return Failed(100, "PERMISSION_DENIED", "群不存在或权限不足")
	}
	mem := group.FindMember(userID)
	if mem == nil {
		return Failed(100, "GROUP_MEMBER_NOT_FOUND", "群成员不存在")
	}
	mem.SetAdmin(enable)
	t, err := bot.Client.GetGroupMembers(group)
	if err != nil {
		log.Warnf("刷新群 %v 成员列表失败: %v", groupID, err)
		return Failed(100, "GET_MEMBERS_API_ERROR", err.Error())
	}
	group.Members = t
	return OK(nil)
}

// CQGetVipInfo 扩展API-获取VIP信息
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96vip%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetVipInfo(userID int64) MSG {
	vip, err := bot.Client.GetVipInfo(userID)
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

// CQGetGroupHonorInfo 获取群荣誉信息
//
// https://git.io/Jtz1H
func (bot *CQBot) CQGetGroupHonorInfo(groupID int64, t string) MSG {
	msg := MSG{"group_id": groupID}
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
		if honor, err := bot.Client.GetGroupHonorInfo(groupID, client.Talkative); err == nil {
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
		if honor, err := bot.Client.GetGroupHonorInfo(groupID, client.Performer); err == nil {
			msg["performer_lis"] = convertMem(honor.ActorList)
		}
	}

	if t == "legend" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupID, client.Legend); err == nil {
			msg["legend_list"] = convertMem(honor.LegendList)
		}
	}

	if t == "strong_newbie" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupID, client.StrongNewbie); err == nil {
			msg["strong_newbie_list"] = convertMem(honor.StrongNewbieList)
		}
	}

	if t == "emotion" || t == "all" {
		if honor, err := bot.Client.GetGroupHonorInfo(groupID, client.Emotion); err == nil {
			msg["emotion_list"] = convertMem(honor.EmotionList)
		}
	}

	return OK(msg)
}

// CQGetStrangerInfo 获取陌生人信息
//
// https://git.io/Jtz17
func (bot *CQBot) CQGetStrangerInfo(userID int64) MSG {
	info, err := bot.Client.GetSummaryInfo(userID)
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

// CQHandleQuickOperation 隐藏API-对事件执行快速操作
//
// https://git.io/Jtz15
func (bot *CQBot) CQHandleQuickOperation(context, operation gjson.Result) MSG {
	postType := context.Get("post_type").Str

	switch postType {
	case "message":
		anonymous := context.Get("anonymous")
		isAnonymous := anonymous.Type != gjson.Null
		msgType := context.Get("message_type").Str
		reply := operation.Get("reply")

		if reply.Exists() {
			autoEscape := global.EnsureBool(operation.Get("auto_escape"), false)
			at := operation.Get("at_sender").Bool() && !isAnonymous && msgType == "group"
			if at && reply.IsArray() {
				// 在 reply 数组头部插入CQ码
				replySegments := make([]MSG, 0)
				segments := make([]MSG, 0)
				segments = append(segments, MSG{
					"type": "at",
					"data": MSG{
						"qq": context.Get("sender.user_id").Int(),
					},
				})

				err := json.UnmarshalFromString(reply.Raw, &replySegments)
				if err != nil {
					log.WithError(err).Warnf("处理 at_sender 过程中发生错误")
					return Failed(-1, "处理 at_sender 过程中发生错误", err.Error())
				}

				segments = append(segments, replySegments...)

				modified, err := json.MarshalToString(segments)
				if err != nil {
					log.WithError(err).Warnf("处理 at_sender 过程中发生错误")
					return Failed(-1, "处理 at_sender 过程中发生错误", err.Error())
				}

				reply = gjson.Parse(modified)
			} else if at && reply.Type == gjson.String {
				reply = gjson.Parse(fmt.Sprintf(
					"\"[CQ:at,qq=%d]%s\"",
					context.Get("sender.user_id").Int(),
					reply.String(),
				))
			}

			if msgType == "group" {
				bot.CQSendGroupMessage(context.Get("group_id").Int(), reply, autoEscape)
			}
			if msgType == "private" {
				bot.CQSendPrivateMessage(context.Get("user_id").Int(), context.Get("group_id").Int(), reply, autoEscape)
			}
		}
		if msgType == "group" {
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

// CQGetImage 获取图片(修改自OneBot)
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E5%9B%BE%E7%89%87%E4%BF%A1%E6%81%AF
func (bot *CQBot) CQGetImage(file string) MSG {
	if !global.PathExists(path.Join(global.ImagePath, file)) {
		return Failed(100)
	}
	b, err := ioutil.ReadFile(path.Join(global.ImagePath, file))
	if err == nil {
		r := binary.NewReader(b)
		r.ReadBytes(16)
		msg := MSG{
			"size":     r.ReadInt32(),
			"filename": r.ReadString(),
			"url":      r.ReadString(),
		}
		local := path.Join(global.CachePath, file+"."+path.Ext(msg["filename"].(string)))
		if !global.PathExists(local) {
			f, _ := os.OpenFile(local, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o0644)
			if body, err := global.HTTPGetReadCloser(msg["url"].(string)); err == nil {
				_, _ = f.ReadFrom(body)
				_ = body.Close()
			}
			f.Close()
		}
		msg["file"] = local
		return OK(msg)
	}
	return Failed(100, "LOAD_FILE_ERROR", err.Error())
}

// CQDownloadFile 扩展API-下载文件到缓存目录
//
// https://docs.go-cqhttp.org/api/#%E4%B8%8B%E8%BD%BD%E6%96%87%E4%BB%B6%E5%88%B0%E7%BC%93%E5%AD%98%E7%9B%AE%E5%BD%95
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

// CQGetForwardMessage 获取合并转发消息
//
// https://git.io/Jtz1F
func (bot *CQBot) CQGetForwardMessage(resID string) MSG {
	m := bot.Client.GetForwardMessage(resID)
	if m == nil {
		return Failed(100, "MSG_NOT_FOUND", "消息不存在")
	}
	r := make([]MSG, 0, len(m.Nodes))
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

// CQGetMessage 获取消息
//
// https://git.io/Jtz1b
func (bot *CQBot) CQGetMessage(messageID int32) MSG {
	msg := bot.GetMessage(messageID)
	if msg == nil {
		return Failed(100, "MSG_NOT_FOUND", "消息不存在")
	}
	sender := msg["sender"].(message.Sender)
	gid, isGroup := msg["group"]
	raw := msg["message"].(string)
	return OK(MSG{
		"message_id":  messageID,
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
			return 0
		}(), false),
	})
}

// CQGetGroupSystemMessages 扩展API-获取群文件系统消息
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%BE%A4%E7%B3%BB%E7%BB%9F%E6%B6%88%E6%81%AF
func (bot *CQBot) CQGetGroupSystemMessages() MSG {
	msg, err := bot.Client.GetGroupSystemMessages()
	if err != nil {
		log.Warnf("获取群系统消息失败: %v", err)
		return Failed(100, "SYSTEM_MSG_API_ERROR", err.Error())
	}
	return OK(msg)
}

// CQGetGroupMessageHistory 获取群消息历史记录
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%B6%88%E6%81%AF%E5%8E%86%E5%8F%B2%E8%AE%B0%E5%BD%95
func (bot *CQBot) CQGetGroupMessageHistory(groupID int64, seq int64) MSG {
	if g := bot.Client.FindGroup(groupID); g == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	if seq == 0 {
		g, err := bot.Client.GetGroupInfo(groupID)
		if err != nil {
			return Failed(100, "GROUP_INFO_API_ERROR", err.Error())
		}
		seq = g.LastMsgSeq
	}
	msg, err := bot.Client.GetGroupMessages(groupID, int64(math.Max(float64(seq-19), 1)), seq)
	if err != nil {
		log.Warnf("获取群历史消息失败: %v", err)
		return Failed(100, "MESSAGES_API_ERROR", err.Error())
	}
	ms := make([]MSG, 0, len(msg))
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

// CQGetOnlineClients 扩展API-获取当前账号在线客户端列表
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E5%BD%93%E5%89%8D%E8%B4%A6%E5%8F%B7%E5%9C%A8%E7%BA%BF%E5%AE%A2%E6%88%B7%E7%AB%AF%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetOnlineClients(noCache bool) MSG {
	if noCache {
		if err := bot.Client.RefreshStatus(); err != nil {
			log.Warnf("刷新客户端状态时出现问题 %v", err)
			return Failed(100, "REFRESH_STATUS_ERROR", err.Error())
		}
	}
	d := make([]MSG, 0, len(bot.Client.OnlineClients))
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

// CQCanSendImage 检查是否可以发送图片(此处永远返回true)
//
// https://git.io/Jtz1N
func (bot *CQBot) CQCanSendImage() MSG {
	return OK(MSG{"yes": true})
}

// CQCanSendRecord 检查是否可以发送语音(此处永远返回true)
//
// https://git.io/Jtz1x
func (bot *CQBot) CQCanSendRecord() MSG {
	return OK(MSG{"yes": true})
}

// CQOcrImage 扩展API-图片OCR
//
// https://docs.go-cqhttp.org/api/#%E5%9B%BE%E7%89%87-ocr
func (bot *CQBot) CQOcrImage(imageID string) MSG {
	img, err := bot.makeImageOrVideoElem(map[string]string{"file": imageID}, false, true)
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

// CQSetGroupPortrait 扩展API-设置群头像
//
// https://docs.go-cqhttp.org/api/#%E8%AE%BE%E7%BD%AE%E7%BE%A4%E5%A4%B4%E5%83%8F
func (bot *CQBot) CQSetGroupPortrait(groupID int64, file, cache string) MSG {
	if g := bot.Client.FindGroup(groupID); g != nil {
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

// CQSetGroupAnonymousBan 群组匿名用户禁言
//
// https://git.io/Jtz1p
func (bot *CQBot) CQSetGroupAnonymousBan(groupID int64, flag string, duration int32) MSG {
	if flag == "" {
		return Failed(100, "INVALID_FLAG", "无效的flag")
	}
	if g := bot.Client.FindGroup(groupID); g != nil {
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

// CQGetStatus 获取运行状态
//
// https://git.io/JtzMe
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

// CQSetEssenceMessage 扩展API-设置精华消息
//
// https://docs.go-cqhttp.org/api/#%E8%AE%BE%E7%BD%AE%E7%B2%BE%E5%8D%8E%E6%B6%88%E6%81%AF
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

// CQDeleteEssenceMessage 扩展API-移出精华消息
//
// https://docs.go-cqhttp.org/api/#%E7%A7%BB%E5%87%BA%E7%B2%BE%E5%8D%8E%E6%B6%88%E6%81%AF
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

// CQGetEssenceMessageList 扩展API-获取精华消息列表
//
// https://docs.go-cqhttp.org/api/#%E8%8E%B7%E5%8F%96%E7%B2%BE%E5%8D%8E%E6%B6%88%E6%81%AF%E5%88%97%E8%A1%A8
func (bot *CQBot) CQGetEssenceMessageList(groupCode int64) MSG {
	g := bot.Client.FindGroup(groupCode)
	if g == nil {
		return Failed(100, "GROUP_NOT_FOUND", "群聊不存在")
	}
	msgList, err := bot.Client.GetGroupEssenceMsgList(groupCode)
	if err != nil {
		return Failed(100, "GET_ESSENCE_LIST_FOUND", err.Error())
	}
	list := make([]MSG, 0, len(msgList))
	for _, m := range msgList {
		msg := MSG{
			"sender_nick":   m.SenderNick,
			"sender_time":   m.SenderTime,
			"operator_time": m.AddDigestTime,
			"operator_nick": m.AddDigestNick,
		}
		msg["sender_id"], _ = strconv.ParseUint(m.SenderUin, 10, 64)
		msg["operator_id"], _ = strconv.ParseUint(m.AddDigestUin, 10, 64)
		msg["message_id"] = toGlobalID(groupCode, int32(m.MessageID))
		list = append(list, msg)
	}
	return OK(list)
}

// CQCheckURLSafely 扩展API-检查链接安全性
//
// https://docs.go-cqhttp.org/api/#%E6%A3%80%E6%9F%A5%E9%93%BE%E6%8E%A5%E5%AE%89%E5%85%A8%E6%80%A7
func (bot *CQBot) CQCheckURLSafely(url string) MSG {
	return OK(MSG{
		"level": bot.Client.CheckUrlSafely(url),
	})
}

// CQGetVersionInfo 获取版本信息
//
// https://git.io/JtwUs
func (bot *CQBot) CQGetVersionInfo() MSG {
	wd, _ := os.Getwd()
	return OK(MSG{
		"app_name":                   "go-cqhttp",
		"app_version":                Version,
		"app_full_name":              fmt.Sprintf("go-cqhttp-%s_%s_%s-%s", Version, runtime.GOOS, runtime.GOARCH, runtime.Version()),
		"protocol_version":           "v11",
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
			case client.QiDian:
				return 4
			default:
				return -1
			}
		}(),
	})
}

// CQGetModelShow 获取在线机型
//
// https://club.vip.qq.com/onlinestatus/set
func (bot *CQBot) CQGetModelShow(modelName string) MSG {
	variants, err := bot.Client.GetModelShow(modelName)
	if err != nil {
		return Failed(100, "GET_MODEL_SHOW_API_ERROR", "无法获取在线机型")
	}
	a := make([]MSG, 0, len(variants))
	for _, v := range variants {
		a = append(a, MSG{
			"model_show": v.ModelShow,
			"need_pay":   v.NeedPay,
		})
	}
	return OK(MSG{
		"variants": a,
	})
}

// CQSetModelShow 设置在线机型
//
// https://club.vip.qq.com/onlinestatus/set
func (bot *CQBot) CQSetModelShow(modelName string, modelShow string) MSG {
	err := bot.Client.SetModelShow(modelName, modelShow)
	if err != nil {
		return Failed(100, "SET_MODEL_SHOW_API_ERROR", "无法设置在线机型")
	}
	return OK(nil)
}

// OK 生成成功返回值
func OK(data interface{}) MSG {
	return MSG{"data": data, "retcode": 0, "status": "ok"}
}

// Failed 生成失败返回值
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

func convertGroupMemberInfo(groupID int64, m *client.GroupMemberInfo) MSG {
	return MSG{
		"group_id": groupID,
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
			case client.Member:
				return "member"
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
	if utf8.RuneCountInString(str) <= 10 {
		return str
	}
	b := utils.S2B(str)
	limited := make([]rune, 0, 10)
	for i := 0; i < 10; i++ {
		decodeRune, size := utf8.DecodeRune(b)
		b = b[size:]
		limited = append(limited, decodeRune)
	}
	return string(limited) + " ..."
}
