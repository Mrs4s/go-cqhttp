package coolq

import (
	"runtime"

	"github.com/tidwall/gjson"

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

// CQSendMessageV12 发送消息
//
// @route12(send_message)
// @rename(m->message)
func (bot *CQBot) CQSendMessageV12(groupID, userID, detailType string, m gjson.Result) global.MSG { // nolint
	// TODO: implement
	return OK(nil)
}
