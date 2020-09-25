package coolq

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/global"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/url"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var matchReg = regexp.MustCompile(`\[CQ:\w+?.*?]`)
var typeReg = regexp.MustCompile(`\[CQ:(\w+)`)
var paramReg = regexp.MustCompile(`,([\w\-.]+?)=([^,\]]+)`)

var IgnoreInvalidCQCode = false

type PokeElement struct {
	Target int64
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

func (bot *CQBot) ConvertStringMessage(m string, group bool) (r []message.IMessageElement) {
	i := matchReg.FindAllStringSubmatchIndex(m, -1)
	si := 0
	for _, idx := range i {
		if idx[0] > si {
			text := m[si:idx[0]]
			r = append(r, message.NewText(CQCodeUnescapeText(text)))
		}
		code := m[idx[0]:idx[1]]
		si = idx[1]
		t := typeReg.FindAllStringSubmatch(code, -1)[0][1]
		ps := paramReg.FindAllStringSubmatch(code, -1)
		d := make(map[string]string)
		for _, p := range ps {
			d[p[1]] = CQCodeUnescapeValue(p[2])
		}
		if t == "reply" && group {
			if len(r) > 0 {
				if _, ok := r[0].(*message.ReplyElement); ok {
					log.Warnf("警告: 一条信息只能包含一个 Reply 元素.")
					continue
				}
			}
			mid, err := strconv.Atoi(d["id"])
			if err == nil {
				org := bot.GetGroupMessage(int32(mid))
				if org != nil {
					r = append([]message.IMessageElement{
						&message.ReplyElement{
							ReplySeq: org["message-id"].(int32),
							Sender:   org["sender"].(message.Sender).Uin,
							Time:     org["time"].(int32),
							Elements: bot.ConvertStringMessage(org["message"].(string), group),
						},
					}, r...)
					continue
				}
			}
		}
		elem, err := bot.ToElement(t, d, group)
		if err != nil {
			if !IgnoreInvalidCQCode {
				log.Warnf("转换CQ码 %v 到MiraiGo Element时出现错误: %v 将原样发送.", code, err)
				r = append(r, message.NewText(code))
			} else {
				log.Warnf("转换CQ码 %v 到MiraiGo Element时出现错误: %v 将忽略.", code, err)
			}
			continue
		}
		r = append(r, elem)
	}
	if si != len(m) {
		r = append(r, message.NewText(CQCodeUnescapeText(m[si:])))
	}
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
			mid, err := strconv.Atoi(e.Get("data").Get("id").Str)
			if err == nil {
				org := bot.GetGroupMessage(int32(mid))
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
			d[key.Str] = value.Str
			return true
		})
		elem, err := bot.ToElement(t, d, group)
		if err != nil {
			log.Warnf("转换CQ码到MiraiGo Element时出现错误: %v 将忽略本段CQ码.", err)
			return
		}
		r = append(r, elem)
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

func (bot *CQBot) ToElement(t string, d map[string]string, group bool) (message.IMessageElement, error) {
	switch t {
	case "text":
		return message.NewText(d["text"]), nil
	case "image":
		return bot.makeImageElem(t, d, group)
	case "poke":
		if !group {
			return nil, errors.New("todo") // TODO: private poke
		}
		t, _ := strconv.ParseInt(d["qq"], 10, 64)
		return &PokeElement{Target: t}, nil
	case "record":
		if !group {
			return nil, errors.New("private voice unsupported now")
		}
		f := d["file"]
		data, err := global.FindFile(f, d["cache"], global.VOICE_PATH)
		if err != nil {
			return nil, err
		}
		if !global.IsAMRorSILK(data) {
			return nil, errors.New("unsupported voice file format (please use AMR file for now)")
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
			json := fmt.Sprintf("{\"app\": \"com.tencent.structmsg\",\"desc\": \"音乐\",\"meta\": {\"music\": {\"desc\": \"%s\",\"jumpUrl\": \"%s\",\"musicUrl\": \"%s\",\"preview\": \"%s\",\"tag\": \"QQ音乐\",\"title\": \"%s\"}},\"prompt\": \"[分享]%s\",\"ver\": \"0.0.0.1\",\"view\": \"music\"}", content, jumpUrl, purl, preview, name, name)
			return message.NewLightApp(json), nil
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
			json := fmt.Sprintf("{\"app\": \"com.tencent.structmsg\",\"desc\":\"音乐\",\"view\":\"music\",\"prompt\":\"[分享]%s\",\"ver\":\"0.0.0.1\",\"meta\":{ \"music\": { \"desc\": \"%s\", \"jumpUrl\": \"%s\", \"musicUrl\": \"%s\", \"preview\": \"%s\", \"tag\": \"网易云音乐\", \"title\":\"%s\"}}}", name, artistName, jumpUrl, musicUrl, picUrl, name)
			return message.NewLightApp(json), nil
		}
		if d["type"] == "custom" {
			xml := fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="2" templateID="1" action="web" brief="[分享] %s" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="2"><audio cover="%s" src="%s"/><title>%s</title><summary>%s</summary></item><source name="音乐" icon="https://i.gtimg.cn/open/app_icon/01/07/98/56/1101079856_100_m.png" url="http://web.p.qq.com/qqmpmobile/aio/app.html?id=1101079856" action="app" a_actionData="com.tencent.qqmusic" i_actionData="tencent1101079856://" appid="1101079856" /></msg>`,
				d["title"], d["url"], d["image"], d["audio"], d["title"], d["content"])
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
		//println(template)
		i, _ := strconv.ParseInt(resId, 10, 64)
		msg := message.NewRichXml(template, i)
		return msg, nil
	case "json":
		resId := d["resid"]
		i, _ := strconv.ParseInt(resId, 10, 64)
		log.Warnf("json msg=%s", d["data"])
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
		minwidth, _ := strconv.ParseInt(d["minwidth"], 10, 64)
		if minwidth == 0 {
			minwidth = 200
		}
		minheight, _ := strconv.ParseInt(d["minheight"], 10, 64)
		if minheight == 0 {
			minheight = 200
		}
		maxwidth, _ := strconv.ParseInt(d["maxwidth"], 10, 64)
		if maxwidth == 0 {
			maxwidth = 500
		}
		maxheight, _ := strconv.ParseInt(d["maxheight"], 10, 64)
		if maxheight == 0 {
			maxheight = 1000
		}
		img, err := bot.makeImageElem(t, d, group)
		if err != nil {
			return nil, errors.New("send cardimage faild")
		}
		return bot.SendNewPic(img, source, icon, minwidth, minheight, maxwidth, maxheight, group)
	default:
		return nil, errors.New("unsupported cq code: " + t)
	}
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
func (bot *CQBot) makeImageElem(t string, d map[string]string, group bool) (message.IMessageElement, error) {
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
		return bot.ToElement(t, map[string]string{"file": d["url"]}, group)
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
				return bot.ToElement(t, map[string]string{"file": url}, group)
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
					return bot.ToElement(t, map[string]string{"file": url}, group)
				}
				return nil, err
			}
			return rsp, nil
		}
		rsp, err := bot.Client.QueryFriendImage(1, hash, size)
		if err != nil {
			if url != "" {
				return bot.ToElement(t, map[string]string{"file": url}, group)
			}
			return nil, err
		}
		return rsp, nil
	}
	return nil, errors.New("invalid image")
}

//SendNewPic 一种xml 方式发送的群消息图片
func (bot *CQBot) SendNewPic(elem message.IMessageElement, source string, icon string, minwidth int64, minheigt int64, maxwidth int64, maxheight int64, group bool) (*message.ServiceElement, error) {
	var xml string
	xml = ""
	if i, ok := elem.(*message.ImageElement); ok {
		if group == false {
			gm, err := bot.Client.UploadPrivateImage(1, i.Data)
			if err != nil {
				log.Warnf("警告: 好友消息 %v 消息图片上传失败: %v", 1, err)
				return nil, err
			}
			xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", gm.Md5, gm.Md5, len(i.Data), "", minwidth, minheigt, maxwidth, maxheight, source, icon)

		} else {
			gm, err := bot.Client.UploadGroupImage(1, i.Data)
			if err != nil {
				log.Warnf("警告: 群 %v 消息图片上传失败: %v", 1, err)
				return nil, err
			}
			xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", gm.Md5, gm.Md5, len(i.Data), "", minwidth, minheigt, maxwidth, maxheight, source, icon)
		}
	}
	if i, ok := elem.(*message.GroupImageElement); ok {
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", i.Md5, i.Md5, 0, "", minwidth, minheigt, maxwidth, maxheight, source, icon)
	}
	if i, ok := elem.(*message.FriendImageElement); ok {
		xml = fmt.Sprintf(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="5" templateID="12345" action="" brief="&#91;分享&#93;我看到一张很赞的图片，分享给你，快来看！" sourceMsgId="0" url="%s" flag="0" adverSign="0" multiMsgFlag="0"><item layout="0" advertiser_id="0" aid="0"><image uuid="%x" md5="%x" GroupFiledid="0" filesize="%d" local_path="%s" minWidth="%d" minHeight="%d" maxWidth="%d" maxHeight="%d" /></item><source name="%s" icon="%s" action="" appid="-1" /></msg>`, "", i.Md5, i.Md5, 0, "", minwidth, minheigt, maxwidth, maxheight, source, icon)
	}
	if xml != "" {
		log.Warn(xml)
		XmlMsg := message.NewRichXml(xml, 5)
		return XmlMsg, nil
	}
	return nil, errors.New("发送xml图片消息失败")
}
