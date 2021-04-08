package coolq

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	xml2 "encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/Mrs4s/go-cqhttp/global"
)

/*
var matchReg = regexp.MustCompile(`\[CQ:\w+?.*?]`)
var typeReg = regexp.MustCompile(`\[CQ:(\w+)`)
var paramReg = regexp.MustCompile(`,([\w\-.]+?)=([^,\]]+)`)
*/

// RemoveReplyAt 是否删除reply后的at
var RemoveReplyAt bool

// ExtraReplyData 是否上报额外reply信息
var ExtraReplyData bool

// IgnoreInvalidCQCode 是否忽略无效CQ码
var IgnoreInvalidCQCode = false

// SplitURL 是否分割URL
var SplitURL = false

// magicCQ 代表 uint32([]byte("[CQ:"))
var magicCQ = uint32(0)

func init() {
	CQHeader := "[CQ:"
	magicCQ = *(*uint32)(unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&CQHeader)).Data))
}

// add 指针运算
func add(ptr unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(ptr) + offset)
}

const (
	maxImageSize = 1024 * 1024 * 30  // 30MB
	maxVideoSize = 1024 * 1024 * 100 // 100MB
)

// PokeElement 拍一拍
type PokeElement struct {
	Target int64
}

// GiftElement 礼物
type GiftElement struct {
	Target int64
	GiftID message.GroupGift
}

// LocalImageElement 本地图片
type LocalImageElement struct {
	message.ImageElement
	Stream io.ReadSeeker
	File   string
}

// LocalVoiceElement 本地语音
type LocalVoiceElement struct {
	message.VoiceElement
	Stream io.ReadSeeker
}

// LocalVideoElement 本地视频
type LocalVideoElement struct {
	message.ShortVideoElement
	File  string
	thumb io.ReadSeeker
}

// Type 获取元素类型ID
func (e *GiftElement) Type() message.ElementType {
	// Make message.IMessageElement Happy
	return message.At
}

// GiftID 礼物ID数组
var GiftID = [...]message.GroupGift{
	message.SweetWink,
	message.HappyCola,
	message.LuckyBracelet,
	message.Cappuccino,
	message.CatWatch,
	message.FleeceGloves,
	message.RainbowCandy,
	message.Stronger,
	message.LoveMicrophone,
	message.HoldingYourHand,
	message.CuteCat,
	message.MysteryMask,
	message.ImBusy,
	message.LoveMask,
}

// Type 获取元素类型ID
func (e *PokeElement) Type() message.ElementType {
	// Make message.IMessageElement Happy
	return message.At
}

// ToArrayMessage 将消息元素数组转为MSG数组以用于消息上报
func ToArrayMessage(e []message.IMessageElement, id int64, isRaw ...bool) (r []MSG) {
	r = make([]MSG, 0, len(e))
	ur := false
	if len(isRaw) != 0 {
		ur = isRaw[0]
	}
	m := &message.SendingMessage{Elements: e}
	reply := m.FirstOrNil(func(e message.IMessageElement) bool {
		_, ok := e.(*message.ReplyElement)
		return ok
	})
	if reply != nil {
		replyElem := reply.(*message.ReplyElement)
		if ExtraReplyData {
			r = append(r, MSG{
				"type": "reply",
				"data": map[string]string{
					"id":   fmt.Sprint(toGlobalID(id, reply.(*message.ReplyElement).ReplySeq)),
					"seq":  string(replyElem.ReplySeq),
					"qq":   strconv.FormatInt(replyElem.Sender, 10),
					"time": string(replyElem.Time),
					"text": CQCodeEscapeValue(CQCodeEscapeText(ToStringMessage(replyElem.Elements, id))),
				},
			})
		} else {
			r = append(r, MSG{
				"type": "reply",
				"data": map[string]string{"id": fmt.Sprint(toGlobalID(id, replyElem.ReplySeq))},
			})
		}
	}
	for i, elem := range e {
		var m MSG
		switch o := elem.(type) {
		case *message.ReplyElement:
			if RemoveReplyAt && len(e) > i+1 {
				elem, ok := e[i+1].(*message.AtElement)
				if ok && elem.Target == o.Sender {
					e[i+1] = nil
				}
			}
		case *message.TextElement:
			m = MSG{
				"type": "text",
				"data": map[string]string{"text": o.Content},
			}
		case *message.LightAppElement:
			// m = MSG{
			// 	"type": "text",
			// 	"data": map[string]string{"text": o.Content},
			// }
			m = MSG{
				"type": "json",
				"data": map[string]string{"data": o.Content},
			}
		case *message.AtElement:
			if o.Target == 0 {
				m = MSG{
					"type": "at",
					"data": map[string]string{"qq": "all"},
				}
			} else {
				m = MSG{
					"type": "at",
					"data": map[string]string{"qq": fmt.Sprint(o.Target)},
				}
			}
		case *message.RedBagElement:
			m = MSG{
				"type": "redbag",
				"data": map[string]string{"title": o.Title},
			}
		case *message.ForwardElement:
			m = MSG{
				"type": "forward",
				"data": map[string]string{"id": o.ResId},
			}
		case *message.FaceElement:
			m = MSG{
				"type": "face",
				"data": map[string]string{"id": fmt.Sprint(o.Index)},
			}
		case *message.VoiceElement:
			if ur {
				m = MSG{
					"type": "record",
					"data": map[string]string{"file": o.Name},
				}
			} else {
				m = MSG{
					"type": "record",
					"data": map[string]string{"file": o.Name, "url": o.Url},
				}
			}
		case *message.ShortVideoElement:
			if ur {
				m = MSG{
					"type": "video",
					"data": map[string]string{"file": o.Name},
				}
			} else {
				m = MSG{
					"type": "video",
					"data": map[string]string{"file": o.Name, "url": o.Url},
				}
			}
		case *message.ImageElement:
			if ur {
				m = MSG{
					"type": "image",
					"data": map[string]string{"file": o.Filename},
				}
			} else {
				m = MSG{
					"type": "image",
					"data": map[string]string{"file": o.Filename, "url": o.Url},
				}
			}
		case *message.GroupImageElement:
			if ur {
				m = MSG{
					"type": "image",
					"data": map[string]string{"file": hex.EncodeToString(o.Md5) + ".image"},
				}
			} else {
				m = MSG{
					"type": "image",
					"data": map[string]string{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url},
				}
			}
		case *message.FriendImageElement:
			if ur {
				m = MSG{
					"type": "image",
					"data": map[string]string{"file": hex.EncodeToString(o.Md5) + ".image"},
				}
			} else {
				m = MSG{
					"type": "image",
					"data": map[string]string{"file": hex.EncodeToString(o.Md5) + ".image", "url": o.Url},
				}
			}
		case *message.GroupFlashImgElement:
			return []MSG{{
				"type": "image",
				"data": map[string]string{"file": o.Filename, "type": "flash"},
			}}
		case *message.FriendFlashImgElement:
			return []MSG{{
				"type": "image",
				"data": map[string]string{"file": o.Filename, "type": "flash"},
			}}
		case *message.ServiceElement:
			if isOk := strings.Contains(o.Content, "<?xml"); isOk {
				m = MSG{
					"type": "xml",
					"data": map[string]string{"data": o.Content, "resid": fmt.Sprintf("%d", o.Id)},
				}
			} else {
				m = MSG{
					"type": "json",
					"data": map[string]string{"data": o.Content, "resid": fmt.Sprintf("%d", o.Id)},
				}
			}
		default:
			continue
		}
		if m != nil {
			r = append(r, m)
		}
	}
	return
}

// ToStringMessage 将消息元素数组转为字符串以用于消息上报
func ToStringMessage(e []message.IMessageElement, id int64, isRaw ...bool) (r string) {
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
	if reply != nil {
		replyElem := reply.(*message.ReplyElement)
		if ExtraReplyData {
			r += fmt.Sprintf("[CQ:reply,id=%d,seq=%d,qq=%d,time=%d,text=%s]",
				toGlobalID(id, replyElem.ReplySeq),
				replyElem.ReplySeq, replyElem.Sender, replyElem.Time,
				CQCodeEscapeValue(CQCodeEscapeText(ToStringMessage(replyElem.Elements, id))))
		} else {
			r += fmt.Sprintf("[CQ:reply,id=%d]", toGlobalID(id, replyElem.ReplySeq))
		}
	}
	for i, elem := range e {
		switch o := elem.(type) {
		case *message.ReplyElement:
			if RemoveReplyAt && len(e) > i+1 {
				elem, ok := e[i+1].(*message.AtElement)
				if ok && elem.Target == o.Sender {
					e[i+1] = nil
				}
			}
		case *message.TextElement:
			r += CQCodeEscapeText(o.Content)
		case *message.AtElement:
			if o.Target == 0 {
				r += "[CQ:at,qq=all]"
				continue
			}
			r += fmt.Sprintf("[CQ:at,qq=%d]", o.Target)
		case *message.RedBagElement:
			r += fmt.Sprintf("[CQ:redbag,title=%s]", o.Title)
		case *message.ForwardElement:
			r += fmt.Sprintf("[CQ:forward,id=%s]", o.ResId)
		case *message.FaceElement:
			r += fmt.Sprintf(`[CQ:face,id=%d]`, o.Index)
		case *message.VoiceElement:
			if ur {
				r += fmt.Sprintf(`[CQ:record,file=%s]`, o.Name)
			} else {
				r += fmt.Sprintf(`[CQ:record,file=%s,url=%s]`, o.Name, CQCodeEscapeValue(o.Url))
			}
		case *message.ShortVideoElement:
			if ur {
				r += fmt.Sprintf(`[CQ:video,file=%s]`, o.Name)
			} else {
				r += fmt.Sprintf(`[CQ:video,file=%s,url=%s]`, o.Name, CQCodeEscapeValue(o.Url))
			}
		case *message.ImageElement:
			if ur {
				r += fmt.Sprintf(`[CQ:image,file=%s]`, o.Filename)
			} else {
				r += fmt.Sprintf(`[CQ:image,file=%s,url=%s]`, o.Filename, CQCodeEscapeValue(o.Url))
			}
		case *message.GroupImageElement:
			if ur {
				r += fmt.Sprintf("[CQ:image,file=%s]", hex.EncodeToString(o.Md5)+".image")
			} else {
				r += fmt.Sprintf("[CQ:image,file=%s,url=%s]", hex.EncodeToString(o.Md5)+".image", CQCodeEscapeValue(o.Url))
			}
		case *message.FriendImageElement:
			if ur {
				r += fmt.Sprintf("[CQ:image,file=%s]", hex.EncodeToString(o.Md5)+".image")
			} else {
				r += fmt.Sprintf("[CQ:image,file=%s,url=%s]", hex.EncodeToString(o.Md5)+".image", CQCodeEscapeValue(o.Url))
			}
		case *message.GroupFlashImgElement:
			return fmt.Sprintf("[CQ:image,type=flash,file=%s]", o.Filename)
		case *message.FriendFlashImgElement:
			return fmt.Sprintf("[CQ:image,type=flash,file=%s]", o.Filename)
		case *message.ServiceElement:
			if isOk := strings.Contains(o.Content, "<?xml"); isOk {
				r += fmt.Sprintf(`[CQ:xml,data=%s,resid=%d]`, CQCodeEscapeValue(o.Content), o.Id)
			} else {
				r += fmt.Sprintf(`[CQ:json,data=%s,resid=%d]`, CQCodeEscapeValue(o.Content), o.Id)
			}
		case *message.LightAppElement:
			r += fmt.Sprintf(`[CQ:json,data=%s]`, CQCodeEscapeValue(o.Content))
			// r += CQCodeEscapeText(o.Content)
		}
	}
	return
}

// ConvertStringMessage 将消息字符串转为消息元素数组
func (bot *CQBot) ConvertStringMessage(s string, isGroup bool) (r []message.IMessageElement) {
	var t, key string
	d := map[string]string{}
	ptr := unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&s)).Data)
	l := len(s)
	i, j, CQBegin := 0, 0, 0

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
				var org MSG
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
					org = bot.GetMessage(int32(mid))
				}
				if org != nil {
					elem = &message.ReplyElement{
						ReplySeq: org["message-id"].(int32),
						Sender:   org["sender"].(message.Sender).Uin,
						Time:     org["time"].(int32),
						Elements: bot.ConvertStringMessage(customText, isGroup),
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
						Elements: bot.ConvertStringMessage(customText, isGroup),
					}
				}
				r = append([]message.IMessageElement{elem}, r...)
			case err == nil:
				org := bot.GetMessage(int32(mid))
				if org != nil {
					r = append([]message.IMessageElement{
						&message.ReplyElement{
							ReplySeq: org["message-id"].(int32),
							Sender:   org["sender"].(message.Sender).Uin,
							Time:     org["time"].(int32),
							Elements: bot.ConvertStringMessage(org["message"].(string), isGroup),
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
		elem, err := bot.ToElement(t, d, isGroup)
		if err != nil {
			org := s[CQBegin:i]
			if !IgnoreInvalidCQCode {
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

S1: // Plain Text
	for ; i < l; i++ {
		if *(*byte)(add(ptr, uintptr(i))) == '[' && i+4 < l &&
			*(*uint32)(add(ptr, uintptr(i))) == magicCQ { // Magic :uint32([]byte("[CQ:"))
			if i > j {
				if SplitURL {
					for _, str := range global.SplitURL(CQCodeUnescapeText(s[j:i])) {
						r = append(r, message.NewText(str))
					}
				} else {
					r = append(r, message.NewText(CQCodeUnescapeText(s[j:i])))
				}
			}
			CQBegin = i
			i += 4
			j = i
			goto S2
		}
	}
	goto End
S2: // CQCode Type
	for k := range d { // 内存复用，减小GC压力
		delete(d, k)
	}
	for ; i < l; i++ {
		switch *(*byte)(add(ptr, uintptr(i))) {
		case ',': // CQ Code with params
			t = s[j:i]
			i++
			j = i
			goto S3
		case ']': // CQ Code without params
			t = s[j:i]
			i++
			j = i
			saveCQCode()
			goto S1
		}
	}
	goto End
S3: // CQCode param key
	for ; i < l; i++ {
		if *(*byte)(add(ptr, uintptr(i))) == '=' {
			key = s[j:i]
			i++
			j = i
			goto S4
		}
	}
	goto End
S4: // CQCode param value
	for ; i < l; i++ {
		switch *(*byte)(add(ptr, uintptr(i))) {
		case ',': // more param
			d[key] = CQCodeUnescapeValue(s[j:i])
			i++
			j = i
			goto S3
		case ']':
			d[key] = CQCodeUnescapeValue(s[j:i])
			i++
			j = i
			saveCQCode()
			goto S1
		}
	}
	goto End
End:
	if i > j {
		if SplitURL {
			for _, str := range global.SplitURL(CQCodeUnescapeText(s[j:i])) {
				r = append(r, message.NewText(str))
			}
		} else {
			r = append(r, message.NewText(CQCodeUnescapeText(s[j:i])))
		}
	}
	return
}

// ConvertObjectMessage 将消息JSON对象转为消息元素数组
func (bot *CQBot) ConvertObjectMessage(m gjson.Result, isGroup bool) (r []message.IMessageElement) {
	convertElem := func(e gjson.Result) {
		t := e.Get("type").Str
		if t == "reply" && isGroup {
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
				var org MSG
				sender, senderErr := strconv.ParseInt(e.Get("data.qq").String(), 10, 64)
				if senderErr != nil && err != nil {
					log.Warnf("警告: 自定义 Reply 元素中必须包含 Uin 或 id")
					break
				}
				msgTime, timeErr := strconv.ParseInt(e.Get("data.time").String(), 10, 64)
				if timeErr != nil {
					msgTime = time.Now().Unix()
				}
				messageSeq, seqErr := strconv.ParseInt(e.Get("data.seq").String(), 10, 64)
				if err == nil {
					org = bot.GetMessage(int32(mid))
				}
				if org != nil {
					elem = &message.ReplyElement{
						ReplySeq: org["message-id"].(int32),
						Sender:   org["sender"].(message.Sender).Uin,
						Time:     org["time"].(int32),
						Elements: bot.ConvertStringMessage(customText, isGroup),
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
						Elements: bot.ConvertStringMessage(customText, isGroup),
					}
				}
				r = append([]message.IMessageElement{elem}, r...)
			case err == nil:
				org := bot.GetMessage(int32(mid))
				if org != nil {
					r = append([]message.IMessageElement{
						&message.ReplyElement{
							ReplySeq: org["message-id"].(int32),
							Sender:   org["sender"].(message.Sender).Uin,
							Time:     org["time"].(int32),
							Elements: bot.ConvertStringMessage(org["message"].(string), isGroup),
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
		d := make(map[string]string)
		e.Get("data").ForEach(func(key, value gjson.Result) bool {
			d[key.Str] = value.String()
			return true
		})
		elem, err := bot.ToElement(t, d, isGroup)
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
		return bot.ConvertStringMessage(m.Str, isGroup)
	}
	if m.IsArray() {
		for _, e := range m.Array() {
			convertElem(e)
		}
	}
	if m.IsObject() {
		convertElem(m)
	}
	return
}

// ToElement 将解码后的CQCode转换为Element.
//
// 返回 interface{} 存在三种类型
//
// message.IMessageElement []message.IMessageElement nil
func (bot *CQBot) ToElement(t string, d map[string]string, isGroup bool) (m interface{}, err error) {
	switch t {
	case "text":
		if SplitURL {
			var ret []message.IMessageElement
			for _, text := range global.SplitURL(d["text"]) {
				ret = append(ret, message.NewText(text))
			}
			return ret, nil
		}
		return message.NewText(d["text"]), nil
	case "image":
		img, err := bot.makeImageOrVideoElem(d, false, isGroup)
		if err != nil {
			return nil, err
		}
		tp := d["type"]
		if tp != "show" && tp != "flash" {
			return img, nil
		}
		if i, ok := img.(*LocalImageElement); ok { // 秀图，闪照什么的就直接传了吧
			if isGroup {
				img, err = bot.UploadLocalImageAsGroup(1, i)
			} else {
				img, err = bot.UploadLocalImageAsPrivate(1, i)
			}
			if err != nil {
				return nil, err
			}
		}
		switch tp {
		case "flash":
			if i, ok := img.(*message.GroupImageElement); ok {
				return &message.GroupFlashPicElement{GroupImageElement: *i}, nil
			}
			if i, ok := img.(*message.FriendImageElement); ok {
				return &message.FriendFlashPicElement{FriendImageElement: *i}, nil
			}
		case "show":
			id, _ := strconv.ParseInt(d["id"], 10, 64)
			if id < 40000 || id >= 40006 {
				id = 40000
			}
			if i, ok := img.(*message.GroupImageElement); ok {
				return &message.GroupShowPicElement{GroupImageElement: *i, EffectId: int32(id)}, nil
			}
			return img, nil // 私聊还没做
		}

	case "poke":
		t, _ := strconv.ParseInt(d["qq"], 10, 64)
		return &PokeElement{Target: t}, nil
	case "gift":
		if !isGroup {
			return nil, errors.New("private gift unsupported") // no free private gift
		}
		t, _ := strconv.ParseInt(d["qq"], 10, 64)
		id, _ := strconv.Atoi(d["id"])
		if id < 0 || id >= 14 {
			return nil, errors.New("invalid gift id")
		}
		return &GiftElement{Target: t, GiftID: GiftID[id]}, nil
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
		return &message.VoiceElement{Data: data}, nil
	case "record":
		f := d["file"]
		data, err := global.FindFile(f, d["cache"], global.VoicePath)
		if err == global.ErrSyntax {
			data, err = global.FindFile(f, d["cache"], global.VoicePathOld)
		}
		if err != nil {
			return nil, err
		}
		if !global.IsAMRorSILK(data) {
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
		return message.NewFace(int32(id)), nil
	case "at":
		qq := d["qq"]
		if qq == "all" {
			return message.AtAll(), nil
		}
		t, _ := strconv.ParseInt(qq, 10, 64)
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
			aid := strconv.FormatInt(info.Get("track_info.album.id").Int(), 10)
			name := info.Get("track_info.name").Str
			mid := info.Get("track_info.mid").Str
			albumMid := info.Get("track_info.album.mid").Str
			pinfo, _ := global.GetBytes("http://u.y.qq.com/cgi-bin/musicu.fcg?g_tk=2034008533&uin=0&format=json&data={\"comm\":{\"ct\":23,\"cv\":0},\"url_mid\":{\"module\":\"vkey.GetVkeyServer\",\"method\":\"CgiGetVkey\",\"param\":{\"guid\":\"4311206557\",\"songmid\":[\"" + mid + "\"],\"songtype\":[0],\"uin\":\"0\",\"loginflag\":1,\"platform\":\"23\"}}}&_=1599039471576")
			jumpURL := "https://i.y.qq.com/v8/playsong.html?platform=11&appshare=android_qq&appversion=10030010&hosteuin=oKnlNenz7i-s7c**&songmid=" + mid + "&type=0&appsongtype=1&_wv=1&source=qq&ADTAG=qfshare"
			purl := gjson.ParseBytes(pinfo).Get("url_mid.data.midurlinfo.0.purl").Str
			preview := "http://y.gtimg.cn/music/photo_new/T002R180x180M000" + albumMid + ".jpg"
			if len(aid) < 2 {
				return nil, errors.New("song error")
			}
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
				subtype := map[string]int{
					"qq":    message.QQMusic,
					"163":   message.CloudMusic,
					"migu":  message.MiguMusic,
					"kugou": message.KugouMusic,
					"kuwo":  message.KuwoMusic,
				}
				musicType := 0
				if tp, ok := subtype[d["subtype"]]; ok {
					musicType = tp
				}
				return &message.MusicShareElement{
					MusicType:  musicType,
					Title:      d["title"],
					Summary:    d["content"],
					Url:        d["url"],
					PictureUrl: d["image"],
					MusicUrl:   d["purl"],
				}, nil
			}
			xml := fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="2" templateID="1" action="web" brief="[分享] %s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="2"><audio cover="%s" src="%s"/><title>%s</title><summary>%s</summary></item><source name="音乐" icon="https://i.gtimg.cn/open/app_icon/01/07/98/56/1101079856_100_m.png" url="http://web.p.qq.com/qqmpmobile/aio/app.html?id=1101079856" action="app" a_actionData="com.tencent.qqmusic" i_actionData="tencent1101079856://" appid="1101079856" /></msg>`,
				XMLEscape(d["title"]), d["url"], d["image"], d["audio"], XMLEscape(d["title"]), XMLEscape(d["content"]))
			return &message.ServiceElement{
				Id:      60,
				Content: xml,
				SubType: "music",
			}, nil
		}
		return nil, errors.New("unsupported music type: " + d["type"])
	case "xml":
		resID := d["resid"]
		template := CQCodeEscapeValue(d["data"])
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
		minWidth, _ := strconv.ParseInt(d["minwidth"], 10, 64)
		if minWidth == 0 {
			minWidth = 200
		}
		minHeight, _ := strconv.ParseInt(d["minheight"], 10, 64)
		if minHeight == 0 {
			minHeight = 200
		}
		maxWidth, _ := strconv.ParseInt(d["maxwidth"], 10, 64)
		if maxWidth == 0 {
			maxWidth = 500
		}
		maxHeight, _ := strconv.ParseInt(d["maxheight"], 10, 64)
		if maxHeight == 0 {
			maxHeight = 1000
		}
		img, err := bot.makeImageOrVideoElem(d, false, isGroup)
		if err != nil {
			return nil, errors.New("send cardimage faild")
		}
		return bot.makeShowPic(img, source, brief, icon, minWidth, minHeight, maxWidth, maxHeight, isGroup)
	case "video":
		cache := d["cache"]
		if cache == "" {
			cache = "1"
		}
		file, err := bot.makeImageOrVideoElem(d, true, isGroup)
		if err != nil {
			return nil, err
		}
		v := file.(*LocalVideoElement)
		if v.File == "" {
			return v, nil
		}
		var data []byte
		if cover, ok := d["cover"]; ok {
			data, _ = global.FindFile(cover, cache, global.ImagePath)
		} else {
			_ = global.ExtractCover(v.File, v.File+".jpg")
			data, _ = ioutil.ReadFile(v.File + ".jpg")
		}
		v.thumb = bytes.NewReader(data)
		video, _ := os.Open(v.File)
		defer video.Close()
		_, err = video.Seek(4, io.SeekStart)
		if err != nil {
			return nil, err
		}
		header := make([]byte, 4)
		_, err = video.Read(header)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(header, []byte{0x66, 0x74, 0x79, 0x70}) { // check file header ftyp
			_, _ = video.Seek(0, io.SeekStart)
			hash, _ := utils.ComputeMd5AndLength(video)
			cacheFile := path.Join(global.CachePath, hex.EncodeToString(hash)+".mp4")
			if global.PathExists(cacheFile) && cache == "1" {
				goto ok
			}
			err = global.EncodeMP4(v.File, cacheFile)
			if err != nil {
				return nil, err
			}
		ok:
			v.File = cacheFile
		}
		return v, nil
	default:
		return nil, errors.New("unsupported cq code: " + t)
	}
	return nil, nil
}

// XMLEscape 将字符串c转义为XML字符串
func XMLEscape(c string) string {
	buf := new(bytes.Buffer)
	_ = xml2.EscapeText(buf, []byte(c))
	return buf.String()
}

/*CQCodeEscapeText 将字符串raw中部分字符转义

& -> &amp;

[ -> &#91;

] -> &#93;

*/
func CQCodeEscapeText(raw string) string {
	ret := raw
	ret = strings.ReplaceAll(ret, "&", "&amp;")
	ret = strings.ReplaceAll(ret, "[", "&#91;")
	ret = strings.ReplaceAll(ret, "]", "&#93;")
	return ret
}

/*CQCodeEscapeValue 将字符串value中部分字符转义

, -> &#44;

& -> &amp;

[ -> &#91;

] -> &#93;

*/
func CQCodeEscapeValue(value string) string {
	ret := CQCodeEscapeText(value)
	ret = strings.ReplaceAll(ret, ",", "&#44;")
	return ret
}

/*CQCodeUnescapeText 将字符串content中部分字符反转义

&amp; -> &

&#91; -> [

&#93; -> ]

*/
func CQCodeUnescapeText(content string) string {
	ret := content
	ret = strings.ReplaceAll(ret, "&#91;", "[")
	ret = strings.ReplaceAll(ret, "&#93;", "]")
	ret = strings.ReplaceAll(ret, "&amp;", "&")
	return ret
}

/*CQCodeUnescapeValue 将字符串content中部分字符反转义

&#44; -> ,

&amp; -> &

&#91; -> [

&#93; -> ]

*/
func CQCodeUnescapeValue(content string) string {
	ret := strings.ReplaceAll(content, "&#44;", ",")
	ret = CQCodeUnescapeText(ret)
	return ret
}

// makeImageOrVideoElem 图片 elem 生成器，单独拎出来，用于公用
func (bot *CQBot) makeImageOrVideoElem(d map[string]string, video, group bool) (message.IMessageElement, error) {
	f := d["file"]
	if strings.HasPrefix(f, "http") || strings.HasPrefix(f, "https") {
		cache := d["cache"]
		c := d["c"]
		if cache == "" {
			cache = "1"
		}
		hash := md5.Sum([]byte(f))
		cacheFile := path.Join(global.CachePath, hex.EncodeToString(hash[:])+".cache")
		maxSize := func() int64 {
			if video {
				return maxVideoSize
			}
			return maxImageSize
		}()
		thread, _ := strconv.Atoi(c)
		if global.PathExists(cacheFile) && cache == "1" {
			goto hasCacheFile
		}
		if global.PathExists(cacheFile) {
			_ = os.Remove(cacheFile)
		}
		if err := global.DownloadFileMultiThreading(f, cacheFile, maxSize, thread, nil); err != nil {
			return nil, err
		}
	hasCacheFile:
		if video {
			return &LocalVideoElement{File: cacheFile}, nil
		}
		return &LocalImageElement{File: cacheFile}, nil
	}
	if strings.HasPrefix(f, "file") {
		fu, err := url.Parse(f)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(fu.Path, "/") && runtime.GOOS == `windows` {
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
		return &LocalImageElement{File: fu.Path}, nil
	}
	rawPath := path.Join(global.ImagePath, f)
	if video {
		rawPath = path.Join(global.VideoPath, f)
		if !global.PathExists(rawPath) {
			return nil, errors.New("invalid video")
		}
		if path.Ext(rawPath) == ".video" {
			b, _ := ioutil.ReadFile(rawPath)
			r := binary.NewReader(b)
			return &LocalVideoElement{ShortVideoElement: message.ShortVideoElement{ // todo 检查缓存是否有效
				Md5:       r.ReadBytes(16),
				ThumbMd5:  r.ReadBytes(16),
				Size:      r.ReadInt32(),
				ThumbSize: r.ReadInt32(),
				Name:      r.ReadString(),
				Uuid:      r.ReadAvailable(),
			}}, nil
		}
		return &LocalVideoElement{File: rawPath}, nil
	}
	if strings.HasPrefix(f, "base64") {
		b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(f, "base64://", ""))
		if err != nil {
			return nil, err
		}
		return &LocalImageElement{Stream: bytes.NewReader(b)}, nil
	}
	if !global.PathExists(rawPath) && global.PathExists(path.Join(global.ImagePathOld, f)) {
		rawPath = path.Join(global.ImagePathOld, f)
	}
	if !global.PathExists(rawPath) && global.PathExists(rawPath+".cqimg") {
		rawPath += ".cqimg"
	}
	if !global.PathExists(rawPath) && d["url"] != "" {
		return bot.makeImageOrVideoElem(map[string]string{"file": d["url"]}, false, group)
	}
	if global.PathExists(rawPath) {
		file, err := os.Open(rawPath)
		if err != nil {
			return nil, err
		}
		if path.Ext(rawPath) != ".image" && path.Ext(rawPath) != ".cqimg" {
			return &LocalImageElement{Stream: file}, nil
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		if len(b) < 20 {
			return nil, errors.New("invalid local file")
		}
		var (
			size int32
			hash []byte
			url  string
		)
		if path.Ext(rawPath) == ".cqimg" {
			for _, line := range strings.Split(global.ReadAllText(rawPath), "\n") {
				kv := strings.SplitN(line, "=", 2)
				switch kv[0] {
				case "md5":
					hash, _ = hex.DecodeString(strings.ReplaceAll(kv[1], "\r", ""))
				case "size":
					t, _ := strconv.Atoi(strings.ReplaceAll(kv[1], "\r", ""))
					size = int32(t)
				}
			}
		} else {
			r := binary.NewReader(b)
			hash = r.ReadBytes(16)
			size = r.ReadInt32()
			r.ReadString()
			url = r.ReadString()
		}
		if size == 0 {
			if url != "" {
				return bot.makeImageOrVideoElem(map[string]string{"file": url}, false, group)
			}
			return nil, errors.New("img size is 0")
		}
		if len(hash) != 16 {
			return nil, errors.New("invalid hash")
		}
		var rsp message.IMessageElement
		if group {
			rsp, err = bot.Client.QueryGroupImage(int64(rand.Uint32()), hash, size)
			goto ok
		}
		rsp, err = bot.Client.QueryFriendImage(int64(rand.Uint32()), hash, size)
	ok:
		if err != nil {
			if url != "" {
				return bot.makeImageOrVideoElem(map[string]string{"file": url}, false, group)
			}
			return nil, err
		}
		return rsp, nil
	}
	return nil, errors.New("invalid image")
}

// makeShowPic 一种xml 方式发送的群消息图片
func (bot *CQBot) makeShowPic(elem message.IMessageElement, source string, brief string, icon string, minWidth int64, minHeight int64, maxWidth int64, maxHeight int64, group bool) ([]message.IMessageElement, error) {
	xml := ""
	var suf message.IMessageElement
	if brief == "" {
		brief = "&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！"
	}
	if i, ok := elem.(*LocalImageElement); ok {
		if !group {
			gm, err := bot.UploadLocalImageAsPrivate(1, i)
			if err != nil {
				log.Warnf("警告: 好友消息 %v 消息图片上传失败: %v", 1, err)
				return nil, err
			}
			suf = gm
			xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="%s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, brief, "", gm.Md5, gm.Md5, len(i.Data), "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		} else {
			gm, err := bot.UploadLocalImageAsGroup(1, i)
			if err != nil {
				log.Warnf("警告: 群 %v 消息图片上传失败: %v", 1, err)
				return nil, err
			}
			suf = gm
			xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="%s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, brief, "", gm.Md5, gm.Md5, len(i.Data), "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		}
	}

	if i, ok := elem.(*message.GroupImageElement); ok {
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="%s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, brief, "", i.Md5, i.Md5, 0, "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		suf = i
	}
	if i, ok := elem.(*message.FriendImageElement); ok {
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="%s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, brief, "", i.Md5, i.Md5, 0, "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		suf = i
	}
	if xml != "" {
		// log.Warn(xml)
		ret := []message.IMessageElement{suf}
		ret = append(ret, message.NewRichXml(xml, 5))
		return ret, nil
	}
	return nil, errors.New("生成xml图片消息失败")
}
