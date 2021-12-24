package gocq

import (
	"os"

	para "github.com/fumiama/go-hide-param"

	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/selfupdate"
)

// ParseCommand 解析命令
func ParseCommand() (byteKey []byte) {
	arg := os.Args
	if len(arg) > 1 {
		for i := range arg {
			switch arg[i] {
			case "update":
				if len(arg) > i+1 {
					selfupdate.SelfUpdate(arg[i+1])
				} else {
					selfupdate.SelfUpdate("")
				}
			case "key":
				p := i + 1
				if len(arg) > p {
					byteKey = []byte(arg[p])
					para.Hide(p)
				}
			case "faststart":
				base.FastStart = true
			}
		}
	}
	return
}
