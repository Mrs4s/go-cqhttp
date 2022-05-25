package cqcode

import (
	"fmt"
	"strings"

	"github.com/Mrs4s/go-cqhttp/global"
)

type Element struct {
	Type string
	Data []Pair
}

type Pair struct {
	K string
	V string
}

func (e *Element) CQCode() string {
	if e.Type == "text" {
		return EscapeText(e.Data[0].V) // must be {"text": value}
	}
	var sb strings.Builder
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

func (e *Element) MarshalJSON() ([]byte, error) {
	buf := global.NewBuffer()
	defer global.PutBuffer(buf)

	fmt.Fprintf(buf, `{"type":"%s","data":{`, e.Type)
	for i, data := range e.Data {
		if i != 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(buf, `"%s":%q`, data.K, data.V)
	}
	buf.WriteString(`}}`)

	return append([]byte(nil), buf.Bytes()...), nil
}
