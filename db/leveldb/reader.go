package leveldb

import (
	"encoding/binary"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"

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

func (r *intReader) varInt() int64 {
	i, _ := binary.ReadVarint(r)
	return i
}

func (r *intReader) varUint() uint64 {
	i, _ := binary.ReadUvarint(r)
	return i
}

// reader implements the index read.
// data format is the same as the writer's
type reader struct {
	data        intReader
	strings     intReader
	stringIndex map[uint64]string
}

func (r *reader) coder() coder    { o, _ := r.data.ReadByte(); return coder(o) }
func (r *reader) varInt() int64   { return r.data.varInt() }
func (r *reader) varUint() uint64 { return r.data.varUint() }
func (r *reader) int32() int32    { return int32(r.varInt()) }
func (r *reader) int64() int64    { return r.varInt() }
func (r *reader) uint64() uint64  { return r.varUint() }

// func (r *reader) uint32() uint32 { return uint32(r.varUint()) }
// func (r *reader) int() int        { return int(r.varInt()) }
// func (r *reader) uint() uint      { return uint(r.varUint()) }

func (r *reader) string() string {
	off := r.data.varUint()
	if s, ok := r.stringIndex[off]; ok {
		return s
	}
	_, _ = r.strings.Seek(int64(off), io.SeekStart)
	l := int64(r.strings.varUint())
	whence, _ := r.strings.Seek(0, io.SeekCurrent)
	s := r.strings.data[whence : whence+l]
	r.stringIndex[off] = s
	return s
}

func (r *reader) msg() global.MSG {
	length := r.varUint()
	msg := make(global.MSG, length)
	for i := uint64(0); i < length; i++ {
		s := r.string()
		msg[s] = r.obj()
	}
	return msg
}

func (r *reader) arrayMsg() []global.MSG {
	length := r.varUint()
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
		return int(r.varInt())
	case coderUint:
		return uint(r.varUint())
	case coderInt32:
		return int32(r.varInt())
	case coderUint32:
		return uint32(r.varUint())
	case coderInt64:
		return r.varInt()
	case coderUint64:
		return r.varUint()
	case coderString:
		return r.string()
	case coderMSG:
		return r.msg()
	case coderArrayMSG:
		return r.arrayMsg()
	default:
		panic("db/leveldb: invalid coder " + strconv.Itoa(int(coder)))
	}
}

func newReader(data string) (*reader, error) {
	in := newIntReader(data)
	v := in.varUint()
	if v != dataVersion {
		return nil, errors.Errorf("db/leveldb: invalid data version %d", v)
	}
	sl := int64(in.varUint())
	dl := int64(in.varUint())
	whence, _ := in.Seek(0, io.SeekCurrent)
	sData := data[whence : whence+sl]
	dData := data[whence+sl : whence+sl+dl]
	r := reader{
		data:        newIntReader(dData),
		strings:     newIntReader(sData),
		stringIndex: make(map[uint64]string),
	}
	return &r, nil
}
