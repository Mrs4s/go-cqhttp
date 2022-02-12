package leveldb

import (
	"bytes"
	"io"

	"github.com/Mrs4s/go-cqhttp/global"
)

type intWriter struct {
	bytes.Buffer
}

func (w *intWriter) varint(x int64) {
	w.uvarint(uint64(x)<<1 ^ uint64(x>>63))
}

func (w *intWriter) uvarint(x uint64) {
	for x >= 0x80 {
		w.WriteByte(byte(x) | 0x80)
		x >>= 7
	}
	w.WriteByte(byte(x))
}

type writer struct {
	data        intWriter
	strings     intWriter
	stringIndex map[string]uint64
}

func (w *writer) coder(o coder)    { w.data.WriteByte(byte(o)) }
func (w *writer) varint(x int64)   { w.data.varint(x) }
func (w *writer) uvarint(x uint64) { w.data.uvarint(x) }
func (w *writer) nil()             { w.coder(coderNil) }

func (w *writer) int(i int) {
	w.coder(coderInt)
	w.varint(int64(i))
}

func (w *writer) uint(i uint) {
	w.coder(coderUint)
	w.uvarint(uint64(i))
}

func (w *writer) int32(i int32) {
	w.coder(coderInt32)
	w.varint(int64(i))
}

func (w *writer) uint32(i uint32) {
	w.coder(coderUint32)
	w.uvarint(uint64(i))
}

func (w *writer) int64(i int64) {
	w.coder(coderInt64)
	w.varint(i)
}

func (w *writer) uint64(i uint64) {
	w.coder(coderUint64)
	w.uvarint(i)
}

func (w *writer) string(s string) {
	w.coder(coderString)
	off, ok := w.stringIndex[s]
	if !ok {
		off = uint64(w.strings.Len())
		w.strings.uvarint(uint64(len(s)))
		_, _ = w.strings.WriteString(s)
		w.stringIndex[s] = off
	}
	w.uvarint(off)
}

func (w *writer) msg(m global.MSG) {
	w.coder(coderMSG)
	w.uvarint(uint64(len(m)))
	for s, coder := range m {
		w.string(s)
		w.obj(coder)
	}
}

func (w *writer) arrayMsg(a []global.MSG) {
	w.coder(coderArrayMSG)
	w.uvarint(uint64(len(a)))
	for _, v := range a {
		w.msg(v)
	}
}

func (w *writer) obj(o interface{}) {
	switch x := o.(type) {
	case nil:
		w.nil()
	case int:
		w.int(x)
	case int32:
		w.int32(x)
	case int64:
		w.int64(x)
	case uint:
		w.uint(x)
	case uint32:
		w.uint32(x)
	case uint64:
		w.uint64(x)
	case string:
		w.string(x)
	case global.MSG:
		w.msg(x)
	default:
		panic("unsupported type")
	}
}

func (w *writer) bytes() []byte {
	var out intWriter
	buf := bytes.Buffer{}
	out.uvarint(uint64(w.strings.Len()))
	out.uvarint(uint64(w.data.Len()))
	_, _ = io.Copy(&out, &w.strings)
	_, _ = io.Copy(&out, &w.data)
	_, _ = io.Copy(&buf, &out)
	return buf.Bytes()
}
