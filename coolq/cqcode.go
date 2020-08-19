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

func ToArrayMessage(e []message.IMessageElement, code int64, raw ...bool) (r []MSG) {
	ur := false
	if len(raw) != 0 {
		ur = raw[0]
	}
	for _, elem := range e {
		m := MSG{}
		switch o := elem.(type) {
		case *message.TextElement:
			m = MSG{
				"type": "text",
				"data": map[string]string{"text": o.Content},
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
		case *message.ReplyElement:
			m = MSG{
				"type": "reply",
				"data": map[string]string{"id": fmt.Sprint(ToGlobalId(code, o.ReplySeq))},
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
		case *message.ReplyElement:
			r += fmt.Sprintf("[CQ:reply,id=%d]", ToGlobalId(code, o.ReplySeq))
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
			log.Warnf("转换CQ码到MiraiGo Element时出现错误: %v 将原样发送.", err)
			r = append(r, message.NewText(code))
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
			}
			if size == 0 {
				return nil, errors.New("img size is 0")
			}
			if len(hash) != 16 {
				return nil, errors.New("invalid hash")
			}
			if group {
				rsp, err := bot.Client.QueryGroupImage(1, hash, size)
				if err != nil {
					return nil, err
				}
				return rsp, nil
			}
			rsp, err := bot.Client.QueryFriendImage(1, hash, size)
			if err != nil {
				return nil, err
			}
			return rsp, nil
		}
		return nil, errors.New("invalid image")
	case "record":
		if !group {
			return nil, errors.New("private voice unsupported now")
		}
		f := d["file"]
		var data []byte
		if strings.HasPrefix(f, "http") || strings.HasPrefix(f, "https") {
			b, err := global.GetBytes(f)
			if err != nil {
				return nil, err
			}
			data = b
		}
		if strings.HasPrefix(f, "base64") {
			b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(f, "base64://", ""))
			if err != nil {
				return nil, err
			}
			data = b
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
			data = b
		}
		if global.PathExists(path.Join(global.VOICE_PATH, f)) {
			b, err := ioutil.ReadFile(path.Join(global.VOICE_PATH, f))
			if err != nil {
				return nil, err
			}
			data = b
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
