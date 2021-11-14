// Package base provides base config for go-cqhttp
package base

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/modules/config"
)

// command flags
var (
	LittleC  string // config file
	LittleD  bool   // daemon
	LittleH  bool   // Help
	LittleWD string // working directory
)

// config file flags
var (
	Debug               bool // 是否开启 debug 模式
	RemoveReplyAt       bool // 是否删除reply后的at
	ExtraReplyData      bool // 是否上报额外reply信息
	IgnoreInvalidCQCode bool // 是否忽略无效CQ码
	SplitURL            bool // 是否分割URL
	ForceFragmented     bool // 是否启用强制分片
	SkipMimeScan        bool // 是否跳过Mime扫描
	ReportSelfMessage   bool // 是否上报自身消息
	UseSSOAddress       bool // 是否使用服务器下发的新地址进行重连
	LogForceNew         bool // 是否在每次启动时强制创建全新的文件储存日志
	LogColorful         bool // 是否启用日志颜色
	FastStart           bool // 是否为快速启动

	PostFormat        string                 // 上报格式 string or array
	Proxy             string                 // 存储 proxy_rewrite,用于设置代理
	PasswordHash      [16]byte               // 存储QQ密码哈希供登录使用
	AccountToken      []byte                 // 存储 AccountToken 供登录使用
	Account           *config.Account        // 账户配置
	Reconnect         *config.Reconnect      // 重连配置
	LogLevel          string                 // 日志等级
	LogAging          = time.Hour * 24 * 365 // 日志时效
	HeartbeatInterval = time.Second * 5      // 心跳间隔

	Servers  []map[string]yaml.Node // 连接服务列表
	Database map[string]yaml.Node   // 数据库列表
)

// Parse parse flags
func Parse() {
	wd, _ := os.Getwd()
	dc := path.Join(wd, "config.yml")
	flag.StringVar(&LittleC, "c", dc, "configuration filename")
	flag.BoolVar(&LittleD, "d", false, "running as a daemon")
	flag.BoolVar(&LittleH, "h", false, "this Help")
	flag.StringVar(&LittleWD, "w", "", "cover the working directory")
	d := flag.Bool("D", false, "debug mode")
	flag.Parse()

	if *d {
		Debug = true
	}
}

// Init read config from yml file
func Init() {
	conf := config.Parse(LittleC)
	{ // bool config
		if conf.Output.Debug {
			Debug = true
		}
		IgnoreInvalidCQCode = conf.Message.IgnoreInvalidCQCode
		SplitURL = conf.Message.FixURL
		RemoveReplyAt = conf.Message.RemoveReplyAt
		ExtraReplyData = conf.Message.ExtraReplyData
		ForceFragmented = conf.Message.ForceFragment
		SkipMimeScan = conf.Message.SkipMimeScan
		ReportSelfMessage = conf.Message.ReportSelfMessage
		UseSSOAddress = conf.Account.UseSSOAddress
	}
	{ // others
		Proxy = conf.Message.ProxyRewrite
		Account = conf.Account
		Reconnect = conf.Account.ReLogin
		Servers = conf.Servers
		Database = conf.Database
		LogLevel = conf.Output.LogLevel
		LogColorful = conf.Output.LogColorful == nil || *conf.Output.LogColorful
		if conf.Message.PostFormat != "string" && conf.Message.PostFormat != "array" {
			log.Warnf("post-format 配置错误, 将自动使用 string")
			PostFormat = "string"
		} else {
			PostFormat = conf.Message.PostFormat
		}
		if conf.Output.LogAging > 0 {
			LogAging = time.Hour * 24 * time.Duration(conf.Output.LogAging)
		}
		if conf.Heartbeat.Interval > 0 {
			HeartbeatInterval = time.Second * time.Duration(conf.Heartbeat.Interval)
		}
		if conf.Heartbeat.Disabled || conf.Heartbeat.Interval < 0 {
			HeartbeatInterval = 0
		}
	}
}

// Help cli命令行-h的帮助提示
func Help() {
	fmt.Printf(`go-cqhttp service
version: %s
Usage:
server [OPTIONS]
Options:
`, Version)

	flag.PrintDefaults()
	os.Exit(0)
}

// ResetWorkingDir 重设工作路径
func ResetWorkingDir() {
	wd := LittleWD
	args := make([]string, 0, len(os.Args))
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-w" {
			i++ // skip value field
		} else if !strings.HasPrefix(os.Args[i], "-w") {
			args = append(args, os.Args[i])
		}
	}
	p, _ := filepath.Abs(os.Args[0])
	proc := exec.Command(p, args...)
	proc.Stdin = os.Stdin
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	proc.Dir = wd
	err := proc.Run()
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}
