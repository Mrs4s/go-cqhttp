# 管理 API

> 支持跨域

## 公共参数

参数:

| 参数名       | 类型   | 说明                        |
| ------------ | ------ | --------------------------- |
| access_token | string | 校验口令，config.hjson中配置 |



## admin/do_restart

### 热重启

> 热重启

> ps: 目前不支持ws部分的修改生效

method：`POST/GET`

参数:

| 参数名 | 类型 | 说明 |
| ------ | ---- | ---- |
| 无     |      |      |

返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```


### admin/get_web_write

> 拉取验证码/设备锁

method: `GET`


参数:

| 参数名 | 类型 | 说明 |
| ------ | ---- | ---- |
| 无     |      |      |

返回：

```json
{"data": {"ispic": true,"picbase64":"xxxxx"}, "retcode": 0, "status": "ok"}
```
| 参数名   | 类型   | 说明                                                |
| -------- | ------ | --------------------------------------------------- |
| ispic    | bool   | 是否是验证码类型 true是，false为不是（比如设备锁    |
| picbas64 | string | 验证码的base64编码内容，加上头，放入img标签即可显示 |

### admin/do_web_write

> web输入验证码/设备锁确认

method: `POST` formdata


参数:

| 参数名 | 类型   | 说明       |
| ------ | ------ | ---------- |
| input  | string | 输入的内容 |

返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```


### admin/do_restart_docker

> 冷重启

> 注意：此api 会直接结束掉进程，需要依赖docker/supervisor等进程管理工具来自动拉起

method: `POST`


参数:

| 参数名 | 类型 | 说明 |
| ------ | ---- | ---- |
| 无     |      |      |

返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```

### admin/do_process_restart

> 冷重启

method: `POST`


参数:

| 参数名 | 类型 | 说明 |
| ------ | ---- | ---- |
| 无     |      |      |

返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```

### admin/do_config_base

> 基础配置

method: `POST` formdata


参数:

| 参数名       | 类型   | 说明                                  |
| ------------ | ------ | ------------------------------------- |
| uin          | string | qq号                                  |
| password     | string | qq密码                                |
| enable_db    | string | 是否启动数据库,填 'true' 或者 'false' |
| access_token | string | 授权 token                            |

返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```


### admin/do_config_http

> http服务配置

method: `POST` formdata

参数:

| 参数名      | 类型   | 说明                                          |
| ----------- | ------ | --------------------------------------------- |
| port        | string | 服务端口                                      |
| host        | string | 服务监听地址                                  |
| enable      | string | 是否启用 ,填 'true' 或者 'false'              |
| timeout     | string | http请求超时时间                              |
| post_url    | string | post上报地址 不需要就填空字符串，或者不填     |
| post_secret | string | post上报的secret 不需要就填空字符串，或者不填 |

返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```


### admin/do_config_ws

> 正向ws设置

method: `POST` formdata

参数:

| 参数名 | 类型   | 说明                             |
| ------ | ------ | -------------------------------- |
| port   | string | 服务端口                         |
| host   | string | 服务监听地址                     |
| enable | string | 是否启用 ,填 'true' 或者 'false' |


返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```

### admin/do_config_reverse

> 反向ws配置

method: `POST` formdata

参数:

| 参数名 | 类型   | 说明                             |
| ------ | ------ | -------------------------------- |
| port   | string | 服务端口                         |
| host   | string | 服务监听地址                     |
| enable | string | 是否启用 ,填 'true' 或者 'false' |


返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```

### admin/do_config_json

> 直接修改 config.hjson配置

method: `POST` formdata

参数:

| 参数名 | 类型   | 说明                                |
| ------ | ------ | ----------------------------------- |
| json   | string | 完整的config.hjson的配合，json字符串 |


返回：

```json
{"data": {}, "retcode": 0, "status": "ok"}
```

### admin/get_config_json

> 获取当前 config.hjson配置

method: `GET`

参数:

| 参数名 | 类型 | 说明 |
| ------ | ---- | ---- |
| 无     |      |      |


返回：

```json
{"data": {"config":"xxxx"}, "retcode": 0, "status": "ok"}
```

| 参数名 | 类型   | 说明                                |
| ------ | ------ | ----------------------------------- |
| config | string | 完整的config.hjson的配合，json字符串 |

