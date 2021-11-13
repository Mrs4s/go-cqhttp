package coolq

import (
	"strconv"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
)

func convertGroupMemberInfo(groupID int64, m *client.GroupMemberInfo) global.MSG {
	return global.MSG{
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
		"age":               0,
		"area":              "",
		"join_time":         m.JoinTime,
		"last_sent_time":    m.LastSpeakTime,
		"shut_up_timestamp": m.ShutUpTimestamp,
		"level":             strconv.FormatInt(int64(m.Level), 10),
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

func convertGuildMemberInfo(m *client.GuildMemberInfo) global.MSG {
	return global.MSG{
		"tiny_id":  m.TinyId,
		"title":    m.Title,
		"nickname": m.Nickname,
		"role":     m.Role,
	}
}

func (bot *CQBot) formatGroupMessage(m *message.GroupMessage) global.MSG {
	source := MessageSource{
		SourceType: MessageSourceGroup,
		PrimaryID:  uint64(m.GroupCode),
	}
	cqm := ToStringMessage(m.Elements, source, true)
	gm := global.MSG{
		"anonymous":    nil,
		"font":         0,
		"group_id":     m.GroupCode,
		"message":      ToFormattedMessage(m.Elements, source, false),
		"message_type": "group",
		"message_seq":  m.Id,
		"post_type": func() string {
			if m.Sender.Uin == bot.Client.Uin {
				return "message_sent"
			}
			return "message"
		}(),
		"raw_message": cqm,
		"self_id":     bot.Client.Uin,
		"sender": global.MSG{
			"age":     0,
			"area":    "",
			"level":   "",
			"sex":     "unknown",
			"user_id": m.Sender.Uin,
		},
		"sub_type": "normal",
		"time":     m.Time,
		"user_id":  m.Sender.Uin,
	}
	if m.Sender.IsAnonymous() {
		gm["anonymous"] = global.MSG{
			"flag": m.Sender.AnonymousInfo.AnonymousId + "|" + m.Sender.AnonymousInfo.AnonymousNick,
			"id":   m.Sender.Uin,
			"name": m.Sender.AnonymousInfo.AnonymousNick,
		}
		gm["sender"].(global.MSG)["nickname"] = "匿名消息"
		gm["sub_type"] = "anonymous"
	} else {
		group := bot.Client.FindGroup(m.GroupCode)
		mem := group.FindMember(m.Sender.Uin)
		if mem == nil {
			log.Warnf("获取 %v 成员信息失败，尝试刷新成员列表", m.Sender.Uin)
			t, err := bot.Client.GetGroupMembers(group)
			if err != nil {
				log.Warnf("刷新群 %v 成员列表失败: %v", group.Uin, err)
				return nil
			}
			group.Members = t
			mem = group.FindMember(m.Sender.Uin)
			if mem == nil {
				return nil
			}
		}
		ms := gm["sender"].(global.MSG)
		switch mem.Permission {
		case client.Owner:
			ms["role"] = "owner"
		case client.Administrator:
			ms["role"] = "admin"
		case client.Member:
			ms["role"] = "member"
		default:
			ms["role"] = "member"
		}
		ms["nickname"] = mem.Nickname
		ms["card"] = mem.CardName
		ms["title"] = mem.SpecialTitle
	}
	return gm
}

func convertChannelInfo(c *client.ChannelInfo) global.MSG {
	slowModes := make([]global.MSG, 0, len(c.Meta.SlowModes))
	for _, mode := range c.Meta.SlowModes {
		slowModes = append(slowModes, global.MSG{
			"slow_mode_key":    mode.SlowModeKey,
			"slow_mode_text":   mode.SlowModeText,
			"speak_frequency":  mode.SpeakFrequency,
			"slow_mode_circle": mode.SlowModeCircle,
		})
	}
	return global.MSG{
		"channel_id":        c.ChannelId,
		"channel_type":      c.ChannelType,
		"channel_name":      c.ChannelName,
		"owner_guild_id":    c.Meta.GuildId,
		"creator_id":        c.Meta.CreatorUin,
		"creator_tiny_id":   c.Meta.CreatorTinyId,
		"create_time":       c.Meta.CreateTime,
		"current_slow_mode": c.Meta.CurrentSlowMode,
		"talk_permission":   c.Meta.TalkPermission,
		"visible_type":      c.Meta.VisibleType,
		"slow_modes":        slowModes,
	}
}
