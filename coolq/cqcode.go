package coolq

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	xml2 "encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/global"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

/*
var matchReg = regexp.MustCompile(`\[CQ:\w+?.*?]`)
var typeReg = regexp.MustCompile(`\[CQ:(\w+)`)
var paramReg = regexp.MustCompile(`,([\w\-.]+?)=([^,\]]+)`)
*/

var IgnoreInvalidCQCode = false

type PokeElement struct {
	Target int64
}

type GiftElement struct {
	Target int64
	GiftId message.GroupGift
}

type MusicElement struct {
	Title      string
	Summary    string
	Url        string
	PictureUrl string
	MusicUrl   string
}

type QQMusicElement struct {
	MusicElement
}

type CloudMusicElement struct {
	MusicElement
}

type MiguMusicElement struct {
	MusicElement
}

func (e *GiftElement) Type() message.ElementType {
	return message.At
}

func (e *MusicElement) Type() message.ElementType {
	return message.Service
}

var GiftId = [...]message.GroupGift{
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

func (e *PokeElement) Type() message.ElementType {
	return message.At
}

func ToArrayMessage(e []message.IMessageElement, code int64, raw ...bool) (r []MSG) {
	ur := false
	if len(raw) != 0 {
		ur = raw[0]
	}
	m := &message.SendingMessage{Elements: e}
	reply := m.FirstOrNil(func(e message.IMessageElement) bool {
		_, ok := e.(*message.ReplyElement)
		return ok
	})
	if reply != nil {
		r = append(r, MSG{
			"type": "reply",
			"data": map[string]string{"id": fmt.Sprint(ToGlobalId(code, reply.(*message.ReplyElement).ReplySeq))},
		})
	}
	for _, elem := range e {
		m := MSG{}
		switch o := elem.(type) {
		case *message.TextElement:
			m = MSG{
				"type": "text",
				"data": map[string]string{"text": o.Content},
			}
		case *message.LightAppElement:
			//m = MSG{
			//	"type": "text",
			//	"data": map[string]string{"text": o.Content},
			//}
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
					"data": map[string]string{"file": hex.EncodeToString(o.Md5) + ".image", "url": CQCodeEscapeText(o.Url)},
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
					"data": map[string]string{"file": hex.EncodeToString(o.Md5) + ".image", "url": CQCodeEscapeText(o.Url)},
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
		r = append(r, m)
	}
	return
}

func ToStringMessage(e []message.IMessageElement, code int64, raw ...bool) (r string) {
	ur := false
	if len(raw) != 0 {
		ur = raw[0]
	}
	// 方便
	m := &message.SendingMessage{Elements: e}
	reply := m.FirstOrNil(func(e message.IMessageElement) bool {
		_, ok := e.(*message.ReplyElement)
		return ok
	})
	if reply != nil {
		r += fmt.Sprintf("[CQ:reply,id=%d]", ToGlobalId(code, reply.(*message.ReplyElement).ReplySeq))
	}
	for _, elem := range e {
		switch o := elem.(type) {
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
				r += fmt.Sprintf("[CQ:image,file=%s,url=%s]", hex.EncodeToString(o.Md5)+".image", CQCodeEscapeText(o.Url))
			}
		case *message.FriendImageElement:
			if ur {
				r += fmt.Sprintf("[CQ:image,file=%s]", hex.EncodeToString(o.Md5)+".image")
			} else {
				r += fmt.Sprintf("[CQ:image,file=%s,url=%s]", hex.EncodeToString(o.Md5)+".image", CQCodeEscapeText(o.Url))
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
			//r += CQCodeEscapeText(o.Content)
		}
	}
	return
}

func (bot *CQBot) ConvertStringMessage(msg string, group bool) (r []message.IMessageElement) {
	index := 0
	stat := 0
	rMsg := []rune(msg)
	var tempText, cqCode []rune
	hasNext := func() bool {
		return index < len(rMsg)
	}
	next := func() rune {
		r := rMsg[index]
		index++
		return r
	}
	move := func(steps int) {
		index += steps
	}
	peekN := func(count int) string {
		lastIdx := int(math.Min(float64(index+count), float64(len(rMsg))))
		return string(rMsg[index:lastIdx])
	}
	isCQCodeBegin := func(r rune) bool {
		return r == '[' && peekN(3) == "CQ:"
	}
	saveTempText := func() {
		if len(tempText) != 0 {
			r = append(r, message.NewText(CQCodeUnescapeValue(string(tempText))))
		}
		tempText = []rune{}
		cqCode = []rune{}
	}
	saveCQCode := func() {
		defer func() {
			cqCode = []rune{}
			tempText = []rune{}
		}()
		s := strings.SplitN(string(cqCode), ",", -1)
		if len(s) == 0 {
			return
		}
		t := s[0]
		params := make(map[string]string)
		for i := 1; i < len(s); i++ {
			p := s[i]
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			data := strings.SplitN(p, "=", 2)
			if len(data) == 2 {
				params[data[0]] = CQCodeUnescapeValue(data[1])
			} else {
				params[p] = ""
			}
		}
		if t == "reply" { // reply 特殊处理
			if len(r) > 0 {
				if _, ok := r[0].(*message.ReplyElement); ok {
					log.Warnf("警告: 一条信息只能包含一个 Reply 元素.")
					return
				}
			}
			mid, err := strconv.Atoi(params["id"])
			if err == nil {
				org := bot.GetMessage(int32(mid))
				if org != nil {
					r = append([]message.IMessageElement{
						&message.ReplyElement{
							ReplySeq: org["message-id"].(int32),
							Sender:   org["sender"].(message.Sender).Uin,
							Time:     org["time"].(int32),
							Elements: bot.ConvertStringMessage(org["message"].(string), group),
						},
					}, r...)
					return
				}
			}
		}
		elem, err := bot.ToElement(t, params, group)
		if err != nil {
			org := "[" + string(cqCode) + "]"
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
	for hasNext() {
		ch := next()
		switch stat {
		case 0:
			if isCQCodeBegin(ch) {
				saveTempText()
				tempText = append(tempText, []rune("[CQ:")...)
				move(3)
				stat = 1
			} else {
				tempText = append(tempText, ch)
			}
		case 1:
			if isCQCodeBegin(ch) {
				move(-1)
				stat = 0
			} else if ch == ']' {
				saveCQCode()
				stat = 0
			} else {
				cqCode = append(cqCode, ch)
				tempText = append(tempText, ch)
			}
		}
	}
	saveTempText()
	return
}

func (bot *CQBot) ConvertObjectMessage(m gjson.Result, group bool) (r []message.IMessageElement) {
	convertElem := func(e gjson.Result) {
		t := e.Get("type").Str
		if t == "reply" && group {
			if len(r) > 0 {
				if _, ok := r[0].(*message.ReplyElement); ok {
					log.Warnf("警告: 一条信息只能包含一个 Reply 元素.")
					return
				}
			}
			mid, err := strconv.Atoi(e.Get("data").Get("id").String())
			if err == nil {
				org := bot.GetMessage(int32(mid))
				if org != nil {
					r = append([]message.IMessageElement{
						&message.ReplyElement{
							ReplySeq: org["message-id"].(int32),
							Sender:   org["sender"].(message.Sender).Uin,
							Time:     org["time"].(int32),
							Elements: bot.ConvertStringMessage(org["message"].(string), group),
						},
					}, r...)
					return
				}
			}
		}
		d := make(map[string]string)
		e.Get("data").ForEach(func(key, value gjson.Result) bool {
			d[key.Str] = value.String()
			return true
		})
		elem, err := bot.ToElement(t, d, group)
		if err != nil {
			log.Warnf("转换CQ码到MiraiGo Element时出现错误: %v 将忽略本段CQ码.", err)
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
		return bot.ConvertStringMessage(m.Str, group)
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
// 返回 interface{} 存在三种类型
// message.IMessageElement []message.IMessageElement nil
func (bot *CQBot) ToElement(t string, d map[string]string, group bool) (m interface{}, err error) {
	switch t {
	case "text":
		return message.NewText(d["text"]), nil
	case "image":
		img, err := bot.makeImageElem(d, group)
		if err != nil {
			return nil, err
		}
		tp := d["type"]
		if tp != "show" && tp != "flash" {
			return img, nil
		}
		if i, ok := img.(*message.ImageElement); ok { // 秀图，闪照什么的就直接传了吧
			if group {
				img, err = bot.Client.UploadGroupImage(1, i.Data)
			} else {
				img, err = bot.Client.UploadPrivateImage(1, i.Data)
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
		if !group {
			return nil, errors.New("private gift unsupported") // no free private gift
		}
		t, _ := strconv.ParseInt(d["qq"], 10, 64)
		id, _ := strconv.Atoi(d["id"])
		if id < 0 || id >= 14 {
			return nil, errors.New("invalid gift id")
		}
		return &GiftElement{Target: t, GiftId: GiftId[id]}, nil
	case "tts":
		defer func() {
			if r := recover(); r != nil {
				m = nil
				err = errors.New("tts 转换失败")
			}
		}()
		data, err := bot.Client.GetTts(d["text"])
		ioutil.WriteFile("tts.silk", data, 777)
		if err != nil {
			return nil, err
		}
		return &message.VoiceElement{Data: data}, nil
	case "record":
		f := d["file"]
		data, err := global.FindFile(f, d["cache"], global.VOICE_PATH)
		if err != nil {
			return nil, err
		}
		if !global.IsAMRorSILK(data) {
			data, err = global.Encoder(data)
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
		return message.NewAt(t), nil
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
			name := info.Get("track_info.name").Str + " - " + info.Get("track_info.singer.0.name").Str
			mid := info.Get("track_info.mid").Str
			albumMid := info.Get("track_info.album.mid").Str
			pinfo, _ := global.GetBytes("http://u.y.qq.com/cgi-bin/musicu.fcg?g_tk=2034008533&uin=0&format=json&data={\"comm\":{\"ct\":23,\"cv\":0},\"url_mid\":{\"module\":\"vkey.GetVkeyServer\",\"method\":\"CgiGetVkey\",\"param\":{\"guid\":\"4311206557\",\"songmid\":[\"" + mid + "\"],\"songtype\":[0],\"uin\":\"0\",\"loginflag\":1,\"platform\":\"23\"}}}&_=1599039471576")
			jumpUrl := "https://i.y.qq.com/v8/playsong.html?platform=11&appshare=android_qq&appversion=10030010&hosteuin=oKnlNenz7i-s7c**&songmid=" + mid + "&type=0&appsongtype=1&_wv=1&source=qq&ADTAG=qfshare"
			purl := gjson.ParseBytes(pinfo).Get("url_mid.data.midurlinfo.0.purl").Str
			preview := "http://y.gtimg.cn/music/photo_new/T002R180x180M000" + albumMid + ".jpg"
			if len(aid) < 2 {
				return nil, errors.New("song error")
			}
			content := "来自go-cqhttp"
			if d["content"] != "" {
				content = d["content"]
			}
			return &QQMusicElement{MusicElement: MusicElement{
				Title:      name,
				Summary:    content,
				Url:        jumpUrl,
				PictureUrl: preview,
				MusicUrl:   purl,
			}}, nil
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
			jumpUrl := "https://y.music.163.com/m/song/" + d["id"]
			musicUrl := "http://music.163.com/song/media/outer/url?id=" + d["id"]
			picUrl := info.Get("album.picUrl").Str
			artistName := ""
			if info.Get("artists.0").Exists() {
				artistName = info.Get("artists.0.name").Str
			}
			return &CloudMusicElement{MusicElement{
				Title:      name,
				Summary:    artistName,
				Url:        jumpUrl,
				PictureUrl: picUrl,
				MusicUrl:   musicUrl,
			}}, nil
		}
		if d["type"] == "custom" {
			if d["subtype"] == "qq" {
				return &QQMusicElement{MusicElement{
					Title:      d["title"],
					Summary:    d["content"],
					Url:        d["url"],
					PictureUrl: d["image"],
					MusicUrl:   d["purl"],
				}}, nil
			}
			if d["subtype"] == "163" {
				return &CloudMusicElement{MusicElement{
					Title:      d["title"],
					Summary:    d["content"],
					Url:        d["url"],
					PictureUrl: d["image"],
					MusicUrl:   d["purl"],
				}}, nil
			}
			if d["subtype"] == "migu" {
				return &MiguMusicElement{MusicElement{
					Title:      d["title"],
					Summary:    d["content"],
					Url:        d["url"],
					PictureUrl: d["image"],
					MusicUrl:   d["purl"],
				}}, nil
			}
			xml := fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="2" templateID="1" action="web" brief="[分享] %s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="2"><audio cover="%s" src="%s"/><title>%s</title><summary>%s</summary></item><source name="音乐" icon="https://i.gtimg.cn/open/app_icon/01/07/98/56/1101079856_100_m.png" url="http://web.p.qq.com/qqmpmobile/aio/app.html?id=1101079856" action="app" a_actionData="com.tencent.qqmusic" i_actionData="tencent1101079856://" appid="1101079856" /></msg>`,
				XmlEscape(d["title"]), d["url"], d["image"], d["audio"], XmlEscape(d["title"]), XmlEscape(d["content"]))
			return &message.ServiceElement{
				Id:      60,
				Content: xml,
				SubType: "music",
			}, nil
		}
		return nil, errors.New("unsupported music type: " + d["type"])
	case "xml":
		resId := d["resid"]
		template := CQCodeEscapeValue(d["data"])
		i, _ := strconv.ParseInt(resId, 10, 64)
		msg := message.NewRichXml(template, i)
		return msg, nil
	case "json":
		resId := d["resid"]
		i, _ := strconv.ParseInt(resId, 10, 64)
		if i == 0 {
			//默认情况下走小程序通道
			msg := message.NewLightApp(CQCodeUnescapeValue(d["data"]))
			return msg, nil
		}
		//resid不为0的情况下走富文本通道，后续补全透传service Id，此处暂时不处理 TODO
		msg := message.NewRichJson(CQCodeUnescapeValue(d["data"]))
		return msg, nil
	case "cardimage":
		source := d["source"]
		icon := d["icon"]
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
		img, err := bot.makeImageElem(d, group)
		if err != nil {
			return nil, errors.New("send cardimage faild")
		}
		return bot.makeShowPic(img, source, icon, minWidth, minHeight, maxWidth, maxHeight, group)
	default:
		return nil, errors.New("unsupported cq code: " + t)
	}
	return nil, nil
}

func XmlEscape(c string) string {
	buf := new(bytes.Buffer)
	_ = xml2.EscapeText(buf, []byte(c))
	return buf.String()
}

func CQCodeEscapeText(raw string) string {
	ret := raw
	ret = strings.ReplaceAll(ret, "&", "&amp;")
	ret = strings.ReplaceAll(ret, "[", "&#91;")
	ret = strings.ReplaceAll(ret, "]", "&#93;")
	return ret
}

func CQCodeEscapeValue(value string) string {
	ret := CQCodeEscapeText(value)
	ret = strings.ReplaceAll(ret, ",", "&#44;")
	return ret
}

func CQCodeUnescapeText(content string) string {
	ret := content
	ret = strings.ReplaceAll(ret, "&#91;", "[")
	ret = strings.ReplaceAll(ret, "&#93;", "]")
	ret = strings.ReplaceAll(ret, "&amp;", "&")
	return ret
}

func CQCodeUnescapeValue(content string) string {
	ret := strings.ReplaceAll(content, "&#44;", ",")
	ret = CQCodeUnescapeText(ret)
	return ret
}

// 图片 elem 生成器，单独拎出来，用于公用
func (bot *CQBot) makeImageElem(d map[string]string, group bool) (message.IMessageElement, error) {
	f := d["file"]
	if strings.HasPrefix(f, "http") || strings.HasPrefix(f, "https") {
		cache := d["cache"]
		if cache == "" {
			cache = "1"
		}
		hash := md5.Sum([]byte(f))
		cacheFile := path.Join(global.CACHE_PATH, hex.EncodeToString(hash[:])+".cache")
		if global.PathExists(cacheFile) && cache == "1" {
			b, err := ioutil.ReadFile(cacheFile)
			if err == nil {
				return message.NewImage(b), nil
			}
		}
		b, err := global.GetBytes(f)
		if err != nil {
			return nil, err
		}
		_ = ioutil.WriteFile(cacheFile, b, 0644)
		return message.NewImage(b), nil
	}
	if strings.HasPrefix(f, "base64") {
		b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(f, "base64://", ""))
		if err != nil {
			return nil, err
		}
		return message.NewImage(b), nil
	}
	if strings.HasPrefix(f, "file") {
		fu, err := url.Parse(f)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(fu.Path, "/") && runtime.GOOS == `windows` {
			fu.Path = fu.Path[1:]
		}
		b, err := ioutil.ReadFile(fu.Path)
		if err != nil {
			return nil, err
		}
		return message.NewImage(b), nil
	}
	rawPath := path.Join(global.IMAGE_PATH, f)
	if !global.PathExists(rawPath) && global.PathExists(rawPath+".cqimg") {
		rawPath += ".cqimg"
	}
	if !global.PathExists(rawPath) && d["url"] != "" {
		return bot.makeImageElem(map[string]string{"file": d["url"]}, group)
	}
	if global.PathExists(rawPath) {
		b, err := ioutil.ReadFile(rawPath)
		if err != nil {
			return nil, err
		}
		if path.Ext(rawPath) != ".image" && path.Ext(rawPath) != ".cqimg" {
			return message.NewImage(b), nil
		}
		if len(b) < 20 {
			return nil, errors.New("invalid local file")
		}
		var size int32
		var hash []byte
		var url string
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
				return bot.makeImageElem(map[string]string{"file": url}, group)
			}
			return nil, errors.New("img size is 0")
		}
		if len(hash) != 16 {
			return nil, errors.New("invalid hash")
		}
		if group {
			rsp, err := bot.Client.QueryGroupImage(1, hash, size)
			if err != nil {
				if url != "" {
					return bot.makeImageElem(map[string]string{"file": url}, group)
				}
				return nil, err
			}
			return rsp, nil
		}
		rsp, err := bot.Client.QueryFriendImage(1, hash, size)
		if err != nil {
			if url != "" {
				return bot.makeImageElem(map[string]string{"file": url}, group)
			}
			return nil, err
		}
		return rsp, nil
	}
	return nil, errors.New("invalid image")
}

//makeShowPic 一种xml 方式发送的群消息图片
func (bot *CQBot) makeShowPic(elem message.IMessageElement, source string, icon string, minWidth int64, minHeight int64, maxWidth int64, maxHeight int64, group bool) ([]message.IMessageElement, error) {
	xml := ""
	var suf message.IMessageElement
	if i, ok := elem.(*message.ImageElement); ok {
		if group == false {
			gm, err := bot.Client.UploadPrivateImage(1, i.Data)
			if err != nil {
				log.Warnf("警告: 好友消息 %v 消息图片上传失败: %v", 1, err)
				return nil, err
			}
			suf = gm
			xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", gm.Md5, gm.Md5, len(i.Data), "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		} else {
			gm, err := bot.Client.UploadGroupImage(1, i.Data)
			if err != nil {
				log.Warnf("警告: 群 %v 消息图片上传失败: %v", 1, err)
				return nil, err
			}
			suf = gm
			xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", gm.Md5, gm.Md5, len(i.Data), "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		}
	}

	if i, ok := elem.(*message.GroupImageElement); ok {
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", i.Md5, i.Md5, 0, "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		suf = i
	}
	if i, ok := elem.(*message.FriendImageElement); ok {
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", i.Md5, i.Md5, 0, "", minWidth, minHeight, maxWidth, maxHeight, source, icon)
		suf = i
	}
	if xml != "" {
		//log.Warn(xml)
		ret := []message.IMessageElement{suf}
		ret = append(ret, message.NewRichXml(xml, 5))
		return ret, nil
	}
	return nil, errors.New("生成xml图片消息失败")
}
