// Package base provides base config for go-cqhttp
package base

import (
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global/config"
)

// flags
var (
	Debug               bool // 是否开启 debug 模式
	RemoveReplyAt       bool // 是否删除reply后的at
	ExtraReplyData      bool // 是否上报额外reply信息
	IgnoreInvalidCQCode bool // 是否忽略无效CQ码
	SplitURL            bool // 是否分割URL
	ForceFragmented     bool // 是否启用强制分片
	SkipMimeScan        bool // 是否跳过Mime扫描

	PostFormat   = "string" // 上报格式 string or array
	Proxy        string     // 存储 proxy_rewrite,用于设置代理
	PasswordHash [16]byte   // 存储QQ密码哈希供登录使用
	AccountToken []byte     // 存储AccountToken供登录使用
)

// Parse parses flags from config file
func Parse() {
	conf := config.Get()
	{ // bool config
		Debug = conf.Output.Debug
		IgnoreInvalidCQCode = conf.Message.IgnoreInvalidCQCode
		SplitURL = conf.Message.FixURL
		RemoveReplyAt = conf.Message.RemoveReplyAt
		ExtraReplyData = conf.Message.ExtraReplyData
		ForceFragmented = conf.Message.ForceFragment
		SkipMimeScan = conf.Message.SkipMimeScan
	}
	{ // string
		Proxy = conf.Message.ProxyRewrite
		if conf.Message.PostFormat != "string" && conf.Message.PostFormat != "array" {
			log.Warnf("post-format 配置错误, 将自动使用 string")
			PostFormat = "string"
		} else {
			PostFormat = conf.Message.PostFormat
		}
	}
}
