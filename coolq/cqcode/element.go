package cqcode

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/Mrs4s/MiraiGo/binary"
)

// Element single message
type Element struct {
	Type string
	Data []Pair
}

// Pair key value pair
type Pair struct {
	K string
	V string
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
			writeQuote(buf, data.V)
		}
		buf.WriteString(`}}`)
	}), nil
}

const hex = "0123456789abcdef"

func writeQuote(b *bytes.Buffer, s string) {
	i, j := 0, 0

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
}
