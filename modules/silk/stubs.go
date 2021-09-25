// Package silk Silk编码核心模块
package silk

import (
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

func init() {
	base.EncodeSilk = encode
	base.ResampleSilk = resample
}
