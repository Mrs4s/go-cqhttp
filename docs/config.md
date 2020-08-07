# 配置

go-cqhttp 包含 `config.json` 和 `device.json` 两个配置文件, 其中 `config.json` 为运行配置 `device.json` 为虚拟设备信息.

## 从原CQHTTP导入配置

go-cqhttp 支持导入CQHTTP的配置文件, 具体步骤为: 

1. 找到CQHTTP原配置文件 `{CQ工作目录}/app/io.github.richardchien.coolqhttpapi/config/{qq号}.json`
2. 将文件复制到go-cqhttp根目录并重命名为 `cqhttp.json`
3. 重启go-cqhttp后将自动导入配置

## 配置信息

默认生成的配置文件如下所示: 

````json
{
	"uin": 0,
	"password": "",
	"enable_db": true,
	"access_token": "",
	"relogin": false,
	"relogin_delay": 0,
	"http_config": {
		"enabled": true,
		"host": "0.0.0.0",
		"port": 5700,
		"post_urls": {}
	},
	"ws_config": {
		"enabled": true,
		"host": "0.0.0.0",
		"port": 6700
	},
	"ws_reverse_servers": [
		{
			"enabled": false,
			"reverse_url": "ws://you_websocket_universal.server",
			"reverse_api_url": "ws://you_websocket_api.server",
			"reverse_event_url": "ws://you_websocket_event.server",
			"reverse_reconnect_interval": 3000
		}
	]
}
````

| 字段               | 类型     | 说明                                                                |
| ------------------ | -------- | ------------------------------------------------------------------- |
| uin                | int64    | 登录用QQ号                                                          |
| password           | string   | 登录用密码                                                          |
| enable_db          | bool     | 是否开启内置数据库, 关闭后将无法使用 **回复/撤回** 等上下文相关接口 |
| access_token       | string   | 同CQHTTP的 `access_token`  用于身份验证                             |
| relogin            | bool     | 是否自动重新登录                                                    |
| relogin_delay      | int      | 重登录延时（秒）                                                    |
| http_config        | object   | HTTP API配置                                                        |
| ws_config          | object   | Websocket API 配置                                                  |
| ws_reverse_servers | object[] | 反向 Websocket API 配置                                             |


