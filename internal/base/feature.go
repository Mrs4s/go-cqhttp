package base

import (
	"github.com/pkg/errors"
)

// silk encode features
var (
	EncodeSilk   = encodeSilk   // 编码 SilkV3 音频
	ResampleSilk = resampleSilk // 将silk重新编码为 24000 bit rate
)

func encodeSilk(_ []byte, _ string) ([]byte, error) {
	return nil, errors.New("not supported now")
}

func resampleSilk(data []byte) []byte {
	return data
}
