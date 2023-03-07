// Package msg 提供了go-cqhttp消息中间表示，CQ码处理等等
package msg

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/Mrs4s/MiraiGo/binary"
)

// @@@ CQ码转义处理 @@@

// EscapeText 将字符串raw中部分字符转义
//
//   - & -> &amp;
//   - [ -> &#91;
//   - ] -> &#93;
func EscapeText(s string) string {
	count := strings.Count(s, "&")
	count += strings.Count(s, "[")
	count += strings.Count(s, "]")
	if count == 0 {
		return s
	}

	// Apply replacements to buffer.
	var b strings.Builder
	b.Grow(len(s) + count*4)
	start := 0
	for i := 0; i < count; i++ {
		j := start
		for index, r := range s[start:] {
			if r == '&' || r == '[' || r == ']' {
				j += index
				break
			}
		}
		b.WriteString(s[start:j])
		switch s[j] {
		case '&':
			b.WriteString("&amp;")
		case '[':
			b.WriteString("&#91;")
		case ']':
			b.WriteString("&#93;")
		}
		start = j + 1
	}
	b.WriteString(s[start:])
	return b.String()
}

// EscapeValue 将字符串value中部分字符转义
//
//   - , -> &#44;
//   - & -> &amp;
//   - [ -> &#91;
//   - ] -> &#93;
func EscapeValue(value string) string {
	ret := EscapeText(value)
	return strings.ReplaceAll(ret, ",", "&#44;")
}

// UnescapeText 将字符串content中部分字符反转义
//
//   - &amp; -> &
//   - &#91; -> [
//   - &#93; -> ]
func UnescapeText(content string) string {
	ret := content
	ret = strings.ReplaceAll(ret, "&#91;", "[")
	ret = strings.ReplaceAll(ret, "&#93;", "]")
	ret = strings.ReplaceAll(ret, "&amp;", "&")
	return ret
}

// UnescapeValue 将字符串content中部分字符反转义
//
//   - &#44; -> ,
//   - &amp; -> &
//   - &#91; -> [
//   - &#93; -> ]
func UnescapeValue(content string) string {
	ret := strings.ReplaceAll(content, "&#44;", ",")
	return UnescapeText(ret)
}

// @@@ 消息中间表示 @@@

// Pair key value pair
type Pair struct {
	K string
	V string
}

// Element single message
type Element struct {
	Type string
	Data []Pair
}

// Get 获取指定值
func (e *Element) Get(k string) string {
	for _, datum := range e.Data {
		if datum.K == k {
			return datum.V
		}
	}
	return ""
}

// CQCode convert element to cqcode
func (e *Element) CQCode() string {
	buf := strings.Builder{}
	e.WriteCQCodeTo(&buf)
	return buf.String()
}

// WriteCQCodeTo write element's cqcode into sb
func (e *Element) WriteCQCodeTo(sb *strings.Builder) {
	if e.Type == "text" {
		sb.WriteString(EscapeText(e.Data[0].V)) // must be {"text": value}
		return
	}
	sb.WriteString("[CQ:")
	sb.WriteString(e.Type)
	for _, data := range e.Data {
		sb.WriteByte(',')
		sb.WriteString(data.K)
		sb.WriteByte('=')
		sb.WriteString(EscapeValue(data.V))
	}
	sb.WriteByte(']')
}

// MarshalJSON see encoding/json.Marshaler
func (e *Element) MarshalJSON() ([]byte, error) {
	return binary.NewWriterF(func(w *binary.Writer) {
		buf := (*bytes.Buffer)(w)
		// fmt.Fprintf(buf, `{"type":"%s","data":{`, e.Type)
		buf.WriteString(`{"type":"`)
		buf.WriteString(e.Type)
		buf.WriteString(`","data":{`)
		for i, data := range e.Data {
			if i != 0 {
				buf.WriteByte(',')
			}
			// fmt.Fprintf(buf, `"%s":%q`, data.K, data.V)
			buf.WriteByte('"')
			buf.WriteString(data.K)
			buf.WriteString(`":`)
			buf.WriteString(QuoteJSON(data.V))
		}
		buf.WriteString(`}}`)
	}), nil
}

const hex = "0123456789abcdef"

// QuoteJSON 按JSON转义为字符加上双引号
func QuoteJSON(s string) string {
	i, j := 0, 0
	var b strings.Builder
	b.WriteByte('"')
	for j < len(s) {
		c := s[j]

		if c >= 0x20 && c <= 0x7f && c != '\\' && c != '"' {
			// fast path: most of the time, printable ascii characters are used
			j++
			continue
		}

		switch c {
		case '\\', '"', '\n', '\r', '\t':
			b.WriteString(s[i:j])
			b.WriteByte('\\')
			switch c {
			case '\n':
				c = 'n'
			case '\r':
				c = 'r'
			case '\t':
				c = 't'
			}
			b.WriteByte(c)
			j++
			i = j
			continue

		case '<', '>', '&':
			b.WriteString(s[i:j])
			b.WriteString(`\u00`)
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xF])
			j++
			i = j
			continue
		}

		// This encodes bytes < 0x20 except for \t, \n and \r.
		if c < 0x20 {
			b.WriteString(s[i:j])
			b.WriteString(`\u00`)
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xF])
			j++
			i = j
			continue
		}

		r, size := utf8.DecodeRuneInString(s[j:])

		if r == utf8.RuneError && size == 1 {
			b.WriteString(s[i:j])
			b.WriteString(`\ufffd`)
			j += size
			i = j
			continue
		}

		switch r {
		case '\u2028', '\u2029':
			// U+2028 is LINE SEPARATOR.
			// U+2029 is PARAGRAPH SEPARATOR.
			// They are both technically valid characters in JSON strings,
			// but don't work in JSONP, which has to be evaluated as JavaScript,
			// and can lead to security holes there. It is valid JSON to
			// escape them, so we do so unconditionally.
			// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
			b.WriteString(s[i:j])
			b.WriteString(`\u202`)
			b.WriteByte(hex[r&0xF])
			j += size
			i = j
			continue
		}

		j += size
	}

	b.WriteString(s[i:])
	b.WriteByte('"')
	return b.String()
}
