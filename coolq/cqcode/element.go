package cqcode

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/utils"
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
func (e *Element) CQCode() string {
	buf := strings.Builder{}
	e.WriteCQCodeTo(&buf)
	return buf.String()
}

// WriteCQCodeTo convert to cqcode
//    only called by toStringMessage
func (e *Element) WriteCQCodeTo(w io.Writer) {
	if e.Type == "text" {
		_, _ = w.Write(utils.S2B(EscapeText(e.Data[0].V))) // must be {"text": value}
		return
	}
	_, _ = w.Write([]byte("[CQ:"))
	_, _ = w.Write(utils.S2B(e.Type))
	for _, data := range e.Data {
		_, _ = w.Write([]byte{','})
		_, _ = w.Write(utils.S2B(data.K))
		_, _ = w.Write([]byte{'='})
		_, _ = w.Write(utils.S2B(EscapeValue(data.V)))
	}
	_, _ = w.Write([]byte{']'})
	return
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
