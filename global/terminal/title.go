//go:build !windows

package terminal

import (
	"fmt"
	"time"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

func init() {
	// 设置标题
	fmt.Printf("\033]0;go-cqhttp "+base.Version+" © 2020 - %d Mrs4s"+"\007", time.Now().Year())
}
