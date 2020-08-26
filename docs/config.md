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
	"encrypt_password": false,
	"password_encrypted": "",
	"enable_db": true,
	"access_token": "",
	"relogin": false,
	"relogin_delay": 0,
	"post_message_format": "string",
	"ignore_invalid_cqcode": false,
	"force_fragmented": true,
	"http_config": {
		"enabled": true,
		"host": "0.0.0.0",
		"port": 5700,
		"timeout": 5,
		"post_urls": {"url:port": "secret"}
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
| uin                  | int64    | 登录用QQ号                                                          |
| password             | string   | 登录用密码                                                          |
| encrypt_password     | bool     | 是否对密码进行加密.                                                  |
| password_encrypted   | string   | 加密后的密码(请勿修改)                                                |
| enable_db            | bool     | 是否开启内置数据库, 关闭后将无法使用 **回复/撤回** 等上下文相关接口      |
| access_token         | string   | 同CQHTTP的 `access_token`  用于身份验证                             |
| relogin              | bool     | 是否自动重新登录                                                    |
| relogin_delay        | int      | 重登录延时（秒）                                                    |
| post_message_format  | string   | 上报信息类型                                                       |
| ignore_invalid_cqcode| bool     | 是否忽略错误的CQ码                                                  |
| force_fragmented     | bool     | 是否强制分片发送群长消息                                              |
| http_config          | object   | HTTP API配置                                                        |
| ws_config            | object   | Websocket API 配置                                                  |
| ws_reverse_servers   | object[] | 反向 Websocket API 配置                                             |

> 注: 开启密码加密后程序将在每次启动时要求输入解密密钥, 密钥错误会导致登录时提示密码错误.
> 解密后密码将储存在内存中，用于自动重连等功能. 所以此加密并不能防止内存读取.
> 解密密钥在使用完成后并不会留存在内存中, 所以可用相对简单的字符串作为密钥

> 注2: 分片发送为原酷Q发送长消息的老方案, 发送速度更优/兼容性更好。关闭后将优先使用新方案, 能发送更长的消息, 但发送速度更慢，在部分老客户端将无法解析.