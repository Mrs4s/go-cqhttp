package leveldb

import (
	"bytes"

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

// writer implements the index write.
//
// data format(use uvarint to encode integers):
//
//   - version
//   - string data length
//   - index data length
//   - string data
//   - index data
//
// for string data part, each string is encoded as:
//
//   - string length
//   - string
//
// for index data part, each object value is encoded as:
//
//   - coder
//   - value
//
// * coder is the identifier of value's type.
// * specially for string, it's value is the offset in string data part.
type writer struct {
	data        intWriter
	strings     intWriter
	stringIndex map[string]uint64
}

func newWriter() *writer {
	return &writer{
		stringIndex: make(map[string]uint64),
	}
}

func (w *writer) coder(o coder)    { w.data.WriteByte(byte(o)) }
func (w *writer) varint(x int64)   { w.data.varint(x) }
func (w *writer) uvarint(x uint64) { w.data.uvarint(x) }
func (w *writer) nil()             { w.coder(coderNil) }
func (w *writer) int(i int)        { w.varint(int64(i)) }
func (w *writer) uint(i uint)      { w.uvarint(uint64(i)) }
func (w *writer) int32(i int32)    { w.varint(int64(i)) }
func (w *writer) uint32(i uint32)  { w.uvarint(uint64(i)) }
func (w *writer) int64(i int64)    { w.varint(i) }
func (w *writer) uint64(i uint64)  { w.uvarint(i) }

func (w *writer) string(s string) {
	off, ok := w.stringIndex[s]
	if !ok {
		// not found write to string data part
		// | string length | string |
		off = uint64(w.strings.Len())
		w.strings.uvarint(uint64(len(s)))
		_, _ = w.strings.WriteString(s)
		w.stringIndex[s] = off
	}
	// write offset to index data part
	w.uvarint(off)
}

func (w *writer) msg(m global.MSG) {
	w.uvarint(uint64(len(m)))
	for s, obj := range m {
		w.string(s)
		w.obj(obj)
	}
}

func (w *writer) arrayMsg(a []global.MSG) {
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
		w.coder(coderInt)
		w.int(x)
	case int32:
		w.coder(coderInt32)
		w.int32(x)
	case int64:
		w.coder(coderInt64)
		w.int64(x)
	case uint:
		w.coder(coderUint)
		w.uint(x)
	case uint32:
		w.coder(coderUint32)
		w.uint32(x)
	case uint64:
		w.coder(coderUint64)
		w.uint64(x)
	case string:
		w.coder(coderString)
		w.string(x)
	case global.MSG:
		w.coder(coderMSG)
		w.msg(x)
	case []global.MSG:
		w.coder(coderArrayMSG)
		w.arrayMsg(x)
	default:
		panic("unsupported type")
	}
}

func (w *writer) bytes() []byte {
	var out intWriter
	out.uvarint(dataVersion)
	out.uvarint(uint64(w.strings.Len()))
	out.uvarint(uint64(w.data.Len()))
	_, _ = w.strings.WriteTo(&out)
	_, _ = w.data.WriteTo(&out)
	return out.Bytes()
}
