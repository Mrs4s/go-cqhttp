//go:build !windows

package terminal

import (
	"fmt"
	"time"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

// SetTitle 设置标题为 go-cqhttp `版本` `版权`
func SetTitle() {
	fmt.Printf("\033]0;go-cqhttp "+base.Version+" © 2020 - %d Mrs4s"+"\007", time.Now().Year())
}
