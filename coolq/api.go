package coolq

import (
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/global"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
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
func (bot *CQBot) CQGetGroupMemberList(groupId int64, noCache bool) MSG {
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
		return Failed(100)
	}
	member := group.FindMember(userId)
	if member == nil {
		return Failed(102)
	}
	return OK(convertGroupMemberInfo(groupId, member))
}

// https://cqhttp.cc/docs/4.15/#/API?id=send_group_msg-%E5%8F%91%E9%80%81%E7%BE%A4%E6%B6%88%E6%81%AF
func (bot *CQBot) CQSendGroupMessage(groupId int64, i interface{}, autoEscape bool) MSG {
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
				return Failed(100)
			}
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
		return Failed(100)
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
		return Failed(100)
	}
	log.Infof("发送群 %v(%v)  的消息: %v (%v)", groupId, groupId, str, mid)
	return OK(MSG{"message_id": mid})
}

func (bot *CQBot) CQSendGroupForwardMessage(groupId int64, m gjson.Result) MSG {
	if m.Type != gjson.JSON {
		return Failed(100)
	}
	var nodes []*message.ForwardNode
	ts := time.Now().Add(-time.Minute * 5)
	hasCustom := func() bool {
		for _, item := range m.Array() {
			if item.Get("data.uin").Exists() {
				return true
			}
		}
		return false
	}()
	convert := func(e gjson.Result) {
		if e.Get("type").Str != "node" {
			return
		}
		ts.Add(time.Second)
		if e.Get("data.id").Exists() {
			i, _ := strconv.Atoi(e.Get("data.id").Str)
			m := bot.GetGroupMessage(int32(i))
			if m != nil {
				sender := m["sender"].(message.Sender)
				nodes = append(nodes, &message.ForwardNode{
					SenderId:   sender.Uin,
					SenderName: (&sender).DisplayName(),
					Time: func() int32 {
						if hasCustom {
							return int32(ts.Unix())
						}
						return m["time"].(int32)
					}(),
					Message: bot.ConvertStringMessage(m["message"].(string), true),
				})
				return
			}
			log.Warnf("警告: 引用消息 %v 错误或数据库未开启.", e.Get("data.id").Str)
			return
		}
		uin, _ := strconv.ParseInt(e.Get("data.uin").Str, 10, 64)
		name := e.Get("data.name").Str
		content := bot.ConvertObjectMessage(e.Get("data.content"), true)
		if uin != 0 && name != "" && len(content) > 0 {
			var newElem []message.IMessageElement
			for _, elem := range content {
				if img, ok := elem.(*message.ImageElement); ok {
					gm, err := bot.Client.UploadGroupImage(groupId, img.Data)
					if err != nil {
						log.Warnf("警告：群 %v 图片上传失败: %v", groupId, err)
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
				Time:       int32(ts.Unix()),
				Message:    newElem,
			})
			return
		}
		log.Warnf("警告: 非法 Forward node 将跳过")
	}
	if m.IsArray() {
		for _, item := range m.Array() {
			convert(item)
		}
	} else {
		convert(m)
	}
	if len(nodes) > 0 {
		gm := bot.Client.SendGroupForwardMessage(groupId, &message.ForwardMessage{Nodes: nodes})
		return OK(MSG{
			"message_id": ToGlobalId(groupId, gm.Id),
		})
	}
	return Failed(100)
}

// https://cqhttp.cc/docs/4.15/#/API?id=send_private_msg-%E5%8F%91%E9%80%81%E7%A7%81%E8%81%8A%E6%B6%88%E6%81%AF
func (bot *CQBot) CQSendPrivateMessage(userId int64, i interface{}, autoEscape bool) MSG {
	var str string
	if m, ok := i.(gjson.Result); ok {
		if m.Type == gjson.JSON {
			elem := bot.ConvertObjectMessage(m, true)
			mid := bot.SendPrivateMessage(userId, &message.SendingMessage{Elements: elem})
			if mid == -1 {
				return Failed(100)
			}
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
		return Failed(100)
	}
	var elem []message.IMessageElement
	if autoEscape {
		elem = append(elem, message.NewText(str))
	} else {
		elem = bot.ConvertStringMessage(str, false)
	}
	mid := bot.SendPrivateMessage(userId, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return Failed(100)
	}
	log.Infof("发送好友 %v(%v)  的消息: %v (%v)", userId, userId, str, mid)
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

func (bot *CQBot) CQSetGroupMemo(groupId int64, msg string) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.UpdateMemo(msg)
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

// https://cqhttp.cc/docs/4.15/#/API?id=set_group_leave-%E9%80%80%E5%87%BA%E7%BE%A4%E7%BB%84
func (bot *CQBot) CQSetGroupLeave(groupId int64) MSG {
	if g := bot.Client.FindGroup(groupId); g != nil {
		g.Quit()
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
func (bot *CQBot) CQProcessGroupRequest(flag, subType, reason string, approve bool) MSG {
	if subType == "add" {
		req, ok := bot.joinReqCache.Load(flag)
		if !ok {
			return Failed(100)
		}
		bot.joinReqCache.Delete(flag)
		if approve {
			req.(*client.UserJoinGroupRequest).Accept()
		} else {
			req.(*client.UserJoinGroupRequest).Reject(false, reason)
		}
		return OK(nil)
	}
	req, ok := bot.invitedReqCache.Load(flag)
	if ok {
		bot.invitedReqCache.Delete(flag)
		if approve {
			req.(*client.GroupInvitedRequest).Accept()
		} else {
			req.(*client.GroupInvitedRequest).Reject(false, reason)
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

// https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_admin-%E7%BE%A4%E7%BB%84%E8%AE%BE%E7%BD%AE%E7%AE%A1%E7%90%86%E5%91%98
func (bot *CQBot) CQSetGroupAdmin(groupId, userId int64, enable bool) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil || group.OwnerUin != bot.Client.Uin {
		return Failed(100)
	}
	mem := group.FindMember(userId)
	if mem == nil {
		return Failed(100)
	}
	mem.SetAdmin(enable)
	t, err := bot.Client.GetGroupMembers(group)
	if err != nil {
		log.Warnf("刷新群 %v 成员列表失败: %v", groupId, err)
		return Failed(100)
	}
	group.Members = t
	return OK(nil)
}

func (bot *CQBot) CQGetVipInfo(userId int64) MSG {
	msg := MSG{}
	vip, err := bot.Client.GetVipInfo(userId)
	if err != nil {
		return Failed(100)
	}
	msg = MSG{
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
		return Failed(100)
	}
	return OK(MSG{
		"user_id":  info.Uin,
		"nickname": info.Nickname,
		"sex": func() string {
			if info.Sex == 1 {
				return "female"
			}
			return "male"
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
			/*
				at := true
				if operation.Get("at_sender").Exists() {
					at = operation.Get("at_sender").Bool()
				}
			*/
			// TODO: 处理at字段
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
				bot.CQSetGroupKick(context.Get("group_id").Int(), context.Get("user_id").Int(), "")
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
	if !global.PathExists(path.Join(global.IMAGE_PATH, file)) {
		return Failed(100)
	}
	if b, err := ioutil.ReadFile(path.Join(global.IMAGE_PATH, file)); err == nil {
		r := binary.NewReader(b)
		r.ReadBytes(16)
		msg := MSG{
			"size":     r.ReadInt32(),
			"filename": r.ReadString(),
			"url":      r.ReadString(),
		}
		local := path.Join(global.CACHE_PATH, file+"."+path.Ext(msg["filename"].(string)))
		if !global.PathExists(local) {
			if data, err := global.GetBytes(msg["url"].(string)); err == nil {
				_ = ioutil.WriteFile(local, data, 0644)
			}
		}
		msg["file"] = local
		return OK(msg)
	}
	return Failed(100)
}

func (bot *CQBot) CQGetForwardMessage(resId string) MSG {
	m := bot.Client.GetForwardMessage(resId)
	if m == nil {
		return Failed(100)
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
	return OK(MSG{"yes": true})
}

func (bot *CQBot) CQReloadEventFilter() MSG {
	global.BootFilter()
	return OK(nil)
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
		"version":                    Version,
		"protocol": func() int {
			switch client.SystemDeviceInfo.Protocol {
			case client.AndroidPad:
				return 0
			case client.AndroidPhone:
				return 1
			case client.AndroidWatch:
				return 2
			default:
				return -1
			}
		}(),
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
