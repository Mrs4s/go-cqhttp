// Package config 包含go-cqhttp操作配置文件的相关函数
package config

import (
	"bufio"
	_ "embed" // embed the default config file
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/internal/param"
)

// defaultConfig 默认配置文件
//go:embed default_config.yml
var defaultConfig string

// Reconnect 重连配置
type Reconnect struct {
	Disabled bool `yaml:"disabled"`
	Delay    uint `yaml:"delay"`
	MaxTimes uint `yaml:"max-times"`
	Interval int  `yaml:"interval"`
}

// Account 账号配置
type Account struct {
	Uin           int64      `yaml:"uin"`
	Password      string     `yaml:"password"`
	Encrypt       bool       `yaml:"encrypt"`
	Status        int        `yaml:"status"`
	ReLogin       *Reconnect `yaml:"relogin"`
	UseSSOAddress bool       `yaml:"use-sso-address"`
}

// Config 总配置文件
type Config struct {
	Account   *Account `yaml:"account"`
	Heartbeat struct {
		Disabled bool `yaml:"disabled"`
		Interval int  `yaml:"interval"`
	} `yaml:"heartbeat"`

	Message struct {
		PostFormat          string `yaml:"post-format"`
		ProxyRewrite        string `yaml:"proxy-rewrite"`
		IgnoreInvalidCQCode bool   `yaml:"ignore-invalid-cqcode"`
		ForceFragment       bool   `yaml:"force-fragment"`
		FixURL              bool   `yaml:"fix-url"`
		ReportSelfMessage   bool   `yaml:"report-self-message"`
		RemoveReplyAt       bool   `yaml:"remove-reply-at"`
		ExtraReplyData      bool   `yaml:"extra-reply-data"`
		SkipMimeScan        bool   `yaml:"skip-mime-scan"`
	} `yaml:"message"`

	Output struct {
		LogLevel    string `yaml:"log-level"`
		LogAging    int    `yaml:"log-aging"`
		LogForceNew bool   `yaml:"log-force-new"`
		LogColorful *bool  `yaml:"log-colorful"`
		Debug       bool   `yaml:"debug"`
	} `yaml:"output"`

	Servers  []map[string]yaml.Node `yaml:"servers"`
	Database map[string]yaml.Node   `yaml:"database"`
}

// Server 的简介和初始配置
type Server struct {
	Brief    string
	Default  string
	ParseEnv func() (string, *yaml.Node)
}

// LevelDBConfig leveldb 相关配置
type LevelDBConfig struct {
	Enable bool `yaml:"enable"`
}

// MongoDBConfig mongodb 相关配置
type MongoDBConfig struct {
	Enable   bool   `yaml:"enable"`
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

// Parse 从默认配置文件路径中获取
func Parse(path string) *Config {
	fromEnv := os.Getenv("GCQ_UIN") != ""

	file, err := os.ReadFile(path)
	config := &Config{}
	if err == nil {
		err = yaml.NewDecoder(strings.NewReader(os.ExpandEnv(string(file)))).Decode(config)
		if err != nil && !fromEnv {
			log.Fatal("配置文件不合法!", err)
		}
	} else if !fromEnv {
		generateConfig()
		os.Exit(0)
	}
	if fromEnv {
		// type convert tools
		toInt64 := func(str string) int64 {
			i, _ := strconv.ParseInt(str, 10, 64)
			return i
		}

		// load config from environment variable
		param.SetAtDefault(&config.Account.Uin, toInt64(os.Getenv("GCQ_UIN")), int64(0))
		param.SetAtDefault(&config.Account.Password, os.Getenv("GCQ_PWD"), "")
		param.SetAtDefault(&config.Account.Status, int32(toInt64(os.Getenv("GCQ_STATUS"))), int32(0))
		param.SetAtDefault(&config.Account.ReLogin.Disabled, !param.EnsureBool(os.Getenv("GCQ_RELOGIN_DISABLED"), true), false)
		param.SetAtDefault(&config.Account.ReLogin.Delay, uint(toInt64(os.Getenv("GCQ_RELOGIN_DELAY"))), uint(0))
		param.SetAtDefault(&config.Account.ReLogin.MaxTimes, uint(toInt64(os.Getenv("GCQ_RELOGIN_MAX_TIMES"))), uint(0))
		dbConf := &LevelDBConfig{Enable: param.EnsureBool(os.Getenv("GCQ_LEVELDB"), true)}
		if config.Database == nil {
			config.Database = make(map[string]yaml.Node)
		}
		config.Database["leveldb"] = func() yaml.Node {
			n := &yaml.Node{}
			_ = n.Encode(dbConf)
			return *n
		}()

		for _, s := range serverconfs {
			if s.ParseEnv != nil {
				name, node := s.ParseEnv()
				if node != nil {
					config.Servers = append(config.Servers, map[string]yaml.Node{name: *node})
				}
			}
		}
	}
	return config
}

var serverconfs []*Server
var mu sync.Mutex

// AddServer 添加该服务的简介和默认配置
func AddServer(s *Server) {
	mu.Lock()
	serverconfs = append(serverconfs, s)
	mu.Unlock()
}

// generateConfig 生成配置文件
func generateConfig() {
	fmt.Println("未找到配置文件，正在为您生成配置文件中！")
	sb := strings.Builder{}
	sb.WriteString(defaultConfig)
	hint := "请选择你需要的通信方式:"
	for i, s := range serverconfs {
		hint += fmt.Sprintf("\n> %d: %s", i, s.Brief)
	}
	hint += `
请输入你需要的编号(0-9)，可输入多个，同一编号也可输入多个(如: 233)
您的选择是:`
	fmt.Print(hint)
	input := bufio.NewReader(os.Stdin)
	readString, err := input.ReadString('\n')
	if err != nil {
		log.Fatal("输入不合法: ", err)
	}
	rmax := len(serverconfs)
	if rmax > 10 {
		rmax = 10
	}
	for _, r := range readString {
		r -= '0'
		if r >= 0 && r < rune(rmax) {
			sb.WriteString(serverconfs[r].Default)
		}
	}
	_ = os.WriteFile("config.yml", []byte(sb.String()), 0o644)
	fmt.Println("默认配置文件已生成，请修改 config.yml 后重新启动!")
	_, _ = input.ReadString('\n')
}
