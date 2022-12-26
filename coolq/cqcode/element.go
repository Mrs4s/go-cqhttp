package cqcode

import (
	"bytes"
	"strings"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/go-cqhttp/global"
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
			buf.WriteString(global.Quote(data.V))
		}
		buf.WriteString(`}}`)
	}), nil
}
