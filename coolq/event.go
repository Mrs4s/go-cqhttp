package coolq

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/cache"
	"github.com/Mrs4s/go-cqhttp/internal/download"
)

// ToFormattedMessage 将给定[]message.IMessageElement转换为通过coolq.SetMessageFormat所定义的消息上报格式
func ToFormattedMessage(e []message.IMessageElement, source message.Source) (r any) {
	if base.PostFormat == "string" {
		r = toStringMessage(e, source)
	} else if base.PostFormat == "array" {
		r = toElements(e, source)
	}
	return
}

type event struct {
	PostType   string
	DetailType string
	SubType    string
	Time       int64
	SelfID     int64
	Others     global.MSG
}

func (ev *event) MarshalJSON() ([]byte, error) {
	buf := global.NewBuffer()
	defer global.PutBuffer(buf)

	detail := ""
	switch ev.PostType {
	case "message", "message_sent":
		detail = "message_type"
	case "notice":
		detail = "notice_type"
	case "request":
		detail = "request_type"
	case "meta_event":
		detail = "meta_event_type"
	default:
		panic("unknown post type: " + ev.PostType)
	}
	fmt.Fprintf(buf, `{"post_type":"%s","%s":"%s","time":%d,"self_id":%d`, ev.PostType, detail, ev.DetailType, ev.Time, ev.SelfID)
	if ev.SubType != "" {
		fmt.Fprintf(buf, `,"sub_type":"%s"`, ev.SubType)
	}
	for k, v := range ev.Others {
		v, err := json.Marshal(v)
		if err != nil {
			log.Warnf("marshal message payload error: %v", err)
			return nil, err
		}
		fmt.Fprintf(buf, `,"%s":%s`, k, v)
	}
	buf.WriteByte('}')
	return append([]byte(nil), buf.Bytes()...), nil
}

func (bot *CQBot) privateMessageEvent(_ *client.QQClient, m *message.PrivateMessage) {
	bot.checkMedia(m.Elements, m.Sender.Uin)
	source := message.Source{
		SourceType: message.SourcePrivate,
		PrimaryID:  m.Sender.Uin,
	}
	cqm := toStringMessage(m.Elements, source)
	id := bot.InsertPrivateMessage(m)
	log.Infof("收到好友 %v(%v) 的消息: %v (%v)", m.Sender.DisplayName(), m.Sender.Uin, cqm, id)
	typ := "message/private/friend"
	if m.Sender.Uin == bot.Client.Uin {
		typ = "message_sent/private/friend"
	}
	fm := global.MSG{
		"message_id":  id,
		"user_id":     m.Sender.Uin,
		"target_id":   m.Target,
		"message":     ToFormattedMessage(m.Elements, source),
		"raw_message": cqm,
		"font":        0,
		"sender": global.MSG{
			"user_id":  m.Sender.Uin,
			"nickname": m.Sender.Nickname,
			"sex":      "unknown",
			"age":      0,
		},
	}
	bot.dispatchEvent(typ, fm)
}

func (bot *CQBot) groupMessageEvent(c *client.QQClient, m *message.GroupMessage) {
	bot.checkMedia(m.Elements, m.GroupCode)
	for _, elem := range m.Elements {
		if file, ok := elem.(*message.GroupFileElement); ok {
			log.Infof("群 %v(%v) 内 %v(%v) 上传了文件: %v", m.GroupName, m.GroupCode, m.Sender.DisplayName(), m.Sender.Uin, file.Name)
			bot.dispatchEvent("notice/group_upload", global.MSG{
				"group_id": m.GroupCode,
				"user_id":  m.Sender.Uin,
				"file": global.MSG{
					"id":    file.Path,
					"name":  file.Name,
					"size":  file.Size,
					"busid": file.Busid,
					"url":   c.GetGroupFileUrl(m.GroupCode, file.Path, file.Busid),
				},
			})
			return
		}
	}
	source := message.Source{
		SourceType: message.SourceGroup,
		PrimaryID:  m.GroupCode,
	}
	cqm := toStringMessage(m.Elements, source)
	id := bot.InsertGroupMessage(m)
	log.Infof("收到群 %v(%v) 内 %v(%v) 的消息: %v (%v)", m.GroupName, m.GroupCode, m.Sender.DisplayName(), m.Sender.Uin, cqm, id)
	gm := bot.formatGroupMessage(m)
	if gm == nil {
		return
	}
	gm.Others["message_id"] = id
	bot.dispatch(gm)
}

func (bot *CQBot) tempMessageEvent(_ *client.QQClient, e *client.TempMessageEvent) {
	m := e.Message
	bot.checkMedia(m.Elements, m.Sender.Uin)
	source := message.Source{
		SourceType: message.SourcePrivate,
		PrimaryID:  e.Session.Sender,
	}
	cqm := toStringMessage(m.Elements, source)
	if base.AllowTempSession {
		bot.tempSessionCache.Store(m.Sender.Uin, e.Session)
	}

	id := m.Id
	// todo(Mrs4s)
	// if bot.db != nil { // nolint
	// 		id = bot.InsertTempMessage(m.Sender.Uin, m)
	// }
	log.Infof("收到来自群 %v(%v) 内 %v(%v) 的临时会话消息: %v", m.GroupName, m.GroupCode, m.Sender.DisplayName(), m.Sender.Uin, cqm)
	tm := global.MSG{
		"temp_source": e.Session.Source,
		"message_id":  id,
		"user_id":     m.Sender.Uin,
		"message":     ToFormattedMessage(m.Elements, source),
		"raw_message": cqm,
		"font":        0,
		"sender": global.MSG{
			"user_id":  m.Sender.Uin,
			"group_id": m.GroupCode,
			"nickname": m.Sender.Nickname,
			"sex":      "unknown",
			"age":      0,
		},
	}
	bot.dispatchEvent("message/private/group", tm)
}

func (bot *CQBot) guildChannelMessageEvent(c *client.QQClient, m *message.GuildChannelMessage) {
	bot.checkMedia(m.Elements, int64(m.Sender.TinyId))
	guild := c.GuildService.FindGuild(m.GuildId)
	if guild == nil {
		return
	}
	channel := guild.FindChannel(m.ChannelId)
	source := message.Source{
		SourceType:  message.SourceGuildChannel,
		PrimaryID:   int64(m.GuildId),
		SecondaryID: int64(m.ChannelId),
	}
	log.Infof("收到来自频道 %v(%v) 子频道 %v(%v) 内 %v(%v) 的消息: %v", guild.GuildName, guild.GuildId, channel.ChannelName, m.ChannelId, m.Sender.Nickname, m.Sender.TinyId, toStringMessage(m.Elements, source))
	id := bot.InsertGuildChannelMessage(m)
	ev := bot.event("message/guild/channel", global.MSG{
		"guild_id":     fU64(m.GuildId),
		"channel_id":   fU64(m.ChannelId),
		"message_id":   id,
		"user_id":      fU64(m.Sender.TinyId),
		"message":      ToFormattedMessage(m.Elements, source), // todo: 增加对频道消息 Reply 的支持
		"self_tiny_id": fU64(bot.Client.GuildService.TinyId),
		"sender": global.MSG{
			"user_id":  m.Sender.TinyId,
			"tiny_id":  fU64(m.Sender.TinyId),
			"nickname": m.Sender.Nickname,
		},
	})
	ev.Time = m.Time
	bot.dispatch(ev)
}

func (bot *CQBot) guildMessageReactionsUpdatedEvent(c *client.QQClient, e *client.GuildMessageReactionsUpdatedEvent) {
	guild := c.GuildService.FindGuild(e.GuildId)
	if guild == nil {
		return
	}
	msgID := encodeGuildMessageID(e.GuildId, e.ChannelId, e.MessageId, message.SourceGuildChannel)
	str := fmt.Sprintf("频道 %v(%v) 消息 %v 表情贴片已更新: ", guild.GuildName, guild.GuildId, msgID)
	currentReactions := make([]global.MSG, len(e.CurrentReactions))
	for i, r := range e.CurrentReactions {
		str += fmt.Sprintf("%v*%v ", r.Face.Name, r.Count)
		currentReactions[i] = global.MSG{
			"emoji_id":    r.EmojiId,
			"emoji_index": r.Face.Index,
			"emoji_type":  r.EmojiType,
			"emoji_name":  r.Face.Name,
			"count":       r.Count,
			"clicked":     r.Clicked,
		}
	}
	if len(e.CurrentReactions) == 0 {
		str += "无任何表情"
	}
	log.Infof(str)
	bot.dispatchEvent("notice/message_reactions_updated", global.MSG{
		"guild_id":          fU64(e.GuildId),
		"channel_id":        fU64(e.ChannelId),
		"message_id":        msgID,
		"operator_id":       fU64(e.OperatorId),
		"current_reactions": currentReactions,
		"self_tiny_id":      fU64(bot.Client.GuildService.TinyId),
		"user_id":           e.OperatorId,
	})
}

func (bot *CQBot) guildChannelMessageRecalledEvent(c *client.QQClient, e *client.GuildMessageRecalledEvent) {
	guild := c.GuildService.FindGuild(e.GuildId)
	if guild == nil {
		return
	}
	channel := guild.FindChannel(e.ChannelId)
	if channel == nil {
		return
	}
	operator, err := c.GuildService.FetchGuildMemberProfileInfo(e.GuildId, e.OperatorId)
	if err != nil {
		log.Errorf("处理频道撤回事件时出现错误: 获取操作者资料时出现错误 %v", err)
		return
	}
	msgID := encodeGuildMessageID(e.GuildId, e.ChannelId, e.MessageId, message.SourceGuildChannel)
	log.Infof("用户 %v(%v) 撤回了频道 %v(%v) 子频道 %v(%v) 的消息 %v", operator.Nickname, operator.TinyId, guild.GuildName, guild.GuildId, channel.ChannelName, channel.ChannelId, msgID)
	bot.dispatchEvent("notice/guild_channel_recall", global.MSG{
		"guild_id":     fU64(e.GuildId),
		"channel_id":   fU64(e.ChannelId),
		"operator_id":  fU64(e.OperatorId),
		"message_id":   msgID,
		"self_tiny_id": fU64(bot.Client.GuildService.TinyId),
		"user_id":      e.OperatorId,
	})
}

func (bot *CQBot) guildChannelUpdatedEvent(c *client.QQClient, e *client.GuildChannelUpdatedEvent) {
	guild := c.GuildService.FindGuild(e.GuildId)
	if guild == nil {
		return
	}
	log.Infof("频道 %v(%v) 子频道 %v(%v) 信息已更新", guild.GuildName, guild.GuildId, e.NewChannelInfo.ChannelName, e.NewChannelInfo.ChannelId)
	bot.dispatchEvent("notice/channel_updated", global.MSG{
		"guild_id":     fU64(e.GuildId),
		"channel_id":   fU64(e.ChannelId),
		"operator_id":  fU64(e.OperatorId),
		"self_tiny_id": fU64(bot.Client.GuildService.TinyId),
		"user_id":      e.OperatorId,
		"old_info":     convertChannelInfo(e.OldChannelInfo),
		"new_info":     convertChannelInfo(e.NewChannelInfo),
	})
}

func (bot *CQBot) guildChannelCreatedEvent(c *client.QQClient, e *client.GuildChannelOperationEvent) {
	guild := c.GuildService.FindGuild(e.GuildId)
	if guild == nil {
		return
	}
	member, _ := c.GuildService.FetchGuildMemberProfileInfo(e.GuildId, e.OperatorId)
	if member == nil {
		member = &client.GuildUserProfile{Nickname: "未知"}
	}
	log.Infof("频道 %v(%v) 内用户 %v(%v) 创建了子频道 %v(%v)", guild.GuildName, guild.GuildId, member.Nickname, member.TinyId, e.ChannelInfo.ChannelName, e.ChannelInfo.ChannelId)
	bot.dispatchEvent("notice/channel_created", global.MSG{
		"guild_id":     fU64(e.GuildId),
		"channel_id":   fU64(e.ChannelInfo.ChannelId),
		"operator_id":  fU64(e.OperatorId),
		"self_tiny_id": fU64(bot.Client.GuildService.TinyId),
		"user_id":      e.OperatorId,
		"channel_info": convertChannelInfo(e.ChannelInfo),
	})
}

func (bot *CQBot) guildChannelDestroyedEvent(c *client.QQClient, e *client.GuildChannelOperationEvent) {
	guild := c.GuildService.FindGuild(e.GuildId)
	if guild == nil {
		return
	}
	member, _ := c.GuildService.FetchGuildMemberProfileInfo(e.GuildId, e.OperatorId)
	if member == nil {
		member = &client.GuildUserProfile{Nickname: "未知"}
	}
	log.Infof("频道 %v(%v) 内用户 %v(%v) 删除了子频道 %v(%v)", guild.GuildName, guild.GuildId, member.Nickname, member.TinyId, e.ChannelInfo.ChannelName, e.ChannelInfo.ChannelId)
	bot.dispatchEvent("notice/channel_destroyed", global.MSG{
		"guild_id":     fU64(e.GuildId),
		"channel_id":   fU64(e.ChannelInfo.ChannelId),
		"operator_id":  fU64(e.OperatorId),
		"self_tiny_id": fU64(bot.Client.GuildService.TinyId),
		"user_id":      e.OperatorId,
		"channel_info": convertChannelInfo(e.ChannelInfo),
	})
}

func (bot *CQBot) groupMutedEvent(c *client.QQClient, e *client.GroupMuteEvent) {
	g := c.FindGroup(e.GroupCode)
	if e.TargetUin == 0 {
		if e.Time != 0 {
			log.Infof("群 %v 被 %v 开启全员禁言.",
				formatGroupName(g), formatMemberName(g.FindMember(e.OperatorUin)))
		} else {
			log.Infof("群 %v 被 %v 解除全员禁言.",
				formatGroupName(g), formatMemberName(g.FindMember(e.OperatorUin)))
		}
	} else {
		if e.Time > 0 {
			log.Infof("群 %v 内 %v 被 %v 禁言了 %v 秒.",
				formatGroupName(g), formatMemberName(g.FindMember(e.TargetUin)), formatMemberName(g.FindMember(e.OperatorUin)), e.Time)
		} else {
			log.Infof("群 %v 内 %v 被 %v 解除禁言.",
				formatGroupName(g), formatMemberName(g.FindMember(e.TargetUin)), formatMemberName(g.FindMember(e.OperatorUin)))
		}
	}
	typ := "notice/group_ban/ban"
	if e.Time == 0 {
		typ = "notice/group_ban/lift_ban"
	}
	bot.dispatchEvent(typ, global.MSG{
		"duration":    e.Time,
		"group_id":    e.GroupCode,
		"operator_id": e.OperatorUin,
		"user_id":     e.TargetUin,
	})
}

func (bot *CQBot) groupRecallEvent(c *client.QQClient, e *client.GroupMessageRecalledEvent) {
	g := c.FindGroup(e.GroupCode)
	gid := db.ToGlobalID(e.GroupCode, e.MessageId)
	log.Infof("群 %v 内 %v 撤回了 %v 的消息: %v.",
		formatGroupName(g), formatMemberName(g.FindMember(e.OperatorUin)), formatMemberName(g.FindMember(e.AuthorUin)), gid)

	ev := bot.event("notice/group_recall", global.MSG{
		"group_id":    e.GroupCode,
		"user_id":     e.AuthorUin,
		"operator_id": e.OperatorUin,
		"message_id":  gid,
	})
	ev.Time = int64(e.Time)
	bot.dispatch(ev)
}

func (bot *CQBot) groupNotifyEvent(c *client.QQClient, e client.INotifyEvent) {
	group := c.FindGroup(e.From())
	switch notify := e.(type) {
	case *client.GroupPokeNotifyEvent:
		sender := group.FindMember(notify.Sender)
		receiver := group.FindMember(notify.Receiver)
		log.Infof("群 %v 内 %v 戳了戳 %v", formatGroupName(group), formatMemberName(sender), formatMemberName(receiver))
		bot.dispatchEvent("notice/notify/poke", global.MSG{
			"group_id":  group.Code,
			"user_id":   notify.Sender,
			"sender_id": notify.Sender,
			"target_id": notify.Receiver,
		})
	case *client.GroupRedBagLuckyKingNotifyEvent:
		sender := group.FindMember(notify.Sender)
		luckyKing := group.FindMember(notify.LuckyKing)
		log.Infof("群 %v 内 %v 的红包被抢完, %v 是运气王", formatGroupName(group), formatMemberName(sender), formatMemberName(luckyKing))
		bot.dispatchEvent("notice/notify/lucky_king", global.MSG{
			"group_id":  group.Code,
			"user_id":   notify.Sender,
			"sender_id": notify.Sender,
			"target_id": notify.LuckyKing,
		})
	case *client.MemberHonorChangedNotifyEvent:
		log.Info(notify.Content())
		bot.dispatchEvent("notice/notify/honor", global.MSG{
			"group_id": group.Code,
			"user_id":  notify.Uin,
			"honor_type": func() string {
				switch notify.Honor {
				case client.Talkative:
					return "talkative"
				case client.Performer:
					return "performer"
				case client.Emotion:
					return "emotion"
				case client.Legend:
					return "legend"
				case client.StrongNewbie:
					return "strong_newbie"
				default:
					return "ERROR"
				}
			}(),
		})
	}
}

func (bot *CQBot) friendNotifyEvent(c *client.QQClient, e client.INotifyEvent) {
	friend := c.FindFriend(e.From())
	if notify, ok := e.(*client.FriendPokeNotifyEvent); ok {
		if notify.Receiver == notify.Sender {
			log.Infof("好友 %v 戳了戳自己.", friend.Nickname)
		} else {
			log.Infof("好友 %v 戳了戳你.", friend.Nickname)
		}
		bot.dispatchEvent("notice/notify/poke", global.MSG{
			"user_id":   notify.Sender,
			"sender_id": notify.Sender,
			"target_id": notify.Receiver,
		})
	}
}

func (bot *CQBot) memberTitleUpdatedEvent(c *client.QQClient, e *client.MemberSpecialTitleUpdatedEvent) {
	group := c.FindGroup(e.GroupCode)
	mem := group.FindMember(e.Uin)
	log.Infof("群 %v(%v) 内成员 %v(%v) 获得了新的头衔: %v", group.Name, group.Code, mem.DisplayName(), mem.Uin, e.NewTitle)
	bot.dispatchEvent("notice/notify/title", global.MSG{
		"group_id": group.Code,
		"user_id":  e.Uin,
		"title":    e.NewTitle,
	})
}

func (bot *CQBot) friendRecallEvent(c *client.QQClient, e *client.FriendMessageRecalledEvent) {
	f := c.FindFriend(e.FriendUin)
	gid := db.ToGlobalID(e.FriendUin, e.MessageId)
	if f != nil {
		log.Infof("好友 %v(%v) 撤回了消息: %v", f.Nickname, f.Uin, gid)
	} else {
		log.Infof("好友 %v 撤回了消息: %v", e.FriendUin, gid)
	}
	ev := bot.event("notice/friend_recall", global.MSG{
		"user_id":    e.FriendUin,
		"message_id": gid,
	})
	ev.Time = e.Time
	bot.dispatch(ev)
}

func (bot *CQBot) offlineFileEvent(c *client.QQClient, e *client.OfflineFileEvent) {
	f := c.FindFriend(e.Sender)
	if f == nil {
		return
	}
	log.Infof("好友 %v(%v) 发送了离线文件 %v", f.Nickname, f.Uin, e.FileName)
	bot.dispatchEvent("notice/offline_file", global.MSG{
		"user_id": e.Sender,
		"file": global.MSG{
			"name": e.FileName,
			"size": e.FileSize,
			"url":  e.DownloadUrl,
		},
	})
}

func (bot *CQBot) joinGroupEvent(c *client.QQClient, group *client.GroupInfo) {
	if group == nil {
		return
	}
	log.Infof("Bot进入了群 %v.", formatGroupName(group))
	bot.dispatch(bot.groupIncrease(group.Code, 0, c.Uin))
}

func (bot *CQBot) leaveGroupEvent(c *client.QQClient, e *client.GroupLeaveEvent) {
	if e.Operator != nil {
		log.Infof("Bot被 %v T出了群 %v.", formatMemberName(e.Operator), formatGroupName(e.Group))
	} else {
		log.Infof("Bot退出了群 %v.", formatGroupName(e.Group))
	}
	bot.dispatch(bot.groupDecrease(e.Group.Code, c.Uin, e.Operator))
}

func (bot *CQBot) memberPermissionChangedEvent(_ *client.QQClient, e *client.MemberPermissionChangedEvent) {
	st := "unset"
	if e.NewPermission == client.Administrator {
		st = "set"
	}
	bot.dispatchEvent("notice/group_admin/"+st, global.MSG{
		"group_id": e.Group.Code,
		"user_id":  e.Member.Uin,
	})
}

func (bot *CQBot) memberCardUpdatedEvent(_ *client.QQClient, e *client.MemberCardUpdatedEvent) {
	log.Infof("群 %v 的 %v 更新了名片 %v -> %v", formatGroupName(e.Group), formatMemberName(e.Member), e.OldCard, e.Member.CardName)
	bot.dispatchEvent("notice/group_card", global.MSG{
		"group_id": e.Group.Code,
		"user_id":  e.Member.Uin,
		"card_new": e.Member.CardName,
		"card_old": e.OldCard,
	})
}

func (bot *CQBot) memberJoinEvent(_ *client.QQClient, e *client.MemberJoinGroupEvent) {
	log.Infof("新成员 %v 进入了群 %v.", formatMemberName(e.Member), formatGroupName(e.Group))
	bot.dispatch(bot.groupIncrease(e.Group.Code, 0, e.Member.Uin))
}

func (bot *CQBot) memberLeaveEvent(_ *client.QQClient, e *client.MemberLeaveGroupEvent) {
	if e.Operator != nil {
		log.Infof("成员 %v 被 %v T出了群 %v.", formatMemberName(e.Member), formatMemberName(e.Operator), formatGroupName(e.Group))
	} else {
		log.Infof("成员 %v 离开了群 %v.", formatMemberName(e.Member), formatGroupName(e.Group))
	}
	bot.dispatch(bot.groupDecrease(e.Group.Code, e.Member.Uin, e.Operator))
}

func (bot *CQBot) friendRequestEvent(_ *client.QQClient, e *client.NewFriendRequest) {
	log.Infof("收到来自 %v(%v) 的好友请求: %v", e.RequesterNick, e.RequesterUin, e.Message)
	flag := strconv.FormatInt(e.RequestId, 10)
	bot.friendReqCache.Store(flag, e)
	bot.dispatchEvent("request/friend", global.MSG{
		"user_id": e.RequesterUin,
		"comment": e.Message,
		"flag":    flag,
	})
}

func (bot *CQBot) friendAddedEvent(_ *client.QQClient, e *client.NewFriendEvent) {
	log.Infof("添加了新好友: %v(%v)", e.Friend.Nickname, e.Friend.Uin)
	bot.tempSessionCache.Delete(e.Friend.Uin)
	bot.dispatchEvent("notice/friend_add", global.MSG{
		"user_id": e.Friend.Uin,
	})
}

func (bot *CQBot) groupInvitedEvent(_ *client.QQClient, e *client.GroupInvitedRequest) {
	log.Infof("收到来自群 %v(%v) 内用户 %v(%v) 的加群邀请.", e.GroupName, e.GroupCode, e.InvitorNick, e.InvitorUin)
	flag := strconv.FormatInt(e.RequestId, 10)
	bot.dispatchEvent("request/group/invite", global.MSG{
		"group_id":   e.GroupCode,
		"user_id":    e.InvitorUin,
		"invitor_id": 0,
		"comment":    "",
		"flag":       flag,
	})
}

func (bot *CQBot) groupJoinReqEvent(_ *client.QQClient, e *client.UserJoinGroupRequest) {
	log.Infof("群 %v(%v) 收到来自用户 %v(%v) 的加群请求.", e.GroupName, e.GroupCode, e.RequesterNick, e.RequesterUin)
	flag := strconv.FormatInt(e.RequestId, 10)
	bot.dispatchEvent("request/group/add", global.MSG{
		"group_id":   e.GroupCode,
		"user_id":    e.RequesterUin,
		"invitor_id": e.ActionUin,
		"comment":    e.Message,
		"flag":       flag,
	})
}

func (bot *CQBot) otherClientStatusChangedEvent(_ *client.QQClient, e *client.OtherClientStatusChangedEvent) {
	if e.Online {
		log.Infof("Bot 账号在客户端 %v (%v) 登录.", e.Client.DeviceName, e.Client.DeviceKind)
	} else {
		log.Infof("Bot 账号在客户端 %v (%v) 登出.", e.Client.DeviceName, e.Client.DeviceKind)
	}
	bot.dispatchEvent("notice/client_status", global.MSG{
		"online": e.Online,
		"client": global.MSG{
			"app_id":      e.Client.AppId,
			"device_name": e.Client.DeviceName,
			"device_kind": e.Client.DeviceKind,
		},
	})
}

func (bot *CQBot) groupEssenceMsg(c *client.QQClient, e *client.GroupDigestEvent) {
	g := c.FindGroup(e.GroupCode)
	gid := db.ToGlobalID(e.GroupCode, e.MessageID)
	if e.OperationType == 1 {
		log.Infof(
			"群 %v 内 %v 将 %v 的消息(%v)设为了精华消息.",
			formatGroupName(g),
			formatMemberName(g.FindMember(e.OperatorUin)),
			formatMemberName(g.FindMember(e.SenderUin)),
			gid,
		)
	} else {
		log.Infof(
			"群 %v 内 %v 将 %v 的消息(%v)移出了精华消息.",
			formatGroupName(g),
			formatMemberName(g.FindMember(e.OperatorUin)),
			formatMemberName(g.FindMember(e.SenderUin)),
			gid,
		)
	}
	if e.OperatorUin == bot.Client.Uin {
		return
	}
	subtype := "delete"
	if e.OperationType == 1 {
		subtype = "add"
	}
	bot.dispatchEvent("notice/essence/"+subtype, global.MSG{
		"group_id":    e.GroupCode,
		"sender_id":   e.SenderUin,
		"operator_id": e.OperatorUin,
		"message_id":  gid,
	})
}

func (bot *CQBot) groupIncrease(groupCode, operatorUin, userUin int64) *event {
	return bot.event("notice/group_increase/approve", global.MSG{
		"group_id":    groupCode,
		"operator_id": operatorUin,
		"user_id":     userUin,
	})
}

func (bot *CQBot) groupDecrease(groupCode, userUin int64, operator *client.GroupMemberInfo) *event {
	op := userUin
	if operator != nil {
		op = operator.Uin
	}
	subtype := "leave"
	if operator != nil {
		if userUin == bot.Client.Uin {
			subtype = "kick_me"
		} else {
			subtype = "kick"
		}
	}
	return bot.event("notice/group_decrease/"+subtype, global.MSG{
		"group_id":    groupCode,
		"operator_id": op,
		"user_id":     userUin,
	})
}

func (bot *CQBot) checkMedia(e []message.IMessageElement, sourceID int64) {
	for _, elem := range e {
		switch i := elem.(type) {
		case *message.GroupImageElement:
			if i.Flash && sourceID != 0 {
				u, err := bot.Client.GetGroupImageDownloadUrl(i.FileId, sourceID, i.Md5)
				if err != nil {
					log.Warnf("获取闪照地址时出现错误: %v", err)
				} else {
					i.Url = u
				}
			}
			data := binary.NewWriterF(func(w *binary.Writer) {
				w.Write(i.Md5)
				w.WriteUInt32(uint32(i.Size))
				w.WriteString(i.ImageId)
				w.WriteString(i.Url)
			})
			cache.Image.Insert(i.Md5, data)

		case *message.GuildImageElement:
			data := binary.NewWriterF(func(w *binary.Writer) {
				w.Write(i.Md5)
				w.WriteUInt32(uint32(i.Size))
				w.WriteString(i.DownloadIndex)
				w.WriteString(i.Url)
			})
			filename := hex.EncodeToString(i.Md5) + ".image"
			cache.Image.Insert(i.Md5, data)
			if i.Url != "" && !global.PathExists(path.Join(global.ImagePath, "guild-images", filename)) {
				r := download.Request{URL: i.Url}
				if err := r.WriteToFile(path.Join(global.ImagePath, "guild-images", filename)); err != nil {
					log.Warnf("下载频道图片时出现错误: %v", err)
				}
			}
		case *message.FriendImageElement:
			data := binary.NewWriterF(func(w *binary.Writer) {
				w.Write(i.Md5)
				w.WriteUInt32(uint32(i.Size))
				w.WriteString(i.ImageId)
				w.WriteString(i.Url)
			})
			cache.Image.Insert(i.Md5, data)

		case *message.VoiceElement:
			// todo: don't download original file?
			i.Name = strings.ReplaceAll(i.Name, "{", "")
			i.Name = strings.ReplaceAll(i.Name, "}", "")
			if !global.PathExists(path.Join(global.VoicePath, i.Name)) {
				err := download.Request{URL: i.Url}.WriteToFile(path.Join(global.VoicePath, i.Name))
				if err != nil {
					log.Warnf("语音文件 %v 下载失败: %v", i.Name, err)
					continue
				}
			}
		case *message.ShortVideoElement:
			data := binary.NewWriterF(func(w *binary.Writer) {
				w.Write(i.Md5)
				w.Write(i.ThumbMd5)
				w.WriteUInt32(uint32(i.Size))
				w.WriteUInt32(uint32(i.ThumbSize))
				w.WriteString(i.Name)
				w.Write(i.Uuid)
			})
			filename := hex.EncodeToString(i.Md5) + ".video"
			cache.Video.Insert(i.Md5, data)
			i.Name = filename
			i.Url = bot.Client.GetShortVideoUrl(i.Uuid, i.Md5)
		}
	}
}
