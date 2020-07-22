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
	"http_config": {
		"enabled": true,
		"host": "0.0.0.0",
		"port": 5700
	},
	"ws_config": {
		"enabled": true,
		"host": "0.0.0.0",
		"port": 6700
	}
}
````

| 字段         | 类型   | 说明                                                         |
| ------------ | ------ | ------------------------------------------------------------ |
| uin          | int64  | 登录用QQ号                                                   |
| password     | string | 登录用密码                                                   |
| enable_db    | bool   | 是否开启内置数据库, 关闭后将无法使用 **回复/撤回** 等上下文相关接口 |
| access_token | string | 同CQHTTP的 `access_token`  用于身份验证                      |
| http_config  | object | HTTP API配置                                                 |
| ws_config    | object | Websocket API 配置                                           |

