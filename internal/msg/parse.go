package msg

import (
	"github.com/tidwall/gjson"
)

// ParseObject 将消息JSON对象转为消息元素数组
func ParseObject(m gjson.Result) (r []Element) {
	convert := func(e gjson.Result) {
		var elem Element
		elem.Type = e.Get("type").Str
		e.Get("data").ForEach(func(key, value gjson.Result) bool {
			pair := Pair{K: key.Str, V: value.String()}
			elem.Data = append(elem.Data, pair)
			return true
		})
		r = append(r, elem)
	}

	if m.IsArray() {
		m.ForEach(func(_, e gjson.Result) bool {
			convert(e)
			return true
		})
	}
	if m.IsObject() {
		convert(m)
	}
	return
}

func text(txt string) Element {
	return Element{
		Type: "text",
		Data: []Pair{
			{
				K: "text",
				V: txt,
			},
		},
	}
}

// ParseString 将字符串(CQ码)转为消息元素数组
func ParseString(raw string) (r []Element) {
	var elem Element
	for raw != "" {
		i := 0
		for i < len(raw) && !(raw[i] == '[' && i+4 < len(raw) && raw[i:i+4] == "[CQ:") {
			i++
		}
		if i > 0 {
			r = append(r, text(UnescapeText(raw[:i])))
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
		elem.Type = raw[:i]
		elem.Data = nil // reset data
		raw = raw[i:]
		i = 0
		for {
			if raw[0] == ']' {
				r = append(r, elem)
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
			key := raw[:i]
			raw = raw[i+1:] // skip "="
			i = 0
			for i < len(raw) && raw[i] != ',' && raw[i] != ']' {
				i++
			}

			if i+1 > len(raw) {
				return
			}
			elem.Data = append(elem.Data, Pair{
				K: key,
				V: UnescapeValue(raw[:i]),
			})
			raw = raw[i:]
			i = 0
		}
	}
	return
}
