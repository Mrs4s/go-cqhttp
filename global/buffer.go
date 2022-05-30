package global

import (
	"bytes"

	"github.com/Mrs4s/MiraiGo/binary" // 和 MiraiGo 共用同一 buffer 池
)

// NewBuffer 从池中获取新 bytes.Buffer
func NewBuffer() *bytes.Buffer {
	return (*bytes.Buffer)(binary.SelectWriter())
}

// PutBuffer 将 Buffer放入池中
func PutBuffer(buf *bytes.Buffer) {
	binary.PutWriter((*binary.Writer)(buf))
}
