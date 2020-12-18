# 配置

go-cqhttp 包含 `config.hjson` 和 `device.json` 两个配置文件, 其中 `config.json` 为运行配置 `device.json` 为虚拟设备信息.

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
	"relogin": {
		"enabled": true,
		"relogin_delay": 3,
		"max_relogin_times": 0
	},
    "_rate_limit": {
		"enabled": false,
		"frequency": 1,
		"bucket_size": 1
    },
	"post_message_format": "string",
	"ignore_invalid_cqcode": false,
	"force_fragmented": true,
	"heartbeat_interval": 5,
    "use_sso_address": false,
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

| 字段                  | 类型     | 说明                                                                                     |
| --------------------- | -------- | ---------------------------------------------------------------------------------------- |
| uin                   | int64    | 登录用QQ号                                                                               |
| password              | string   | 登录用密码                                                                               |
| encrypt_password      | bool     | 是否对密码进行加密.                                                                      |
| password_encrypted    | string   | 加密后的密码(请勿修改)                                                                   |
| enable_db             | bool     | 是否开启内置数据库, 关闭后将无法使用 **回复/撤回** 等上下文相关接口                      |
| access_token          | string   | 同CQHTTP的 `access_token`  用于身份验证                                                  |
| relogin               | bool     | 是否自动重新登录                                                                         |
| relogin_delay         | int      | 重登录延时（秒）                                                                         |
| max_relogin_times     | uint     | 最大重登录次数，若0则不设置上限                                                          |
| _rate_limit           | bool     | 是否启用API调用限速                                                                      |
| frequency             | float64  | 1s内能调用API的次数                                                                      |
| bucket_size           | int      | 令牌桶的大小，默认为1，修改此值可允许一定程度内连续调用api                               |
| post_message_format   | string   | 上报信息类型                                                                             |
| ignore_invalid_cqcode | bool     | 是否忽略错误的CQ码                                                                       |
| force_fragmented      | bool     | 是否强制分片发送群长消息                                                                 |
| fix_url               | bool     | 是否对链接的发送进行预处理, 可缓解链接信息被风控导致无法发送的情况, 但可能影响客户端着色(不影响内容)|
| use_sso_address       | bool     | 是否使用服务器下发的地址                                                                 |
| heartbeat_interval    | int64    | 心跳间隔时间，单位秒。小于0则关闭心跳，等于0使用默认值(5秒)                              |
| http_config           | object   | HTTP API配置                                                                             |
| ws_config             | object   | Websocket API 配置                                                                       |
| ws_reverse_servers    | object[] | 反向 Websocket API 配置                                                                  |
| log_level             | string   | 指定日志收集级别，将收集的日志单独存放到固定文件中，便于查看日志线索 当前支持 warn,error |

> 注: 开启密码加密后程序将在每次启动时要求输入解密密钥, 密钥错误会导致登录时提示密码错误.
> 解密后密码将储存在内存中，用于自动重连等功能. 所以此加密并不能防止内存读取.
> 解密密钥在使用完成后并不会留存在内存中, 所以可用相对简单的字符串作为密钥

> 注2: 分片发送为原酷Q发送长消息的老方案, 发送速度更优/兼容性更好，但在有发言频率限制的群里，可能无法发送。关闭后将优先使用新方案, 能发送更长的消息, 但发送速度更慢，在部分老客户端将无法解析.

> 注3：关闭心跳服务可能引起断线，请谨慎关闭

## 设备信息

默认生成的设备信息如下所示: 

``` json
{
	"protocol": 0,
	"display": "xxx",
	"finger_print": "xxx",
	"boot_id": "xxx",
	"proc_version": "xxx",
	"imei": "xxx"
}
```

在大部分情况下 我们只需要关心 `protocol` 字段: 

| 值  | 类型          | 限制                                                             |
| --- | ------------- | ---------------------------------------------------------------- |
| 0   | iPad          | 无                                                               |
| 1   | Android Phone | 无                                                               |
| 2   | Android Watch | 无法接收 `notify` 事件、无法接收口令红包、无法接收撤回消息            |
| 3   | MacOS         | 无                                                               |

> 注意, 根据协议的不同, 各类消息有所限制

## 自定义服务器IP

> 某些海外服务器使用默认地址可能会存在链路问题，此功能可以指定 go-cqhttp 连接哪些地址以达到最优化.

将文件 `address.txt` 创建到 `go-cqhttp` 工作目录, 并键入 `IP:PORT` 以换行符为分割即可.

示例:
````
1.1.1.1:53
1.1.2.2:8899
````