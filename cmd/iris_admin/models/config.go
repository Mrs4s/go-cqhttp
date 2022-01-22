package models

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/internal/param"
	config2 "github.com/Mrs4s/go-cqhttp/modules/config"
	"github.com/Mrs4s/go-cqhttp/server"
)

// QQConfig 配置信息
type QQConfig struct {
	ID          int64  `gorm:"primary_key,column:id"`
	ConfigKey   string `gorm:"column:key"`   // 配置的key
	ConfigName  string `gorm:"column:name"`  // 配置的中文名
	ConfigValue string `gorm:"column:value"` // 配置信息
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type serverHTTP struct {
	Enable  bool             `json:"enable"`
	Host    string           `json:"host"`
	Port    int              `json:"port"`
	Timeout int              `json:"timeout"`
	Post    []serverHTTPPost `json:"post"`
}

type serverHTTPPost struct {
	Disable bool   `json:"disable"`
	URL     string `json:"url"`
	Secret  string `json:"secret"`
}

type serverWs struct {
	Enable bool   `json:"enable"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

type serverWsReverse struct {
	Enable            bool   `json:"enable"`
	Universal         string `json:"universal"` // 反向WS Universal 地址 注意 设置了此项地址后下面两项将会被忽略
	API               string `json:"api"`       // 反向WS API 地址
	Event             string `json:"event"`     // 反向WS Event 地址
	ReconnectInterval int    `json:"reconnect-interval"`
}

// NewQqConfig 初始化db
func NewQqConfig() *gorm.DB {
	return orm.Table("qq_config")
}

// GetQqConfig 拉取配置信息，从db转成yaml一样的配置
func GetQqConfig() (*config2.Config, error) {
	var QQconfigs []QQConfig
	result := NewQqConfig().Find(&QQconfigs)
	if result.Error != nil {
		return nil, result.Error
	}
	return paraseConfig(QQconfigs)
}

type config struct {
	AccountUin                           string `json:"account_uin"`
	AccountPassword                      string `json:"account_password"`
	AccountEncrypt                       string `json:"account_encrypt"`
	AccountEncryptByteKey                string `json:"account_encrypt_bytekey"`
	AccountStatus                        string `json:"account_status"`
	AccountReloginDisable                string `json:"account_relogin_disable"`
	AccountReloginDelay                  string `json:"account_relogin_delay"`
	AccountReloginInterval               string `json:"account_relogin_interval"`
	AccountReloginMaxTimes               string `json:"account_relogin_max-times"`
	AccountUseSsoAddress                 string `json:"account_use-sso-address"`
	HeartbeatDisable                     string `json:"heartbeat_disable"`
	HeartbeatInterval                    string `json:"heartbeat_interval"`
	MessagePostFormat                    string `json:"message_post-format"`
	MessageIgnoreInvalidCqcode           string `json:"message_ignore-invalid-cqcode"`
	MessageForceFragment                 string `json:"message_force-fragment"`
	MessageFixURL                        string `json:"message_fix-url"`
	MessageProxyRewrite                  string `json:"message_proxy-rewrite"`
	MessageReportSelfMessage             string `json:"message_report-self-message"`
	MessageRemoveReplyAt                 string `json:"message_remove-reply-at"`
	MessageExtraReplyData                string `json:"message_extra-reply-data"`
	MessageSkipMimeScan                  string `json:"message_skip-mime-scan"`
	OutputLogLevel                       string `json:"output_log-level"`
	OutputLogAging                       string `json:"output_log-aging"`
	OutputLogForceNew                    string `json:"output_log-force-new"`
	OutputDebug                          string `json:"output_debug"`
	DefaultMiddlewaresAccessToken        string `json:"default_middlewares-access-token"`
	DefaultMiddlewaresFilter             string `json:"default_middlewares-filter"`
	DefaultMiddlewaresRateLimitEnable    string `json:"default_middlewares_rate-limit-enable"`
	DefaultMiddlewaresRateLimitFrequency string `json:"default_middlewares_rate-limit-frequency"`
	DefaultMiddlewaresRatteLimitBucket   string `json:"default_middlewares_ratte-limit-bucket"`
	DatabasesLeveldbEnable               string `json:"databases_leveldb_enable"`
	DatabaseCacheImage                   string `json:"database_cache_image"`
	DatabaseCacheVideo                   string `json:"database_cache_video"`
	ServersHTTP                          string `json:"servers_http"`
	ServersWs                            string `json:"servers_ws"`
	ServersWsReverse                     string `json:"servers_ws-reverse"`
}

// 翻译成go-cqhttp的cfg配置
func paraseConfig(configs []QQConfig) (*config2.Config, error) {
	configmap := make(map[string]string)
	for _, cfg := range configs {
		configmap[cfg.ConfigKey] = cfg.ConfigValue
	}
	bytes, _ := json.Marshal(configmap)
	var cfgs config
	_ = json.Unmarshal(bytes, &cfgs)
	uin, _ := strconv.ParseInt(cfgs.AccountUin, 10, 64)
	accountStatus, _ := strconv.Atoi(cfgs.AccountStatus)
	accountReloginDelay, _ := strconv.ParseUint(cfgs.AccountReloginDelay, 10, 64)
	accountReloginMaxtime, _ := strconv.ParseUint(cfgs.AccountReloginMaxTimes, 10, 64)
	accountReloginInterval, _ := strconv.Atoi(cfgs.AccountReloginInterval)
	hartbeatInterval, _ := strconv.Atoi(cfgs.HeartbeatInterval)
	outlogLogAging, _ := strconv.Atoi(cfgs.OutputLogAging)
	outlogColorfull := false
	cfb := &config2.Config{
		Account: &config2.Account{
			Uin:           uin,
			Password:      cfgs.AccountPassword,
			Encrypt:       param.EnsureBool(cfgs.AccountEncrypt, false),
			Status:        accountStatus,
			UseSSOAddress: param.EnsureBool(cfgs.AccountUseSsoAddress, true),
			ReLogin: &config2.Reconnect{
				Disabled: param.EnsureBool(cfgs.AccountReloginDisable, false),
				Delay:    uint(accountReloginDelay),
				MaxTimes: uint(accountReloginMaxtime),
				Interval: accountReloginInterval,
			},
		},
		Heartbeat: struct {
			Disabled bool `yaml:"disabled"`
			Interval int  `yaml:"interval"`
		}{
			Disabled: param.EnsureBool(cfgs.HeartbeatDisable, false),
			Interval: hartbeatInterval,
		},
		Message: struct {
			PostFormat          string `yaml:"post-format"`
			ProxyRewrite        string `yaml:"proxy-rewrite"`
			IgnoreInvalidCQCode bool   `yaml:"ignore-invalid-cqcode"`
			ForceFragment       bool   `yaml:"force-fragment"`
			FixURL              bool   `yaml:"fix-url"`
			ReportSelfMessage   bool   `yaml:"report-self-message"`
			RemoveReplyAt       bool   `yaml:"remove-reply-at"`
			ExtraReplyData      bool   `yaml:"extra-reply-data"`
			SkipMimeScan        bool   `yaml:"skip-mime-scan"`
		}{
			PostFormat:          cfgs.MessagePostFormat,
			ProxyRewrite:        cfgs.MessageProxyRewrite,
			IgnoreInvalidCQCode: param.EnsureBool(cfgs.MessageIgnoreInvalidCqcode, false),
			ForceFragment:       param.EnsureBool(cfgs.MessageForceFragment, false),
			FixURL:              param.EnsureBool(cfgs.MessageFixURL, false),
			ReportSelfMessage:   param.EnsureBool(cfgs.MessageReportSelfMessage, false),
			RemoveReplyAt:       param.EnsureBool(cfgs.MessageRemoveReplyAt, false),
			ExtraReplyData:      param.EnsureBool(cfgs.MessageExtraReplyData, false),
			SkipMimeScan:        param.EnsureBool(cfgs.MessageSkipMimeScan, false),
		},
		Output: struct {
			LogLevel    string `yaml:"log-level"`
			LogAging    int    `yaml:"log-aging"`
			LogForceNew bool   `yaml:"log-force-new"`
			LogColorful *bool  `yaml:"log-colorful"`
			Debug       bool   `yaml:"debug"`
		}{
			LogLevel:    cfgs.OutputLogLevel,
			LogAging:    outlogLogAging,
			LogForceNew: param.EnsureBool(cfgs.OutputLogForceNew, true),
			LogColorful: &outlogColorfull,
			Debug:       param.EnsureBool(cfgs.OutputDebug, false),
		},
	}
	cfb.Database = make(map[string]yaml.Node)
	dbConf := &config2.LevelDBConfig{Enable: param.EnsureBool(cfgs.DatabasesLeveldbEnable, true)}
	cfb.Database["leveldb"] = func() yaml.Node {
		n := &yaml.Node{}
		_ = n.Encode(dbConf)
		return *n
	}()
	cfb.Database["cache"] = func() yaml.Node {
		n := &yaml.Node{}
		cache := struct {
			Image string `yaml:"image"`
			Video string `yaml:"video"`
		}{
			Image: cfgs.DatabaseCacheImage,
			Video: cfgs.DatabaseCacheVideo,
		}
		_ = n.Encode(cache)
		return *n
	}()
	cfb.Servers, err = parseServers(&cfgs)
	return cfb, err
}

func parseServers(cfg *config) ([]map[string]yaml.Node, error) {
	// 处理http的服务绑定
	var httpcfg serverHTTP
	err := json.Unmarshal([]byte(cfg.ServersHTTP), &httpcfg)
	if err != nil {
		return nil, errors.New("http的config解析错误")
	}
	var wscfg serverWs
	err = json.Unmarshal([]byte(cfg.ServersWs), &wscfg)
	if err != nil {
		return nil, errors.New("ws的config解析错误")
	}
	var wsrcfg serverWsReverse
	err = json.Unmarshal([]byte(cfg.ServersWsReverse), &wsrcfg)
	if err != nil {
		return nil, errors.New("ws-reverse的config解析错误")
	}
	var serverMap []map[string]yaml.Node
	serverMap = append(serverMap, parseServerHTTP(&httpcfg, cfg), parseServerWs(&wscfg, cfg), parseServerWsReverse(&wsrcfg, cfg))
	return serverMap, nil
}

// parseServerHTTP 生成yaml版本的http的配置
func parseServerHTTP(cfg *serverHTTP, c *config) map[string]yaml.Node {
	node := yaml.Node{}
	httpConf := &server.HTTPServer{
		Host: cfg.Host,
		Port: cfg.Port,
		MiddleWares: server.MiddleWares{
			AccessToken: c.DefaultMiddlewaresAccessToken,
		},
	}
	for _, post := range cfg.Post {
		if post.Disable {
			continue
		}
		httpConf.Post = append(httpConf.Post, struct {
			URL             string  `yaml:"url"`
			Secret          string  `yaml:"secret"`
			MaxRetries      *uint64 `yaml:"max-retries"`
			RetriesInterval *uint64 `yaml:"retries-interval"`
		}{URL: post.URL, Secret: post.Secret})
	}
	_ = node.Encode(httpConf)
	cfgMap := make(map[string]yaml.Node)
	if cfg.Enable {
		cfgMap["http"] = node
	}
	return cfgMap
}

func parseServerWs(cfg *serverWs, c *config) map[string]yaml.Node {
	node := yaml.Node{}
	wsConf := &server.WebsocketServer{
		Disabled: !cfg.Enable,
		Host:     cfg.Host,
		Port:     cfg.Port,
		MiddleWares: server.MiddleWares{
			AccessToken: c.DefaultMiddlewaresAccessToken,
		},
	}
	_ = node.Encode(wsConf)
	cfgMap := make(map[string]yaml.Node)
	if cfg.Enable {
		cfgMap["ws"] = node
	}
	return cfgMap
}

func parseServerWsReverse(cfg *serverWsReverse, c *config) map[string]yaml.Node {
	node := yaml.Node{}
	wsrConf := &server.WebsocketReverse{
		Disabled:          !cfg.Enable,
		Universal:         cfg.Universal,
		API:               cfg.API,
		Event:             cfg.Event,
		ReconnectInterval: cfg.ReconnectInterval,
		MiddleWares: server.MiddleWares{
			AccessToken: c.DefaultMiddlewaresAccessToken,
		},
	}
	_ = node.Encode(wsrConf)
	cfgMap := make(map[string]yaml.Node)
	if cfg.Enable {
		cfgMap["ws-reverse"] = node
	}
	return cfgMap
}

// Getbytekey 获取 密码加密的 bytekey
func Getbytekey() ([]byte, error) {
	var QQconfigs []QQConfig
	result := NewQqConfig().Find(&QQconfigs)
	if result.Error != nil {
		return nil, result.Error
	}
	configmap := make(map[string]string)
	for _, cfg := range QQconfigs {
		configmap[cfg.ConfigKey] = cfg.ConfigValue
	}
	bytes, _ := json.Marshal(configmap)
	var cfgs config
	_ = json.Unmarshal(bytes, &cfgs)
	return []byte(cfgs.AccountEncryptByteKey), nil
}

// Setbytekey 设置加密密码用的 bytekey
func Setbytekey(bytekey string) error {
	result := NewQqConfig().Where("key=?", "account_encrypt_bytekey").Update("value", bytekey)
	return result.Error
}
