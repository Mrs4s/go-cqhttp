package coolq

import (
	"strconv"

	"github.com/Mrs4s/MiraiGo/topic"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
)

func convertGroupMemberInfo(groupID int64, m *client.GroupMemberInfo) global.MSG {
	sex := "unknown"
	if m.Gender == 1 { // unknown = 0xff
		sex = "female"
	} else if m.Gender == 0 {
		sex = "male"
	}
	role := "member"
	switch m.Permission { // nolint:exhaustive
	case client.Owner:
		role = "owner"
	case client.Administrator:
		role = "admin"
	}
	return global.MSG{
		"group_id":          groupID,
		"user_id":           m.Uin,
		"nickname":          m.Nickname,
		"card":              m.CardName,
		"sex":               sex,
		"age":               0,
		"area":              "",
		"join_time":         m.JoinTime,
		"last_sent_time":    m.LastSpeakTime,
		"shut_up_timestamp": m.ShutUpTimestamp,
		"level":             strconv.FormatInt(int64(m.Level), 10),
		"role":              role,
		"unfriendly":        false,
		"title":             m.SpecialTitle,
		"title_expire_time": m.SpecialTitleExpireTime,
		"card_changeable":   false,
	}
}

func convertGuildMemberInfo(m []*client.GuildMemberInfo) (r []global.MSG) {
	for _, mem := range m {
		r = append(r, global.MSG{
			"tiny_id":   fU64(mem.TinyId),
			"title":     mem.Title,
			"nickname":  mem.Nickname,
			"role_id":   fU64(mem.Role),
			"role_name": mem.RoleName,
		})
	}
	return
}

func (bot *CQBot) formatGroupMessage(m *message.GroupMessage) global.MSG {
	source := message.Source{
		SourceType: message.SourceGroup,
		PrimaryID:  m.GroupCode,
	}
	cqm := ToStringMessage(m.Elements, source, true)
	postType := "message"
	if m.Sender.Uin == bot.Client.Uin {
		postType = "message_sent"
	}
	gm := global.MSG{
		"anonymous":    nil,
		"font":         0,
		"group_id":     m.GroupCode,
		"message":      ToFormattedMessage(m.Elements, source, false),
		"message_type": "group",
		"message_seq":  m.Id,
		"post_type":    postType,
		"raw_message":  cqm,
		"self_id":      bot.Client.Uin,
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
		role := "member"
		switch mem.Permission { // nolint:exhaustive
		case client.Owner:
			role = "owner"
		case client.Administrator:
			role = "admin"
		}
		ms["role"] = role
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
		"channel_id":        fU64(c.ChannelId),
		"channel_type":      c.ChannelType,
		"channel_name":      c.ChannelName,
		"owner_guild_id":    fU64(c.Meta.GuildId),
		"creator_tiny_id":   fU64(c.Meta.CreatorTinyId),
		"create_time":       c.Meta.CreateTime,
		"current_slow_mode": c.Meta.CurrentSlowMode,
		"talk_permission":   c.Meta.TalkPermission,
		"visible_type":      c.Meta.VisibleType,
		"slow_modes":        slowModes,
	}
}

func convertChannelFeedInfo(f *topic.Feed) global.MSG {
	m := global.MSG{
		"id":          f.Id,
		"title":       f.Title,
		"sub_title":   f.SubTitle,
		"create_time": f.CreateTime,
		"guild_id":    fU64(f.GuildId),
		"channel_id":  fU64(f.ChannelId),
		"poster_info": global.MSG{
			"tiny_id":  f.Poster.TinyIdStr,
			"nickname": f.Poster.Nickname,
			"icon_url": f.Poster.IconUrl,
		},
		"contents": FeedContentsToArrayMessage(f.Contents),
	}
	images := make([]global.MSG, 0, len(f.Images))
	videos := make([]global.MSG, 0, len(f.Videos))
	for _, image := range f.Images {
		images = append(images, global.MSG{
			"file_id":    image.FileId,
			"pattern_id": image.PatternId,
			"url":        image.Url,
			"width":      image.Width,
			"height":     image.Height,
		})
	}
	for _, video := range f.Videos {
		videos = append(videos, global.MSG{
			"file_id":    video.FileId,
			"pattern_id": video.PatternId,
			"url":        video.Url,
			"width":      video.Width,
			"height":     video.Height,
		})
	}
	m["resource"] = global.MSG{
		"images": images,
		"videos": videos,
	}
	return m
}

func convertReactions(reactions []*message.GuildMessageEmojiReaction) (r []global.MSG) {
	r = make([]global.MSG, len(reactions))
	for i, re := range reactions {
		r[i] = global.MSG{
			"emoji_id":    re.EmojiId,
			"emoji_index": re.Face.Index,
			"emoji_type":  re.EmojiType,
			"emoji_name":  re.Face.Name,
			"count":       re.Count,
			"clicked":     re.Clicked,
		}
	}
	return
}

func fU64(v uint64) string {
	return strconv.FormatUint(v, 10)
}
