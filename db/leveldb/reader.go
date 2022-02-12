package leveldb

import (
	"encoding/binary"
	"io"
	"strconv"
	"strings"

	"github.com/Mrs4s/go-cqhttp/global"
)

type intReader struct {
	data string
	*strings.Reader
}

func newIntReader(s string) intReader {
	return intReader{
		data:   s,
		Reader: strings.NewReader(s),
	}
}

func (r *intReader) varint() int64 {
	i, _ := binary.ReadVarint(r)
	return i
}

func (r *intReader) uvarint() uint64 {
	i, _ := binary.ReadUvarint(r)
	return i
}

type reader struct {
	data        intReader
	strings     intReader
	stringIndex map[uint64]string
}

func (r *reader) coder() coder    { o, _ := r.data.ReadByte(); return coder(o) }
func (r *reader) varint() int64   { return r.data.varint() }
func (r *reader) uvarint() uint64 { return r.data.uvarint() }

func (r *reader) sync(c coder) {
	if coder := r.coder(); coder != c {
		panic("db/leveldb: bad sync expected " + strconv.Itoa(int(c)) + " but got " + strconv.Itoa(int(coder)))
	}
}

func (r *reader) int() int {
	r.sync(coderInt)
	return int(r.varint())
}

func (r *reader) uint() uint {
	r.sync(coderUint)
	return uint(r.uvarint())
}

func (r *reader) int32() int32 {
	r.sync(coderInt32)
	return int32(r.varint())
}

func (r *reader) uint32() uint32 {
	r.sync(coderUint32)
	return uint32(r.uvarint())
}

func (r *reader) int64() int64 {
	r.sync(coderInt64)
	return r.varint()
}

func (r *reader) uint64() uint64 {
	r.sync(coderUint64)
	return r.uvarint()
}

func (r *reader) string() string {
	r.sync(coderString)
	return r.stringNoSync()
}

func (r *reader) stringNoSync() string {
	off := r.data.uvarint()
	if off == 0 {
		return ""
	}
	if s, ok := r.stringIndex[off]; ok {
		return s
	}
	_, _ = r.strings.Seek(int64(off), io.SeekStart)
	l := int64(r.strings.uvarint())
	whence, _ := r.strings.Seek(0, io.SeekCurrent)
	s := r.strings.data[whence : whence+l]
	r.stringIndex[off] = s
	return s
}

func (r *reader) msg() global.MSG {
	r.sync(coderMSG)
	return r.msgNoSync()
}

func (r *reader) msgNoSync() global.MSG {
	length := r.uvarint()
	msg := make(global.MSG, length)
	for i := uint64(0); i < length; i++ {
		s := r.string()
		msg[s] = r.obj()
	}
	return msg
}

func (r *reader) arrayMsg() []global.MSG {
	r.sync(coderArrayMSG)
	return r.arrayMsgNoSync()
}

func (r *reader) arrayMsgNoSync() []global.MSG {
	length := r.uvarint()
	msgs := make([]global.MSG, length)
	for i := range msgs {
		msgs[i] = r.msg()
	}
	return msgs
}

func (r *reader) obj() interface{} {
	switch coder := r.coder(); coder {
	case coderNil:
		return nil
	case coderInt:
		return int(r.varint())
	case coderUint:
		return uint(r.uvarint())
	case coderInt32:
		return int32(r.varint())
	case coderUint32:
		return uint32(r.uvarint())
	case coderInt64:
		return r.varint()
	case coderUint64:
		return r.uvarint()
	case coderString:
		return r.stringNoSync()
	case coderMSG:
		return r.msgNoSync()
	case coderArrayMSG:
		return r.arrayMsgNoSync()
	default:
		panic("db/leveldb: invalid coder " + strconv.Itoa(int(coder)))
	}
}

func newReader(data string) *reader {
	in := newIntReader(data)
	sl := int64(in.uvarint())
	dl := int64(in.uvarint())
	whence, _ := in.Seek(0, io.SeekCurrent)
	sData := data[whence : whence+sl]
	dData := data[whence+sl : whence+sl+dl]
	r := reader{
		data:        newIntReader(dData),
		strings:     newIntReader(sData),
		stringIndex: make(map[uint64]string),
	}
	return &r
}
