package coolq

import (
	"runtime"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

// CQGetVersion 获取版本信息 OneBotV12
//
// https://git.io/JtwUs
// @route12(get_version)
func (bot *CQBot) CQGetVersion() global.MSG {
	return OK(global.MSG{
		"impl":            "go_cqhttp",
		"platform":        "qq",
		"version":         base.Version,
		"onebot_version":  12,
		"runtime_version": runtime.Version(),
		"runtime_os":      runtime.GOOS,
	})
}
