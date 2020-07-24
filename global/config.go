package global

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

type JsonConfig struct {
	Uin            int64                `json:"uin"`
	Password       string               `json:"password"`
	EnableDB       bool                 `json:"enable_db"`
	AccessToken    string               `json:"access_token"`
	Reconnect      bool                 `json:"reconnect"`
	ReconnectDelay int                  `json:"reconnect_delay"`
	HttpConfig     *GoCQHttpConfig      `json:"http_config"`
	WSConfig       *GoCQWebsocketConfig `json:"ws_config"`
}

type CQHttpApiConfig struct {
	Host                         string `json:"host"`
	Port                         uint16 `json:"port"`
	UseHttp                      bool   `json:"use_http"`
	WSHost                       string `json:"ws_host"`
	WSPort                       uint16 `json:"ws_port"`
	UseWS                        bool   `json:"use_ws"`
	WSReverseUrl                 string `json:"ws_reverse_url"`
	WSReverseApiUrl              string `json:"ws_reverse_api_url"`
	WSReverseEventUrl            string `json:"ws_reverse_event_url"`
	WSReverseReconnectInterval   uint16 `json:"ws_reverse_reconnect_interval"`
	WSReverseReconnectOnCode1000 bool   `json:"ws_reverse_reconnect_on_code_1000"`
	UseWsReverse                 bool   `json:"use_ws_reverse"`
	PostUrl                      string `json:"post_url"`
	AccessToken                  string `json:"access_token"`
	Secret                       string `json:"secret"`
	PostMessageFormat            string `json:"post_message_format"`
}

type GoCQHttpConfig struct {
	Enabled  bool              `json:"enabled"`
	Host     string            `json:"host"`
	Port     uint16            `json:"port"`
	PostUrls map[string]string `json:"post_urls"`
}

type GoCQWebsocketConfig struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    uint16 `json:"port"`
}

func DefaultConfig() *JsonConfig {
	return &JsonConfig{
		EnableDB: true,
		HttpConfig: &GoCQHttpConfig{
			Enabled:  true,
			Host:     "0.0.0.0",
			Port:     5700,
			PostUrls: map[string]string{},
		},
		WSConfig: &GoCQWebsocketConfig{
			Enabled: true,
			Host:    "0.0.0.0",
			Port:    6700,
		},
	}
}

func Load(p string) *JsonConfig {
	if !PathExists(p) {
		log.Warnf("尝试加载配置文件 %v 失败: 文件不存在", p)
		return nil
	}
	c := JsonConfig{}
	err := json.Unmarshal([]byte(ReadAllText(p)), &c)
	if err != nil {
		log.Warnf("尝试加载配置文件 %v 时出现错误: %v", p, err)
		return nil
	}
	return &c
}

func (c *JsonConfig) Save(p string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	WriteAllText(p, string(data))
	return nil
}
