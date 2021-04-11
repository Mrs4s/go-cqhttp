// Package config 包含go-cqhttp操作配置文件的相关函数
package config

import (
	_ "embed" // embed the default config file
	"os"
	"path"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// DefaultConfig 默认配置文件
//go:embed default_config.yml
var DefaultConfig string

var currentPath = getCurrentPath()

// DefaultConfigFile 默认配置文件路径
var DefaultConfigFile = path.Join(currentPath, "config.yml")

// Config 总配置文件
type Config struct {
	Account struct {
		Uin      int64  `yaml:"uin"`
		Password string `yaml:"password"`
		Encrypt  bool   `yaml:"encrypt"`
		Status   int32  `yaml:"status"`
		ReLogin  struct {
			Disabled bool `yaml:"disabled"`
			Delay    int  `yaml:"delay"`
			MaxTimes uint `yaml:"max-times"`
			Interval int  `yaml:"interval"`
		}
		UseSSOAddress bool `yaml:"use-sso-address"`
	} `yaml:"account"`

	Heartbeat struct {
		Disabled bool `yaml:"disabled"`
		Interval int  `yaml:"interval"`
	} `yaml:"heartbeat"`

	Message struct {
		PostFormat          string `yaml:"post-format"`
		IgnoreInvalidCQCode bool   `yaml:"ignore-invalid-cqcode"`
		ForceFragment       bool   `yaml:"force-fragment"`
		FixURL              bool   `yaml:"fix-url"`
		ProxyRewrite        string `yaml:"proxy-rewrite"`
		ReportSelfMessage   bool   `yaml:"report-self-message"`
		RemoveReplyAt       bool   `yaml:"remove-reply-at"`
		ExtraReplyData      bool   `yaml:"extra-reply-data"`
	} `yaml:"message"`

	Output struct {
		LogLevel string `yaml:"log-level"`
		Debug    bool   `yaml:"debug"`
	} `yaml:"output"`

	Servers  []map[string]yaml.Node `yaml:"servers"`
	Database map[string]yaml.Node   `yaml:"database"`
}

// MiddleWares 通信中间件
type MiddleWares struct {
	AccessToken string `yaml:"access-token"`
	Filter      string `yaml:"filter"`
	RateLimit   struct {
		Enabled   bool    `yaml:"enabled"`
		Frequency float64 `yaml:"frequency"`
		Bucket    int     `yaml:"bucket"`
	} `yaml:"rate-limit"`
}

// HTTPServer HTTP通信相关配置
type HTTPServer struct {
	Disabled bool   `yaml:"disabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Timeout  int32  `yaml:"timeout"`
	Post     []struct {
		URL    string `yaml:"url"`
		Secret string `yaml:"secret"`
	}

	MiddleWares `yaml:"middlewares"`
}

// PprofServer pprof性能分析服务器相关配置
type PprofServer struct {
	Disabled bool   `yaml:"disabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
}

// WebsocketServer 正向WS相关配置
type WebsocketServer struct {
	Disabled bool   `yaml:"disabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`

	MiddleWares `yaml:"middlewares"`
}

// WebsocketReverse 反向WS相关配置
type WebsocketReverse struct {
	Disabled          bool   `yaml:"disabled"`
	Universal         string `yaml:"universal"`
	API               string `yaml:"api"`
	Event             string `yaml:"event"`
	ReconnectInterval int    `yaml:"reconnect-interval"`

	MiddleWares `yaml:"middlewares"`
}

// LevelDBConfig leveldb 相关配置
type LevelDBConfig struct {
	Enable bool `yaml:"enable"`
}

var (
	config *Config
	once   sync.Once
)

// Get 从默认配置文件路径中获取
func Get() *Config {
	once.Do(func() {
		file, err := os.Open(DefaultConfigFile)
		if err != nil {
			log.Error("获取配置文件失败: ", err)
			return
		}
		defer file.Close()
		config = &Config{}
		if err = yaml.NewDecoder(file).Decode(config); err != nil {
			log.Fatal("配置文件不合法!", err)
		}
	})
	return config
}

// getCurrentPath 获取当前文件的路径，直接返回string
func getCurrentPath() string {
	cwd, e := os.Getwd()
	if e != nil {
		panic(e)
	}
	return cwd
}
