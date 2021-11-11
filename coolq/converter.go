package coolq

import (
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/global"
	"strconv"
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
