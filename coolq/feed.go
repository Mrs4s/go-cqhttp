package coolq

import (
	"github.com/Mrs4s/MiraiGo/topic"

	"github.com/Mrs4s/go-cqhttp/global"
)

// FeedContentsToArrayMessage 将话题频道帖子内容转换为 Array Message
func FeedContentsToArrayMessage(contents []topic.IFeedRichContentElement) []global.MSG {
	r := make([]global.MSG, 0, len(contents))
	for _, e := range contents {
		var m global.MSG
		switch elem := e.(type) {
		case *topic.TextElement:
			m = global.MSG{
				"type": "text",
				"data": global.MSG{"text": elem.Content},
			}
		case *topic.AtElement:
			m = global.MSG{
				"type": "at",
				"data": global.MSG{"id": elem.Id, "qq": elem.Id},
			}
		case *topic.EmojiElement:
			m = global.MSG{
				"type": "face",
				"data": global.MSG{"id": elem.Id},
			}
		case *topic.ChannelQuoteElement:
			m = global.MSG{
				"type": "channel_quote",
				"data": global.MSG{
					"guild_id":     fU64(elem.GuildId),
					"channel_id":   fU64(elem.ChannelId),
					"display_text": elem.DisplayText,
				},
			}
		case *topic.UrlQuoteElement:
			m = global.MSG{
				"type": "url_quote",
				"data": global.MSG{
					"url":          elem.Url,
					"display_text": elem.DisplayText,
				},
			}
		}
		if m != nil {
			r = append(r, m)
		}
	}
	return r
}
