package coolq

import (
	"encoding/base64"
	"errors"
	"fmt"
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

func ToStringMessage(e []message.IMessageElement, code int64, raw ...bool) (r string) {
	ur := false
	if len(raw) != 0 {
		ur = raw[0]
	}
	for _, elem := range e {
		switch o := elem.(type) {
		case *message.TextElement:
			r += o.Content
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
		case *message.ImageElement:
			if ur {
				r += fmt.Sprintf(`[CQ:image,file=%s]`, o.Filename)
			} else {
				r += fmt.Sprintf(`[CQ:image,file=%s,url=%s]`, o.Filename, o.Url)
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
			r = append(r, message.NewText(text))
		}
		code := m[idx[0]:idx[1]]
		si = idx[1]
		t := typeReg.FindAllStringSubmatch(code, -1)[0][1]
		ps := paramReg.FindAllStringSubmatch(code, -1)
		d := make(map[string]string)
		for _, p := range ps {
			d[p[1]] = p[2]
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
			log.Warnf("转换CQ码到MiraiGo Element时出现错误: %v 将忽略本段CQ码.", err)
			continue
		}
		r = append(r, elem)
	}
	if si != len(m) {
		r = append(r, message.NewText(m[si:]))
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
			b, err := global.GetBytes(f)
			if err != nil {
				return nil, err
			}
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
		if global.PathExists(path.Join(global.IMAGE_PATH, f)) {
			b, err := ioutil.ReadFile(path.Join(global.IMAGE_PATH, f))
			if err != nil {
				return nil, err
			}
			if len(b) < 20 {
				return nil, errors.New("invalid local file")
			}
			return message.NewImage(b), nil
		}
		return nil, errors.New("invalid image")
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
