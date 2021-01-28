package global

import (
	"os"
	"strconv"
	"time"

	"github.com/hjson/hjson-go"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

//DefaultConfigWithComments 为go-cqhttp的默认配置文件
var DefaultConfigWithComments = `
/*
    go-cqhttp 默认配置文件
*/

{
    // QQ号
    uin: 0
    // QQ密码
    password: ""
    // 是否启用密码加密
    encrypt_password: false
    // 加密后的密码, 如未启用密码加密将为空, 请勿随意修改.
    password_encrypted: ""
    // 是否启用内置数据库
    // 启用将会增加10-20MB的内存占用和一定的磁盘空间
    // 关闭将无法使用 撤回 回复 get_msg 等上下文相关功能
    enable_db: true
    // 访问密钥, 强烈推荐在公网的服务器设置
    access_token: ""
    // 重连设置
    relogin: {
        // 是否启用自动重连
        // 如不启用掉线后将不会自动重连
        enabled: true
        // 重连延迟, 单位秒
        relogin_delay: 3
        // 最大重连次数, 0为无限制
        max_relogin_times: 0
    }
    // API限速设置
    // 该设置为全局生效
    // 原 cqhttp 虽然启用了 rate_limit 后缀, 但是基本没插件适配
    // 目前该限速设置为令牌桶算法, 请参考: 
    // https://baike.baidu.com/item/%E4%BB%A4%E7%89%8C%E6%A1%B6%E7%AE%97%E6%B3%95/6597000?fr=aladdin
    _rate_limit: {
        // 是否启用限速
        enabled: false
        // 令牌回复频率, 单位秒
        frequency: 1
        // 令牌桶大小
        bucket_size: 1
    }
    // 是否忽略无效的CQ码
    // 如果为假将原样发送
    ignore_invalid_cqcode: false
    // 是否强制分片发送消息
    // 分片发送将会带来更快的速度
    // 但是兼容性会有些问题
    force_fragmented: false
    // 心跳频率, 单位秒
    // -1 为关闭心跳
    heartbeat_interval: 0
    // HTTP设置
    http_config: {
        // 是否启用正向HTTP服务器
        enabled: true
        // 服务端监听地址
        host: 0.0.0.0
        // 服务端监听端口
        port: 5700
        // 反向HTTP超时时间, 单位秒
        // 最小值为5，小于5将会忽略本项设置
        timeout: 0
        // 反向HTTP POST地址列表
        // 格式: 
        // {
        //    地址: secret
        // }
        post_urls: {}
    }
    // 正向WS设置
    ws_config: {
        // 是否启用正向WS服务器
        enabled: true
        // 正向WS服务器监听地址
        host: 0.0.0.0
        // 正向WS服务器监听端口
        port: 6700
    }
    // 反向WS设置
    ws_reverse_servers: [
        // 可以添加多个反向WS推送
        {
            // 是否启用该推送
            enabled: false
            // 反向WS Universal 地址
            // 注意 设置了此项地址后下面两项将会被忽略
            // 留空请使用 ""
            reverse_url: ws://you_websocket_universal.server
            // 反向WS API 地址
            reverse_api_url: ws://you_websocket_api.server
            // 反向WS Event 地址
            reverse_event_url: ws://you_websocket_event.server
            // 重连间隔 单位毫秒
            reverse_reconnect_interval: 3000
        }
    ]
    // 上报数据类型
    // 可选: string array
    post_message_format: string
    // 是否使用服务器下发的新地址进行重连
    // 注意, 此设置可能导致在海外服务器上连接情况更差
    use_sso_address: false
    // 是否启用 DEBUG
    debug: false
    // 日志等级 trace,debug,info,warn,error
    log_level: "info"
    // WebUi 设置
    web_ui: {
        // 是否启用 WebUi
        enabled: true
        // 监听地址
        host: 127.0.0.1
        // 监听端口
        web_ui_port: 9999
        // 是否接收来自web的输入
        web_input: false
    }
}
`

//PasswordHash 存储QQ密码哈希供登录使用
var PasswordHash [16]byte

//JSONConfig Config对应的结构体
type JSONConfig struct {
	Uin               int64  `json:"uin"`
	Password          string `json:"password"`
	EncryptPassword   bool   `json:"encrypt_password"`
	PasswordEncrypted string `json:"password_encrypted"`
	EnableDB          bool   `json:"enable_db"`
	AccessToken       string `json:"access_token"`
	ReLogin           struct {
		Enabled         bool `json:"enabled"`
		ReLoginDelay    int  `json:"relogin_delay"`
		MaxReloginTimes uint `json:"max_relogin_times"`
	} `json:"relogin"`
	RateLimit struct {
		Enabled    bool    `json:"enabled"`
		Frequency  float64 `json:"frequency"`
		BucketSize int     `json:"bucket_size"`
	} `json:"_rate_limit"`
	IgnoreInvalidCQCode bool                          `json:"ignore_invalid_cqcode"`
	ForceFragmented     bool                          `json:"force_fragmented"`
	FixURL              bool                          `json:"fix_url"`
	ProxyRewrite        string                        `json:"proxy_rewrite"`
	HeartbeatInterval   time.Duration                 `json:"heartbeat_interval"`
	HTTPConfig          *GoCQHTTPConfig               `json:"http_config"`
	WSConfig            *GoCQWebSocketConfig          `json:"ws_config"`
	ReverseServers      []*GoCQReverseWebSocketConfig `json:"ws_reverse_servers"`
	PostMessageFormat   string                        `json:"post_message_format"`
	UseSSOAddress       bool                          `json:"use_sso_address"`
	Debug               bool                          `json:"debug"`
	LogLevel            string                        `json:"log_level"`
	WebUI               *GoCQWebUI                    `json:"web_ui"`
}

//CQHTTPAPIConfig HTTPAPI对应的Config结构体
type CQHTTPAPIConfig struct {
	Host                         string `json:"host"`
	Port                         uint16 `json:"port"`
	UseHTTP                      bool   `json:"use_http"`
	WSHost                       string `json:"ws_host"`
	WSPort                       uint16 `json:"ws_port"`
	UseWS                        bool   `json:"use_ws"`
	WSReverseURL                 string `json:"ws_reverse_url"`
	WSReverseAPIURL              string `json:"ws_reverse_api_url"`
	WSReverseEventURL            string `json:"ws_reverse_event_url"`
	WSReverseReconnectInterval   uint16 `json:"ws_reverse_reconnect_interval"`
	WSReverseReconnectOnCode1000 bool   `json:"ws_reverse_reconnect_on_code_1000"`
	UseWsReverse                 bool   `json:"use_ws_reverse"`
	PostURL                      string `json:"post_url"`
	AccessToken                  string `json:"access_token"`
	Secret                       string `json:"secret"`
	PostMessageFormat            string `json:"post_message_format"`
}

//GoCQHTTPConfig 正向HTTP对应config结构体
type GoCQHTTPConfig struct {
	Enabled  bool              `json:"enabled"`
	Host     string            `json:"host"`
	Port     uint16            `json:"port"`
	Timeout  int32             `json:"timeout"`
	PostUrls map[string]string `json:"post_urls"`
}

//GoCQWebSocketConfig 正向WebSocket对应Config结构体
type GoCQWebSocketConfig struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    uint16 `json:"port"`
}

//GoCQReverseWebSocketConfig 反向WebSocket对应Config结构体
type GoCQReverseWebSocketConfig struct {
	Enabled                  bool   `json:"enabled"`
	ReverseURL               string `json:"reverse_url"`
	ReverseAPIURL            string `json:"reverse_api_url"`
	ReverseEventURL          string `json:"reverse_event_url"`
	ReverseReconnectInterval uint16 `json:"reverse_reconnect_interval"`
}

//GoCQWebUI WebUI对应Config结构体
type GoCQWebUI struct {
	Enabled   bool   `json:"enabled"`
	Host      string `json:"host"`
	WebUIPort uint64 `json:"web_ui_port"`
	WebInput  bool   `json:"web_input"`
}

//DefaultConfig 返回一份默认配置对应结构体
func DefaultConfig() *JSONConfig {
	return &JSONConfig{
		EnableDB: true,
		ReLogin: struct {
			Enabled         bool `json:"enabled"`
			ReLoginDelay    int  `json:"relogin_delay"`
			MaxReloginTimes uint `json:"max_relogin_times"`
		}{
			Enabled:         true,
			ReLoginDelay:    3,
			MaxReloginTimes: 0,
		},
		RateLimit: struct {
			Enabled    bool    `json:"enabled"`
			Frequency  float64 `json:"frequency"`
			BucketSize int     `json:"bucket_size"`
		}{
			Enabled:    false,
			Frequency:  1,
			BucketSize: 1,
		},
		PostMessageFormat: "string",
		ForceFragmented:   false,
		HTTPConfig: &GoCQHTTPConfig{
			Enabled:  true,
			Host:     "0.0.0.0",
			Port:     5700,
			PostUrls: map[string]string{},
		},
		WSConfig: &GoCQWebSocketConfig{
			Enabled: true,
			Host:    "0.0.0.0",
			Port:    6700,
		},
		ReverseServers: []*GoCQReverseWebSocketConfig{
			{
				Enabled:                  false,
				ReverseURL:               "ws://you_websocket_universal.server",
				ReverseAPIURL:            "ws://you_websocket_api.server",
				ReverseEventURL:          "ws://you_websocket_event.server",
				ReverseReconnectInterval: 3000,
			},
		},
		WebUI: &GoCQWebUI{
			Enabled:   true,
			Host:      "127.0.0.1",
			WebInput:  false,
			WebUIPort: 9999,
		},
	}
}

//LoadConfig 加载配置文件
func LoadConfig(p string) *JSONConfig {
	if !PathExists(p) {
		log.Warnf("尝试加载配置文件 %v 失败: 文件不存在", p)
		return nil
	}
	var dat map[string]interface{}
	var c = JSONConfig{}
	err := hjson.Unmarshal([]byte(ReadAllText(p)), &dat)
	if err == nil {
		b, _ := json.Marshal(dat)
		err = json.Unmarshal(b, &c)
	}
	if err != nil {
		log.Warnf("尝试加载配置文件 %v 时出现错误: %v", p, err)
		log.Infoln("原文件已备份")
		_ = os.Rename(p, p+".backup"+strconv.FormatInt(time.Now().Unix(), 10))
		return nil
	}
	return &c
}

//Save 写入配置文件至path
func (c *JSONConfig) Save(path string) error {
	data, err := hjson.MarshalWithOptions(c, hjson.EncoderOptions{
		Eol:            "\n",
		BracesSameLine: true,
		IndentBy:       "    ",
	})
	if err != nil {
		return err
	}
	return WriteAllText(path, string(data))
}
