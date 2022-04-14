// Package config 包含go-cqhttp操作配置文件的相关函数
package config

import (
	"bufio"
	_ "embed" // embed the default config file
	"fmt"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// defaultConfig 默认配置文件
//
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
	Uin              int64      `yaml:"uin"`
	Password         string     `yaml:"password"`
	Encrypt          bool       `yaml:"encrypt"`
	Status           int        `yaml:"status"`
	ReLogin          *Reconnect `yaml:"relogin"`
	UseSSOAddress    bool       `yaml:"use-sso-address"`
	AllowTempSession bool       `yaml:"allow-temp-session"`
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
	Brief   string
	Default string
}

// Parse 从默认配置文件路径中获取
func Parse(path string) *Config {
	file, err := os.ReadFile(path)
	config := &Config{}
	if err == nil {
		err = yaml.NewDecoder(strings.NewReader(expand(string(file), os.Getenv))).Decode(config)
		if err != nil {
			log.Fatal("配置文件不合法!", err)
		}
	} else {
		generateConfig()
		os.Exit(0)
	}
	return config
}

var serverconfs []*Server

// AddServer 添加该服务的简介和默认配置
func AddServer(s *Server) {
	serverconfs = append(serverconfs, s)
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

// expand 使用正则进行环境变量展开
// os.ExpandEnv 字符 $ 无法逃逸
// https://github.com/golang/go/issues/43482
func expand(s string, mapping func(string) string) string {
	r := regexp.MustCompile(`\${([a-zA-Z_]+[a-zA-Z0-9_:/.]*)}`)
	return r.ReplaceAllStringFunc(s, func(s string) string {
		s = strings.Trim(s, "${}")
		before, after, ok := strings.Cut(s, ":")
		m := mapping(before)
		if ok && m == "" {
			return after
		}
		return m
	})
}
