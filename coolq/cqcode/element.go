package cqcode

import (
	"bytes"
	"strconv"
	"strings"

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

// CQCode convert to cqcode
//    only called by toStringMessage
func (e *Element) CQCode(sb strings.Builder) string {
	if e.Type == "text" {
		return EscapeText(e.Data[0].V) // must be {"text": value}
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
	return sb.String()
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
			buf.WriteString(strconv.Quote(data.V))
		}
		buf.WriteString(`}}`)
	}), nil
}
