package coolq

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/Mrs4s/go-cqhttp/coolq/cqcode"
	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/cache"
	"github.com/Mrs4s/go-cqhttp/internal/param"
)

/*
var matchReg = regexp.MustCompile(`\[CQ:\w+?.*?]`)
var typeReg = regexp.MustCompile(`\[CQ:(\w+)`)
var paramReg = regexp.MustCompile(`,([\w\-.]+?)=([^,\]]+)`)
*/

// PokeElement 拍一拍
type PokeElement struct {
	Target int64
}

// LocalImageElement 本地图片
type LocalImageElement struct {
	Stream io.ReadSeeker
	File   string
	URL    string

	Flash    bool
	EffectID int32
}

// LocalVideoElement 本地视频
type LocalVideoElement struct {
	File  string
	thumb io.ReadSeeker
}

const (
	maxImageSize = 1024 * 1024 * 30  // 30MB
	maxVideoSize = 1024 * 1024 * 100 // 100MB
)

// Type implements the message.IMessageElement.
func (e *LocalImageElement) Type() message.ElementType {
	return message.Image
}

// Type impl message.IMessageElement
func (e *LocalVideoElement) Type() message.ElementType {
	return message.Video
}

// Type 获取元素类型ID
func (e *PokeElement) Type() message.ElementType {
	// Make message.IMessageElement Happy
	return message.At
}

func replyID(r *message.ReplyElement, source message.Source) int32 {
	id := source.PrimaryID
	seq := r.ReplySeq
	if source.SourceType == message.SourcePrivate {
		// 私聊似乎腾讯服务器有bug?
		seq = int32(uint16(seq))
		id = r.Sender
	}
	if r.GroupID != 0 {
		id = r.GroupID
	}
	return db.ToGlobalID(id, seq)
}

// ToArrayMessage 将消息元素数组转为MSG数组以用于消息上报
func ToArrayMessage(e []message.IMessageElement, source message.Source) (r []global.MSG) {
	r = make([]global.MSG, 0, len(e))
	m := &message.SendingMessage{Elements: e}
	reply := m.FirstOrNil(func(e message.IMessageElement) bool {
		_, ok := e.(*message.ReplyElement)
		return ok
	})
	if reply != nil && source.SourceType&(message.SourceGroup|message.SourcePrivate) != 0 {
		replyElem := reply.(*message.ReplyElement)
		id := replyID(replyElem, source)
		if base.ExtraReplyData {
			r = append(r, global.MSG{
				"type": "reply",
				"data": map[string]string{
					"id":   strconv.FormatInt(int64(id), 10),
					"seq":  strconv.FormatInt(int64(replyElem.ReplySeq), 10),
					"qq":   strconv.FormatInt(replyElem.Sender, 10),
					"time": strconv.FormatInt(int64(replyElem.Time), 10),
					"text": ToStringMessage(replyElem.Elements, source),
				},
			})
		} else {
			r = append(r, global.MSG{
				"type": "reply",
				"data": map[string]string{"id": strconv.FormatInt(int64(id), 10)},
			})
		}
	}
	for i, elem := range e {
		var m global.MSG
		switch o := elem.(type) {
		case *message.ReplyElement:
			if base.RemoveReplyAt && i+1 < len(e) {
				elem, ok := e[i+1].(*message.AtElement)
				if ok && elem.Target == o.Sender {
					e[i+1] = nil
				}
			}
			continue
		case *message.TextElement:
			m = global.MSG{
				"type": "text",
				"data": map[string]string{"text": o.Content},
			}
		case *message.LightAppElement:
			m = global.MSG{
				"type": "json",
				"data": map[string]string{"data": o.Content},
			}
		case *message.AtElement:
			target := "all"
			if o.Target != 0 {
				target = strconv.FormatUint(uint64(o.Target), 10)
			}
			m = global.MSG{
				"type": "at",
				"data": map[string]string{"qq": target},
			}
		case *message.RedBagElement:
			m = global.MSG{
				"type": "redbag",
				"data": map[string]string{"title": o.Title},
			}
		case *message.ForwardElement:
			m = global.MSG{
				"type": "forward",
				"data": map[string]string{"id": o.ResId},
			}
		case *message.FaceElement:
			m = global.MSG{
				"type": "face",
				"data": map[string]string{"id": strconv.FormatInt(int64(o.Index), 10)},
			}
		case *message.VoiceElement:
			m = global.MSG{
				"type": "record",
				"data": map[string]string{"file": o.Name, "url": o.Url},
			}
		case *message.ShortVideoElement:
			m = global.MSG{
				"type": "video",
				"data": map[string]string{"file": o.Name, "url": o.Url},
			}
		case *message.GroupImageElement:
			data := map[string]string{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url, "subType": strconv.FormatInt(int64(o.ImageBizType), 10)}
			switch {
			case o.Flash:
				data["type"] = "flash"
			case o.EffectID != 0:
				data["type"] = "show"
				data["id"] = strconv.FormatInt(int64(o.EffectID), 10)
			}
			m = global.MSG{
				"type": "image",
				"data": data,
			}
		case *message.GuildImageElement:
			data := map[string]string{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url}
			m = global.MSG{
				"type": "image",
				"data": data,
			}
		case *message.FriendImageElement:
			data := map[string]string{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url}
			if o.Flash {
				data["type"] = "flash"
			}
			m = global.MSG{
				"type": "image",
				"data": data,
			}
		case *message.DiceElement:
			m = global.MSG{
				"type": "dice",
				"data": map[string]string{"value": strconv.FormatInt(int64(o.Value), 10)},
			}
		case *message.MarketFaceElement:
			m = global.MSG{
				"type": "text",
				"data": map[string]string{"text": o.Name},
			}
		case *message.ServiceElement:
			if isOk := strings.Contains(o.Content, "<?xml"); isOk {
				m = global.MSG{
					"type": "xml",
					"data": map[string]string{"data": o.Content, "resid": strconv.FormatInt(int64(o.Id), 10)},
				}
			} else {
				m = global.MSG{
					"type": "json",
					"data": map[string]string{"data": o.Content, "resid": strconv.FormatInt(int64(o.Id), 10)},
				}
			}
		case *message.AnimatedSticker:
			m = global.MSG{
				"type": "face",
				"data": map[string]string{"id": strconv.FormatInt(int64(o.ID), 10), "type": "sticker"},
			}
		default:
			continue
		}
		r = append(r, m)
	}
	return
}

// ToStringMessage 将消息元素数组转为字符串以用于消息上报
func ToStringMessage(e []message.IMessageElement, source message.Source, isRaw ...bool) (r string) {
	sb := global.NewBuffer()
	sb.Reset()
	write := func(format string, a ...interface{}) {
		_, _ = fmt.Fprintf(sb, format, a...)
	}
	ur := false
	if len(isRaw) != 0 {
		ur = isRaw[0]
	}
	// 方便
	m := &message.SendingMessage{Elements: e}
	reply := m.FirstOrNil(func(e message.IMessageElement) bool {
		_, ok := e.(*message.ReplyElement)
		return ok
	})
	if reply != nil && source.SourceType&(message.SourceGroup|message.SourcePrivate) != 0 {
		replyElem := reply.(*message.ReplyElement)
		id := replyID(replyElem, source)
		if base.ExtraReplyData {
			write("[CQ:reply,id=%d,seq=%d,qq=%d,time=%d,text=%s]",
				id, replyElem.ReplySeq, replyElem.Sender, replyElem.Time,
				cqcode.EscapeValue(ToStringMessage(replyElem.Elements, source)))
		} else {
			write("[CQ:reply,id=%d]", id)
		}
	}
	for i, elem := range e {
		switch o := elem.(type) {
		case *message.ReplyElement:
			if base.RemoveReplyAt && len(e) > i+1 {
				elem, ok := e[i+1].(*message.AtElement)
				if ok && elem.Target == o.Sender {
					e[i+1] = nil
				}
			}
		case *message.TextElement:
			sb.WriteString(cqcode.EscapeText(o.Content))
		case *message.AtElement:
			if o.Target == 0 {
				write("[CQ:at,qq=all]")
				continue
			}
			write("[CQ:at,qq=%d]", uint64(o.Target))
		case *message.RedBagElement:
			write("[CQ:redbag,title=%s]", o.Title)
		case *message.ForwardElement:
			write("[CQ:forward,id=%s]", o.ResId)
		case *message.FaceElement:
			write(`[CQ:face,id=%d]`, o.Index)
		case *message.VoiceElement:
			if ur {
				write(`[CQ:record,file=%s]`, o.Name)
			} else {
				write(`[CQ:record,file=%s,url=%s]`, o.Name, cqcode.EscapeValue(o.Url))
			}
		case *message.ShortVideoElement:
			if ur {
				write(`[CQ:video,file=%s]`, o.Name)
			} else {
				write(`[CQ:video,file=%s,url=%s]`, o.Name, cqcode.EscapeValue(o.Url))
			}
		case *message.GroupImageElement:
			var arg string
			if o.Flash {
				arg = ",type=flash"
			} else if o.EffectID != 0 {
				arg = ",type=show,id=" + strconv.FormatInt(int64(o.EffectID), 10)
			}
			arg += ",subType=" + strconv.FormatInt(int64(o.ImageBizType), 10)
			if ur {
				write("[CQ:image,file=%x.image%s]", o.Md5, arg)
			} else {
				write("[CQ:image,file=%x.image,url=%s%s]", o.Md5, cqcode.EscapeValue(o.Url), arg)
			}
		case *message.FriendImageElement:
			var arg string
			if o.Flash {
				arg = ",type=flash"
			}
			if ur {
				write("[CQ:image,file=%x.image%s]", o.Md5, arg)
			} else {
				write("[CQ:image,file=%x.image,url=%s%s]", cqcode.EscapeValue(o.Url), arg)
			}
		case *LocalImageElement:
			var arg string
			if o.Flash {
				arg = ",type=flash"
			}
			data, err := os.ReadFile(o.File)
			if err == nil {
				m := md5.Sum(data)
				if ur {
					write("[CQ:image,file=%x.image%s]", m[:], arg)
				} else {
					write("[CQ:image,file=%x.image,url=%s%s]", m[:], cqcode.EscapeValue(o.URL), arg)
				}
			}
		case *message.GuildImageElement:
			write("[CQ:image,file=%x.image,url=%s]", o.Md5, cqcode.EscapeValue(o.Url))
		case *message.DiceElement:
			write("[CQ:dice,value=%v]", o.Value)
		case *message.MarketFaceElement:
			sb.WriteString(o.Name)
		case *message.ServiceElement:
			if isOk := strings.Contains(o.Content, "<?xml"); isOk {
				write(`[CQ:xml,data=%s,resid=%d]`, cqcode.EscapeValue(o.Content), o.Id)
			} else {
				write(`[CQ:json,data=%s,resid=%d]`, cqcode.EscapeValue(o.Content), o.Id)
			}
		case *message.LightAppElement:
			write(`[CQ:json,data=%s]`, cqcode.EscapeValue(o.Content))
		case *message.AnimatedSticker:
			write(`[CQ:face,id=%d,type=sticker]`, o.ID)
		}
	}
	r = sb.String() // 内部已拷贝
	global.PutBuffer(sb)
	return
}

// ToMessageContent 将消息转换成 Content. 忽略 Reply
// 不同于 onebot 的 Array Message, 此函数转换出来的 Content 的 data 段为实际类型
// 方便数据库查询
func ToMessageContent(e []message.IMessageElement) (r []global.MSG) {
	for _, elem := range e {
		var m global.MSG
		switch o := elem.(type) {
		case *message.TextElement:
			m = global.MSG{
				"type": "text",
				"data": global.MSG{"text": o.Content},
			}
		case *message.LightAppElement:
			m = global.MSG{
				"type": "json",
				"data": global.MSG{"data": o.Content},
			}
		case *message.AtElement:
			if o.Target == 0 {
				m = global.MSG{
					"type": "at",
					"data": global.MSG{
						"subType": "all",
					},
				}
			} else {
				m = global.MSG{
					"type": "at",
					"data": global.MSG{
						"subType": "user",
						"target":  o.Target,
						"display": o.Display,
					},
				}
			}
		case *message.RedBagElement:
			m = global.MSG{
				"type": "redbag",
				"data": global.MSG{"title": o.Title, "type": int(o.MsgType)},
			}
		case *message.ForwardElement:
			m = global.MSG{
				"type": "forward",
				"data": global.MSG{"id": o.ResId},
			}
		case *message.FaceElement:
			m = global.MSG{
				"type": "face",
				"data": global.MSG{"id": o.Index},
			}
		case *message.VoiceElement:
			m = global.MSG{
				"type": "record",
				"data": global.MSG{"file": o.Name, "url": o.Url},
			}
		case *message.ShortVideoElement:
			m = global.MSG{
				"type": "video",
				"data": global.MSG{"file": o.Name, "url": o.Url},
			}
		case *message.GroupImageElement:
			data := global.MSG{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url, "subType": uint32(o.ImageBizType)}
			switch {
			case o.Flash:
				data["type"] = "flash"
			case o.EffectID != 0:
				data["type"] = "show"
				data["id"] = o.EffectID
			}
			m = global.MSG{
				"type": "image",
				"data": data,
			}
		case *message.GuildImageElement:
			m = global.MSG{
				"type": "image",
				"data": global.MSG{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url},
			}
		case *message.FriendImageElement:
			data := global.MSG{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url}
			if o.Flash {
				data["type"] = "flash"
			}
			m = global.MSG{
				"type": "image",
				"data": data,
			}
		case *message.DiceElement:
			m = global.MSG{"type": "dice", "data": global.MSG{"value": o.Value}}
		case *message.MarketFaceElement:
			m = global.MSG{"type": "text", "data": global.MSG{"text": o.Name}}
		case *message.ServiceElement:
			if isOk := strings.Contains(o.Content, "<?xml"); isOk {
				m = global.MSG{
					"type": "xml",
					"data": global.MSG{"data": o.Content, "resid": o.Id},
				}
			} else {
				m = global.MSG{
					"type": "json",
					"data": global.MSG{"data": o.Content, "resid": o.Id},
				}
			}
		case *message.AnimatedSticker:
			m = global.MSG{
				"type": "face",
				"data": global.MSG{"id": o.ID, "type": "sticker"},
			}
		default:
			continue
		}
		r = append(r, m)
	}
	return
}

// ConvertStringMessage 将消息字符串转为消息元素数组
func (bot *CQBot) ConvertStringMessage(raw string, sourceType message.SourceType) (r []message.IMessageElement) {
	var t, key string
	d := map[string]string{}

	saveCQCode := func() {
		if t == "reply" { // reply 特殊处理
			if len(r) > 0 {
				if _, ok := r[0].(*message.ReplyElement); ok {
					log.Warnf("警告: 一条信息只能包含一个 Reply 元素.")
					return
				}
			}
			mid, err := strconv.Atoi(d["id"])
			customText := d["text"]
			switch {
			case customText != "":
				var elem *message.ReplyElement
				var org db.StoredMessage
				sender, senderErr := strconv.ParseInt(d["qq"], 10, 64)
				if senderErr != nil && err != nil {
					log.Warnf("警告: 自定义 Reply 元素中必须包含 Uin 或 id")
					break
				}
				msgTime, timeErr := strconv.ParseInt(d["time"], 10, 64)
				if timeErr != nil {
					msgTime = time.Now().Unix()
				}
				messageSeq, seqErr := strconv.ParseInt(d["seq"], 10, 64)
				if err == nil {
					org, _ = db.GetMessageByGlobalID(int32(mid))
				}
				if org != nil {
					elem = &message.ReplyElement{
						ReplySeq: org.GetAttribute().MessageSeq,
						Sender:   org.GetAttribute().SenderUin,
						Time:     int32(org.GetAttribute().Timestamp),
						Elements: bot.ConvertStringMessage(customText, sourceType),
					}
					if senderErr != nil {
						elem.Sender = sender
					}
					if timeErr != nil {
						elem.Time = int32(msgTime)
					}
					if seqErr != nil {
						elem.ReplySeq = int32(messageSeq)
					}
				} else {
					elem = &message.ReplyElement{
						ReplySeq: int32(messageSeq),
						Sender:   sender,
						Time:     int32(msgTime),
						Elements: bot.ConvertStringMessage(customText, sourceType),
					}
				}
				r = append([]message.IMessageElement{elem}, r...)
			case err == nil:
				org, err := db.GetMessageByGlobalID(int32(mid))
				if err == nil {
					r = append([]message.IMessageElement{
						&message.ReplyElement{
							ReplySeq: org.GetAttribute().MessageSeq,
							Sender:   org.GetAttribute().SenderUin,
							Time:     int32(org.GetAttribute().Timestamp),
							Elements: bot.ConvertContentMessage(org.GetContent(), sourceType),
						},
					}, r...)
				}
			default:
				log.Warnf("警告: Reply 元素中必须包含 text 或 id")
			}
			return
		}
		if t == "forward" { // 单独处理转发
			if id, ok := d["id"]; ok {
				if fwdMsg := bot.Client.DownloadForwardMessage(id); fwdMsg == nil {
					log.Warnf("警告: Forward 信息不存在或已过期")
				} else {
					r = []message.IMessageElement{fwdMsg}
				}
			} else {
				log.Warnf("警告: Forward 元素中必须包含 id")
			}
			return
		}
		elem, err := bot.ToElement(t, d, sourceType)
		if err != nil {
			org := "[CQ:" + t
			for k, v := range d {
				org += "," + k + "=" + v
			}
			org += "]"
			if !base.IgnoreInvalidCQCode {
				log.Warnf("转换CQ码 %v 时出现错误: %v 将原样发送.", org, err)
				r = append(r, message.NewText(org))
			} else {
				log.Warnf("转换CQ码 %v 时出现错误: %v 将忽略.", org, err)
			}
			return
		}
		switch i := elem.(type) {
		case message.IMessageElement:
			r = append(r, i)
		case []message.IMessageElement:
			r = append(r, i...)
		}
	}

	for raw != "" {
		i := 0
		for i < len(raw) && !(raw[i] == '[' && i+4 < len(raw) && raw[i:i+4] == "[CQ:") {
			i++
		}
		if i > 0 {
			if base.SplitURL {
				for _, txt := range param.SplitURL(cqcode.UnescapeText(raw[:i])) {
					r = append(r, message.NewText(txt))
				}
			} else {
				r = append(r, message.NewText(cqcode.UnescapeText(raw[:i])))
			}
		}

		if i+4 > len(raw) {
			return
		}
		raw = raw[i+4:] // skip "[CQ:"
		i = 0
		for i < len(raw) && raw[i] != ',' && raw[i] != ']' {
			i++
		}
		if i+1 > len(raw) {
			return
		}
		t = raw[:i]
		for k := range d { // clear the map, reuse it
			delete(d, k)
		}
		raw = raw[i:]
		i = 0
		for {
			if raw[0] == ']' {
				saveCQCode()
				raw = raw[1:]
				break
			}
			raw = raw[1:]

			for i < len(raw) && raw[i] != '=' {
				i++
			}
			if i+1 > len(raw) {
				return
			}
			key = raw[:i]
			raw = raw[i+1:] // skip "="
			i = 0
			for i < len(raw) && raw[i] != ',' && raw[i] != ']' {
				i++
			}

			if i+1 > len(raw) {
				return
			}
			d[key] = cqcode.UnescapeValue(raw[:i])
			raw = raw[i:]
			i = 0
		}
	}
	return
}

// ConvertObjectMessage 将消息JSON对象转为消息元素数组
func (bot *CQBot) ConvertObjectMessage(m gjson.Result, sourceType message.SourceType) (r []message.IMessageElement) {
	d := make(map[string]string)
	convertElem := func(e gjson.Result) {
		t := e.Get("type").Str
		if t == "reply" && sourceType&(message.SourceGroup|message.SourcePrivate) != 0 {
			if len(r) > 0 {
				if _, ok := r[0].(*message.ReplyElement); ok {
					log.Warnf("警告: 一条信息只能包含一个 Reply 元素.")
					return
				}
			}
			mid, err := strconv.Atoi(e.Get("data.id").String())
			customText := e.Get("data.text").String()
			switch {
			case customText != "":
				var elem *message.ReplyElement
				var org db.StoredMessage
				sender, senderErr := strconv.ParseInt(e.Get("data.[user_id,qq]").String(), 10, 64)
				if senderErr != nil && err != nil {
					log.Warnf("警告: 自定义 Reply 元素中必须包含 user_id 或 id")
					break
				}
				msgTime, timeErr := strconv.ParseInt(e.Get("data.time").String(), 10, 64)
				if timeErr != nil {
					msgTime = time.Now().Unix()
				}
				messageSeq, seqErr := strconv.ParseInt(e.Get("data.seq").String(), 10, 64)
				if err == nil {
					org, _ = db.GetMessageByGlobalID(int32(mid))
				}
				if org != nil {
					elem = &message.ReplyElement{
						ReplySeq: org.GetAttribute().MessageSeq,
						Sender:   org.GetAttribute().SenderUin,
						Time:     int32(org.GetAttribute().Timestamp),
						Elements: bot.ConvertStringMessage(customText, sourceType),
					}
					if senderErr != nil {
						elem.Sender = sender
					}
					if timeErr != nil {
						elem.Time = int32(msgTime)
					}
					if seqErr != nil {
						elem.ReplySeq = int32(messageSeq)
					}
				} else {
					elem = &message.ReplyElement{
						ReplySeq: int32(messageSeq),
						Sender:   sender,
						Time:     int32(msgTime),
						Elements: bot.ConvertStringMessage(customText, sourceType),
					}
				}
				r = append([]message.IMessageElement{elem}, r...)
			case err == nil:
				org, err := db.GetMessageByGlobalID(int32(mid))
				if err == nil {
					r = append([]message.IMessageElement{
						&message.ReplyElement{
							ReplySeq: org.GetAttribute().MessageSeq,
							Sender:   org.GetAttribute().SenderUin,
							Time:     int32(org.GetAttribute().Timestamp),
							Elements: bot.ConvertContentMessage(org.GetContent(), sourceType),
						},
					}, r...)
				}
			default:
				log.Warnf("警告: Reply 元素中必须包含 text 或 id")
			}
			return
		}
		if t == "forward" {
			id := e.Get("data.id").String()
			if id == "" {
				log.Warnf("警告: Forward 元素中必须包含 id")
			} else {
				if fwdMsg := bot.Client.DownloadForwardMessage(id); fwdMsg == nil {
					log.Warnf("警告: Forward 信息不存在或已过期")
				} else {
					r = []message.IMessageElement{fwdMsg}
				}
			}
			return
		}
		for i := range d {
			delete(d, i)
		}
		e.Get("data").ForEach(func(key, value gjson.Result) bool {
			d[key.Str] = value.String()
			return true
		})
		elem, err := bot.ToElement(t, d, sourceType)
		if err != nil {
			log.Warnf("转换CQ码 (%v) 到MiraiGo Element时出现错误: %v 将忽略本段CQ码.", e.Raw, err)
			return
		}
		switch i := elem.(type) {
		case message.IMessageElement:
			r = append(r, i)
		case []message.IMessageElement:
			r = append(r, i...)
		}
	}
	if m.Type == gjson.String {
		return bot.ConvertStringMessage(m.Str, sourceType)
	}
	if m.IsArray() {
		m.ForEach(func(_, e gjson.Result) bool {
			convertElem(e)
			return true
		})
	}
	if m.IsObject() {
		convertElem(m)
	}
	return
}

// ConvertContentMessage 将数据库用的 content 转换为消息元素数组
func (bot *CQBot) ConvertContentMessage(content []global.MSG, sourceType message.SourceType) (r []message.IMessageElement) {
	for _, c := range content {
		data := c["data"].(global.MSG)
		switch c["type"] {
		case "text":
			r = append(r, message.NewText(data["text"].(string)))
		case "image":
			u, ok := data["url"]
			d := make(map[string]string, 2)
			if ok {
				d["url"] = u.(string)
			}
			d["file"] = data["file"].(string)
			e, err := bot.makeImageOrVideoElem(d, false, sourceType)
			if err != nil {
				log.Warnf("make image elem error: %v", err)
				continue
			}
			flash, id := false, int32(0)
			if t, ok := data["type"]; ok {
				if t.(string) == "flash" {
					flash = true
				}
				if t.(string) == "show" {
					id = data["id"].(int32)
					if id < 40000 || id >= 40006 {
						id = 40000
					}
				}
			}
			switch img := e.(type) {
			case *LocalImageElement:
				img.Flash = flash
				img.EffectID = id
			case *message.GroupImageElement:
				img.Flash = flash
				img.EffectID = id
				img.ImageBizType = message.ImageBizType(data["subType"].(uint32))
			case *message.FriendImageElement:
				img.Flash = flash
			}
			r = append(r, e)
		case "at":
			switch data["subType"].(string) {
			case "all":
				r = append(r, message.NewAt(0))
			case "user":
				r = append(r, message.NewAt(data["target"].(int64), data["display"].(string)))
			default:
				continue
			}
		case "redbag":
			r = append(r, &message.RedBagElement{
				MsgType: message.RedBagMessageType(data["type"].(int)),
				Title:   data["title"].(string),
			})
		case "forward":
			r = append(r, &message.ForwardElement{
				ResId: data["id"].(string),
			})
		case "face":
			r = append(r, message.NewFace(data["id"].(int32)))
		case "video":
			e, err := bot.makeImageOrVideoElem(map[string]string{"file": data["file"].(string)}, true, sourceType)
			if err != nil {
				log.Warnf("make image elem error: %v", err)
				continue
			}
			r = append(r, e)
		}
	}
	return
}

// ToElement 将解码后的CQCode转换为Element.
//
// 返回 interface{} 存在三种类型
//
// message.IMessageElement []message.IMessageElement nil
func (bot *CQBot) ToElement(t string, d map[string]string, sourceType message.SourceType) (m interface{}, err error) {
	switch t {
	case "text":
		if base.SplitURL {
			var ret []message.IMessageElement
			for _, text := range param.SplitURL(d["text"]) {
				ret = append(ret, message.NewText(text))
			}
			return ret, nil
		}
		return message.NewText(d["text"]), nil
	case "image":
		img, err := bot.makeImageOrVideoElem(d, false, sourceType)
		if err != nil {
			return nil, err
		}
		tp := d["type"]
		flash, id := false, int64(0)
		switch tp {
		case "flash":
			flash = true
		case "show":
			id, _ = strconv.ParseInt(d["id"], 10, 64)
			if id < 40000 || id >= 40006 {
				id = 40000
			}
		default:
			return img, err
		}
		switch img := img.(type) {
		case *LocalImageElement:
			img.Flash = flash
			img.EffectID = int32(id)
		case *message.GroupImageElement:
			img.Flash = flash
			img.EffectID = int32(id)
			i, _ := strconv.ParseInt(d["subType"], 10, 64)
			img.ImageBizType = message.ImageBizType(i)
		case *message.FriendImageElement:
			img.Flash = flash
		}
		return img, err
	case "poke":
		t, _ := strconv.ParseInt(d["qq"], 10, 64)
		return &PokeElement{Target: t}, nil
	case "tts":
		defer func() {
			if r := recover(); r != nil {
				m = nil
				err = errors.New("tts 转换失败")
			}
		}()
		data, err := bot.Client.GetTts(d["text"])
		if err != nil {
			return nil, err
		}
		return &message.VoiceElement{Data: base.ResampleSilk(data)}, nil
	case "record":
		f := d["file"]
		data, err := global.FindFile(f, d["cache"], global.VoicePath)
		if err != nil {
			return nil, err
		}
		if !global.IsAMRorSILK(data) {
			lawful, mt := base.IsLawfulAudio(bytes.NewReader(data))
			if !lawful {
				return nil, errors.New("audio type error: " + mt)
			}
			data, err = global.EncoderSilk(data)
			if err != nil {
				return nil, err
			}
		}
		return &message.VoiceElement{Data: data}, nil
	case "face":
		id, err := strconv.Atoi(d["id"])
		if err != nil {
			return nil, err
		}
		if d["type"] == "sticker" {
			return &message.AnimatedSticker{ID: int32(id)}, nil
		}
		return message.NewFace(int32(id)), nil
	case "at":
		qq := d["qq"]
		if qq == "all" {
			return message.AtAll(), nil
		}
		t, err := strconv.ParseInt(qq, 10, 64)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSpace(d["name"])
		if len(name) > 0 {
			name = "@" + name
		}
		return message.NewAt(t, name), nil
	case "share":
		return message.NewUrlShare(d["url"], d["title"], d["content"], d["image"]), nil
	case "music":
		if d["type"] == "qq" {
			info, err := global.QQMusicSongInfo(d["id"])
			if err != nil {
				return nil, err
			}
			if !info.Get("track_info").Exists() {
				return nil, errors.New("song not found")
			}
			name := info.Get("track_info.name").Str
			mid := info.Get("track_info.mid").Str
			albumMid := info.Get("track_info.album.mid").Str
			pinfo, _ := global.GetBytes("http://u.y.qq.com/cgi-bin/musicu.fcg?g_tk=2034008533&uin=0&format=json&data={\"comm\":{\"ct\":23,\"cv\":0},\"url_mid\":{\"module\":\"vkey.GetVkeyServer\",\"method\":\"CgiGetVkey\",\"param\":{\"guid\":\"4311206557\",\"songmid\":[\"" + mid + "\"],\"songtype\":[0],\"uin\":\"0\",\"loginflag\":1,\"platform\":\"23\"}}}&_=1599039471576")
			jumpURL := "https://i.y.qq.com/v8/playsong.html?platform=11&appshare=android_qq&appversion=10030010&hosteuin=oKnlNenz7i-s7c**&songmid=" + mid + "&type=0&appsongtype=1&_wv=1&source=qq&ADTAG=qfshare"
			purl := gjson.ParseBytes(pinfo).Get("url_mid.data.midurlinfo.0.purl").Str
			preview := "http://y.gtimg.cn/music/photo_new/T002R180x180M000" + albumMid + ".jpg"
			content := info.Get("track_info.singer.0.name").Str
			if d["content"] != "" {
				content = d["content"]
			}
			return &message.MusicShareElement{
				MusicType:  message.QQMusic,
				Title:      name,
				Summary:    content,
				Url:        jumpURL,
				PictureUrl: preview,
				MusicUrl:   purl,
			}, nil
		}
		if d["type"] == "163" {
			info, err := global.NeteaseMusicSongInfo(d["id"])
			if err != nil {
				return nil, err
			}
			if !info.Exists() {
				return nil, errors.New("song not found")
			}
			name := info.Get("name").Str
			jumpURL := "https://y.music.163.com/m/song/" + d["id"]
			musicURL := "http://music.163.com/song/media/outer/url?id=" + d["id"]
			picURL := info.Get("album.picUrl").Str
			artistName := ""
			if info.Get("artists.0").Exists() {
				artistName = info.Get("artists.0.name").Str
			}
			return &message.MusicShareElement{
				MusicType:  message.CloudMusic,
				Title:      name,
				Summary:    artistName,
				Url:        jumpURL,
				PictureUrl: picURL,
				MusicUrl:   musicURL,
			}, nil
		}
		if d["type"] == "custom" {
			if d["subtype"] != "" {
				var subType int
				switch d["subtype"] {
				default:
					subType = message.QQMusic
				case "163":
					subType = message.CloudMusic
				case "migu":
					subType = message.MiguMusic
				case "kugou":
					subType = message.KugouMusic
				case "kuwo":
					subType = message.KuwoMusic
				}
				return &message.MusicShareElement{
					MusicType:  subType,
					Title:      d["title"],
					Summary:    d["content"],
					Url:        d["url"],
					PictureUrl: d["image"],
					MusicUrl:   d["audio"],
				}, nil
			}
			xml := fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="2" templateID="1" action="web" brief="[分享] %s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="2"><audio cover="%s" src="%s"/><title>%s</title><summary>%s</summary></item><source name="音乐" icon="https://i.gtimg.cn/open/app_icon/01/07/98/56/1101079856_100_m.png" url="http://web.p.qq.com/qqmpmobile/aio/app.html?id=1101079856" action="app" a_actionData="com.tencent.qqmusic" i_actionData="tencent1101079856://" appid="1101079856" /></msg>`,
				utils.XmlEscape(d["title"]), d["url"], d["image"], d["audio"], utils.XmlEscape(d["title"]), utils.XmlEscape(d["content"]))
			return &message.ServiceElement{
				Id:      60,
				Content: xml,
				SubType: "music",
			}, nil
		}
		return nil, errors.New("unsupported music type: " + d["type"])
	case "dice":
		value := d["value"]
		i, _ := strconv.ParseInt(value, 10, 64)
		if i < 0 || i > 6 {
			return nil, errors.New("invalid dice value " + value)
		}
		return message.NewDice(int32(i)), nil
	case "xml":
		resID := d["resid"]
		template := cqcode.EscapeValue(d["data"])
		i, _ := strconv.ParseInt(resID, 10, 64)
		msg := message.NewRichXml(template, i)
		return msg, nil
	case "json":
		resID := d["resid"]
		i, _ := strconv.ParseInt(resID, 10, 64)
		if i == 0 {
			// 默认情况下走小程序通道
			msg := message.NewLightApp(d["data"])
			return msg, nil
		}
		// resid不为0的情况下走富文本通道，后续补全透传service Id，此处暂时不处理 TODO
		msg := message.NewRichJson(d["data"])
		return msg, nil
	case "cardimage":
		source := d["source"]
		icon := d["icon"]
		brief := d["brief"]
		parseIntWithDefault := func(name string, origin int64) int64 {
			v, _ := strconv.ParseInt(d[name], 10, 64)
			if v <= 0 {
				return origin
			}
			return v
		}
		minWidth := parseIntWithDefault("minwidth", 200)
		maxWidth := parseIntWithDefault("maxwidth", 500)
		minHeight := parseIntWithDefault("minheight", 200)
		maxHeight := parseIntWithDefault("maxheight", 1000)
		img, err := bot.makeImageOrVideoElem(d, false, sourceType)
		if err != nil {
			return nil, errors.New("send cardimage faild")
		}
		return bot.makeShowPic(img, source, brief, icon, minWidth, minHeight, maxWidth, maxHeight, sourceType == message.SourceGroup)
	case "video":
		file, err := bot.makeImageOrVideoElem(d, true, sourceType)
		if err != nil {
			return nil, err
		}
		v, ok := file.(*LocalVideoElement)
		if !ok {
			return file, nil
		}
		if v.File == "" {
			return v, nil
		}
		var data []byte
		if cover, ok := d["cover"]; ok {
			data, _ = global.FindFile(cover, d["cache"], global.ImagePath)
		} else {
			err = global.ExtractCover(v.File, v.File+".jpg")
			if err != nil {
				return nil, err
			}
			data, _ = os.ReadFile(v.File + ".jpg")
		}
		v.thumb = bytes.NewReader(data)
		video, _ := os.Open(v.File)
		defer video.Close()
		_, _ = video.Seek(4, io.SeekStart)
		header := make([]byte, 4)
		_, _ = video.Read(header)
		if !bytes.Equal(header, []byte{0x66, 0x74, 0x79, 0x70}) { // check file header ftyp
			_, _ = video.Seek(0, io.SeekStart)
			hash, _ := utils.ComputeMd5AndLength(video)
			cacheFile := path.Join(global.CachePath, hex.EncodeToString(hash)+".mp4")
			if !(d["cache"] == "" || d["cache"] == "1") || !global.PathExists(cacheFile) {
				err = global.EncodeMP4(v.File, cacheFile)
				if err != nil {
					return nil, err
				}
			}
			v.File = cacheFile
		}
		return v, nil
	default:
		return nil, errors.New("unsupported cq code: " + t)
	}
}

// makeImageOrVideoElem 图片 elem 生成器，单独拎出来，用于公用
func (bot *CQBot) makeImageOrVideoElem(d map[string]string, video bool, sourceType message.SourceType) (message.IMessageElement, error) {
	f := d["file"]
	u, ok := d["url"]
	if !ok {
		u = ""
	}
	if strings.HasPrefix(f, "http") {
		hash := md5.Sum([]byte(f))
		cacheFile := path.Join(global.CachePath, hex.EncodeToString(hash[:])+".cache")
		maxSize := int64(maxImageSize)
		if video {
			maxSize = maxVideoSize
		}
		thread, _ := strconv.Atoi(d["c"])
		exist := global.PathExists(cacheFile)
		if exist && (d["cache"] == "" || d["cache"] == "1") {
			goto useCacheFile
		}
		if exist {
			_ = os.Remove(cacheFile)
		}
		if err := global.DownloadFileMultiThreading(f, cacheFile, maxSize, thread, nil); err != nil {
			return nil, err
		}
	useCacheFile:
		if video {
			return &LocalVideoElement{File: cacheFile}, nil
		}
		return &LocalImageElement{File: cacheFile, URL: f}, nil
	}
	if strings.HasPrefix(f, "file") {
		fu, err := url.Parse(f)
		if err != nil {
			return nil, err
		}
		if runtime.GOOS == `windows` && strings.HasPrefix(fu.Path, "/") {
			fu.Path = fu.Path[1:]
		}
		info, err := os.Stat(fu.Path)
		if err != nil {
			if !os.IsExist(err) {
				return nil, errors.New("file not found")
			}
			return nil, err
		}
		if video {
			if info.Size() == 0 || info.Size() >= maxVideoSize {
				return nil, errors.New("invalid video size")
			}
			return &LocalVideoElement{File: fu.Path}, nil
		}
		if info.Size() == 0 || info.Size() >= maxImageSize {
			return nil, errors.New("invalid image size")
		}
		return &LocalImageElement{File: fu.Path, URL: f}, nil
	}
	if !video && strings.HasPrefix(f, "base64") {
		b, err := param.Base64DecodeString(strings.TrimPrefix(f, "base64://"))
		if err != nil {
			return nil, err
		}
		return &LocalImageElement{Stream: bytes.NewReader(b), URL: f}, nil
	}
	rawPath := path.Join(global.ImagePath, f)
	if video {
		if strings.HasSuffix(f, ".video") {
			hash, err := hex.DecodeString(strings.TrimSuffix(f, ".video"))
			if err == nil {
				if b := cache.Video.Get(hash); b != nil {
					return bot.readVideoCache(b), nil
				}
			}
		}
		rawPath = path.Join(global.VideoPath, f)
		if !global.PathExists(rawPath) {
			return nil, errors.New("invalid video")
		}
		if path.Ext(rawPath) != ".video" {
			return &LocalVideoElement{File: rawPath}, nil
		}
		b, _ := os.ReadFile(rawPath)
		return bot.readVideoCache(b), nil
	}
	// 目前频道内上传的图片均无法被查询到, 需要单独处理
	if sourceType == message.SourceGuildChannel {
		cacheFile := path.Join(global.ImagePath, "guild-images", f)
		if global.PathExists(cacheFile) {
			return &LocalImageElement{File: cacheFile}, nil
		}
	}
	if strings.HasSuffix(f, ".image") {
		hash, err := hex.DecodeString(strings.TrimSuffix(f, ".image"))
		if err == nil {
			if b := cache.Image.Get(hash); b != nil {
				return bot.readImageCache(b, sourceType)
			}
		}
	}
	exist := global.PathExists(rawPath)
	if !exist {
		if d["url"] != "" {
			return bot.makeImageOrVideoElem(map[string]string{"file": d["url"]}, false, sourceType)
		}
		return nil, errors.New("invalid image")
	}
	if path.Ext(rawPath) != ".image" {
		return &LocalImageElement{File: rawPath, URL: u}, nil
	}
	b, err := os.ReadFile(rawPath)
	if err != nil {
		return nil, err
	}
	return bot.readImageCache(b, sourceType)
}

func (bot *CQBot) readImageCache(b []byte, sourceType message.SourceType) (message.IMessageElement, error) {
	var err error
	if len(b) < 20 {
		return nil, errors.New("invalid cache")
	}
	r := binary.NewReader(b)
	hash := r.ReadBytes(16)
	size := r.ReadInt32()
	r.ReadString()
	imageURL := r.ReadString()
	if size == 0 && imageURL != "" {
		return bot.makeImageOrVideoElem(map[string]string{"file": imageURL}, false, sourceType)
	}
	var rsp message.IMessageElement
	switch sourceType { // nolint:exhaustive
	case message.SourceGroup:
		rsp, err = bot.Client.QueryGroupImage(int64(rand.Uint32()), hash, size)
	case message.SourceGuildChannel:
		if len(bot.Client.GuildService.Guilds) == 0 {
			err = errors.New("cannot query guild image: not any joined guild")
			break
		}
		guild := bot.Client.GuildService.Guilds[0]
		rsp, err = bot.Client.GuildService.QueryImage(guild.GuildId, guild.Channels[0].ChannelId, hash, uint64(size))
	default:
		rsp, err = bot.Client.QueryFriendImage(int64(rand.Uint32()), hash, size)
	}
	if err != nil && imageURL != "" {
		return bot.makeImageOrVideoElem(map[string]string{"file": imageURL}, false, sourceType)
	}
	return rsp, err
}

func (bot *CQBot) readVideoCache(b []byte) message.IMessageElement {
	r := binary.NewReader(b)
	return &message.ShortVideoElement{ // todo 检查缓存是否有效
		Md5:       r.ReadBytes(16),
		ThumbMd5:  r.ReadBytes(16),
		Size:      r.ReadInt32(),
		ThumbSize: r.ReadInt32(),
		Name:      r.ReadString(),
		Uuid:      r.ReadAvailable(),
	}
}

// makeShowPic 一种xml 方式发送的群消息图片
func (bot *CQBot) makeShowPic(elem message.IMessageElement, source string, brief string, icon string, minWidth int64, minHeight int64, maxWidth int64, maxHeight int64, group bool) ([]message.IMessageElement, error) {
	xml := ""
	var suf message.IMessageElement
	if brief == "" {
		brief = "&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！"
	}
	if local, ok := elem.(*LocalImageElement); ok {
		r := rand.Uint32()
		typ := message.SourceGroup
		if !group {
			typ = message.SourcePrivate
		}
		e, err := bot.uploadLocalImage(message.Source{SourceType: typ, PrimaryID: int64(r)}, local)
		if err != nil {
			log.Warnf("警告: 图片上传失败: %v", err)
			return nil, err
		}
		elem = e
	}
	switch i := elem.(type) {
	case *message.GroupImageElement:
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="%s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, brief, "", i.Md5, i.Md5, 0, "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		suf = i
	case *message.FriendImageElement:
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="%s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, brief, "", i.Md5, i.Md5, 0, "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		suf = i
	}
	if xml == "" {
		return nil, errors.New("生成xml图片消息失败")
	}
	ret := []message.IMessageElement{suf, message.NewRichXml(xml, 5)}
	return ret, nil
}
