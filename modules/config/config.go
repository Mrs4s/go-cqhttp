// Package config 包含go-cqhttp操作配置文件的相关函数
package config

import (
	"bufio"
	_ "embed" // embed the default config file
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Mrs4s/go-cqhttp/internal/param"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
		Debug       bool   `yaml:"debug"`
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
	Disabled    bool   `yaml:"disabled"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Timeout     int32  `yaml:"timeout"`
	LongPolling struct {
		Enabled      bool `yaml:"enabled"`
		MaxQueueSize int  `yaml:"max-queue-size"`
	} `yaml:"long-polling"`
	Post []struct {
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

// LambdaServer 云函数配置
type LambdaServer struct {
	Disabled bool   `yaml:"disabled"`
	Type     string `yaml:"type"`

	MiddleWares `yaml:"middlewares"`
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

	file, err := os.Open(path)
	config := &Config{}
	if err == nil {
		defer func() { _ = file.Close() }()
		if err = yaml.NewDecoder(file).Decode(config); err != nil && !fromEnv {
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
		accessTokenEnv := os.Getenv("GCQ_ACCESS_TOKEN")
		if os.Getenv("GCQ_HTTP_PORT") != "" {
			node := &yaml.Node{}
			httpConf := &HTTPServer{
				Host: "0.0.0.0",
				Port: 5700,
				MiddleWares: MiddleWares{
					AccessToken: accessTokenEnv,
				},
			}
			param.SetExcludeDefault(&httpConf.Disabled, param.EnsureBool(os.Getenv("GCQ_HTTP_DISABLE"), false), false)
			param.SetExcludeDefault(&httpConf.Host, os.Getenv("GCQ_HTTP_HOST"), "")
			param.SetExcludeDefault(&httpConf.Port, int(toInt64(os.Getenv("GCQ_HTTP_PORT"))), 0)
			if os.Getenv("GCQ_HTTP_POST_URL") != "" {
				httpConf.Post = append(httpConf.Post, struct {
					URL    string `yaml:"url"`
					Secret string `yaml:"secret"`
				}{os.Getenv("GCQ_HTTP_POST_URL"), os.Getenv("GCQ_HTTP_POST_SECRET")})
			}
			_ = node.Encode(httpConf)
			config.Servers = append(config.Servers, map[string]yaml.Node{"http": *node})
		}
		if os.Getenv("GCQ_WS_PORT") != "" {
			node := &yaml.Node{}
			wsServerConf := &WebsocketServer{
				Host: "0.0.0.0",
				Port: 6700,
				MiddleWares: MiddleWares{
					AccessToken: accessTokenEnv,
				},
			}
			param.SetExcludeDefault(&wsServerConf.Disabled, param.EnsureBool(os.Getenv("GCQ_WS_DISABLE"), false), false)
			param.SetExcludeDefault(&wsServerConf.Host, os.Getenv("GCQ_WS_HOST"), "")
			param.SetExcludeDefault(&wsServerConf.Port, int(toInt64(os.Getenv("GCQ_WS_PORT"))), 0)
			_ = node.Encode(wsServerConf)
			config.Servers = append(config.Servers, map[string]yaml.Node{"ws": *node})
		}
		if os.Getenv("GCQ_RWS_API") != "" || os.Getenv("GCQ_RWS_EVENT") != "" || os.Getenv("GCQ_RWS_UNIVERSAL") != "" {
			node := &yaml.Node{}
			rwsConf := &WebsocketReverse{
				MiddleWares: MiddleWares{
					AccessToken: accessTokenEnv,
				},
			}
			param.SetExcludeDefault(&rwsConf.Disabled, param.EnsureBool(os.Getenv("GCQ_RWS_DISABLE"), false), false)
			param.SetExcludeDefault(&rwsConf.API, os.Getenv("GCQ_RWS_API"), "")
			param.SetExcludeDefault(&rwsConf.Event, os.Getenv("GCQ_RWS_EVENT"), "")
			param.SetExcludeDefault(&rwsConf.Universal, os.Getenv("GCQ_RWS_UNIVERSAL"), "")
			_ = node.Encode(rwsConf)
			config.Servers = append(config.Servers, map[string]yaml.Node{"ws-reverse": *node})
		}
	}
	return config
}

// generateConfig 生成配置文件
func generateConfig() {
	fmt.Println("未找到配置文件，正在为您生成配置文件中！")
	sb := strings.Builder{}
	sb.WriteString(defaultConfig)
	fmt.Print(`请选择你需要的通信方式:
> 1: HTTP通信
> 2: 正向 Websocket 通信
> 3: 反向 Websocket 通信
> 4: pprof 性能分析服务器
> 5: 云函数服务
请输入你需要的编号，可输入多个，同一编号也可输入多个(如: 233)
您的选择是:`)
	input := bufio.NewReader(os.Stdin)
	readString, err := input.ReadString('\n')
	if err != nil {
		log.Fatal("输入不合法: ", err)
	}
	for _, r := range readString {
		switch r {
		case '1':
			sb.WriteString(httpDefault)
		case '2':
			sb.WriteString(wsDefault)
		case '3':
			sb.WriteString(wsReverseDefault)
		case '4':
			sb.WriteString(pprofDefault)
		case '5':
			sb.WriteString(lambdaDefault)
		}
	}
	_ = os.WriteFile("config.yml", []byte(sb.String()), 0o644)
	fmt.Println("默认配置文件已生成，请修改 config.yml 后重新启动!")
	_, _ = input.ReadString('\n')
}

const httpDefault = `  # HTTP 通信设置
  - http:
      # 服务端监听地址
      host: 127.0.0.1
      # 服务端监听端口
      port: 5700
      # 反向HTTP超时时间, 单位秒
      # 最小值为5，小于5将会忽略本项设置
      timeout: 5
      # 长轮询拓展
      long-polling:
        # 是否开启
        enabled: false
        # 消息队列大小，0 表示不限制队列大小，谨慎使用
        max-queue-size: 2000
      middlewares:
        <<: *default # 引用默认中间件
      # 反向HTTP POST地址列表
      post:
      #- url: '' # 地址
      #  secret: ''           # 密钥
      #- url: 127.0.0.1:5701 # 地址
      #  secret: ''          # 密钥
`

const lambdaDefault = `  # LambdaServer 配置
  - lambda:
      type: scf # scf: 腾讯云函数 aws: aws Lambda
      middlewares:
        <<: *default # 引用默认中间件
`

const wsDefault = `  # 正向WS设置
  - ws:
      # 正向WS服务器监听地址
      host: 127.0.0.1
      # 正向WS服务器监听端口
      port: 6700
      middlewares:
        <<: *default # 引用默认中间件
`

const wsReverseDefault = `  # 反向WS设置
  - ws-reverse:
      # 反向WS Universal 地址
      # 注意 设置了此项地址后下面两项将会被忽略
      universal: ws://your_websocket_universal.server
      # 反向WS API 地址
      api: ws://your_websocket_api.server
      # 反向WS Event 地址
      event: ws://your_websocket_event.server
      # 重连间隔 单位毫秒
      reconnect-interval: 3000
      middlewares:
        <<: *default # 引用默认中间件
`

const pprofDefault = `  # pprof 性能分析服务器, 一般情况下不需要启用.
  # 如果遇到性能问题请上传报告给开发者处理
  # 注意: pprof服务不支持中间件、不支持鉴权. 请不要开放到公网
  - pprof:
      # pprof服务器监听地址
      host: 127.0.0.1
      # pprof服务器监听端口
      port: 7700
`
