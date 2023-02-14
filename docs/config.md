# 配置

> 注意, 最新文档已经移动到 [go-cqhttp-docs](https://github.com/ishkong/go-cqhttp-docs), 当前文档只做兼容性保留, 所以内容可能有不足.

go-cqhttp 包含 `config.yml` 和 `device.json` 两个配置文件, 其中 `config.yml` 为运行配置 `device.json` 为虚拟设备信息.

## 配置信息

go-cqhttp 的配置文件采用 YAML , 在使用之前希望你能了解 YAML 的语法([教程](https://www.runoob.com/w3cnote/yaml-intro.html))

默认生成的配置文件如下所示:

````yaml
# go-cqhttp 默认配置文件

account: # 账号相关
  uin: 1233456 # QQ账号
  password: '' # 密码为空时使用扫码登录
  encrypt: false  # 是否开启密码加密
  status: 0      # 在线状态 请参考 https://docs.go-cqhttp.org/guide/config.html#在线状态
  relogin: # 重连设置
    delay: 3   # 首次重连延迟, 单位秒
    interval: 3   # 重连间隔
    max-times: 0  # 最大重连次数, 0为无限制

  # 是否使用服务器下发的新地址进行重连
  # 注意, 此设置可能导致在海外服务器上连接情况更差
  use-sso-address: true

heartbeat:
  # 心跳频率, 单位秒
  # -1 为关闭心跳
  interval: 5

message:
  # 上报数据类型
  # 可选: string,array
  post-format: string
  # 是否忽略无效的CQ码, 如果为假将原样发送
  ignore-invalid-cqcode: false
  # 是否强制分片发送消息
  # 分片发送将会带来更快的速度
  # 但是兼容性会有些问题
  force-fragment: false
  # 是否将url分片发送
  fix-url: false
  # 下载图片等请求网络代理
  proxy-rewrite: ''
  # 是否上报自身消息
  report-self-message: false
  # 移除服务端的Reply附带的At
  remove-reply-at: false
  # 为Reply附加更多信息
  extra-reply-data: false
  # 跳过 Mime 扫描, 忽略错误数据
  skip-mime-scan: false

output:
  # 日志等级 trace,debug,info,warn,error
  log-level: warn
  # 日志时效 单位天. 超过这个时间之前的日志将会被自动删除. 设置为 0 表示永久保留.
  log-aging: 15
  # 是否在每次启动时强制创建全新的文件储存日志. 为 false 的情况下将会在上次启动时创建的日志文件续写
  log-force-new: true
  # 是否启用 DEBUG
  debug: false # 开启调试模式

# 默认中间件锚点
default-middlewares: &default
  # 访问密钥, 强烈推荐在公网的服务器设置
  access-token: ''
  # 事件过滤器文件目录
  filter: ''
  # API限速设置
  # 该设置为全局生效
  # 原 cqhttp 虽然启用了 rate_limit 后缀, 但是基本没插件适配
  # 目前该限速设置为令牌桶算法, 请参考:
  # https://baike.baidu.com/item/%E4%BB%A4%E7%89%8C%E6%A1%B6%E7%AE%97%E6%B3%95/6597000?fr=aladdin
  rate-limit:
    enabled: false # 是否启用限速
    frequency: 1  # 令牌回复频率, 单位秒
    bucket: 1     # 令牌桶大小

# 连接服务列表
servers:
  # HTTP 通信设置
  - http:
      # 服务端监听地址
      # 如需指定监听ipv4， 可使用 `address: tcp4://0.0.0.0:5700` (ipv6同理)
      address: 0.0.0.0:5700
      # 反向HTTP超时时间, 单位秒
      # 最小值为5，小于5将会忽略本项设置
      timeout: 5
      middlewares:
        <<: *default # 引用默认中间件
      # 反向HTTP POST地址列表
      post:
      #- url: '' # 地址
      #  secret: ''           # 密钥
      #- url: 127.0.0.1:5701 # 地址
      #  secret: ''          # 密钥

  # 正向WS设置
  - ws:
      # 正向WS服务器监听地址
      # 如需指定监听ipv4， 可使用 `address: tcp4://0.0.0.0:6700` (ipv6同理)
      address: 0.0.0.0:6700
      middlewares:
        <<: *default # 引用默认中间件

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
  # pprof 性能分析服务器, 一般情况下不需要启用.
  # 如果遇到性能问题请上传报告给开发者处理
  # 注意: pprof服务不支持中间件、不支持鉴权. 请不要开放到公网
  - pprof:
      # pprof服务器监听地址
      host: 127.0.0.1
      # pprof服务器监听端口
      port: 7700
      
  # LambdaServer 配置
  - lambda:
      type: scf # 可用 scf,aws (aws未经过测试)
      middlewares:
        <<: *default # 引用默认中间件

  # 可添加更多
  #- ws-reverse:
  #- ws:
  #- http:

database: # 数据库相关设置
  leveldb:
    # 是否启用内置leveldb数据库
    # 启用将会增加10-20MB的内存占用和一定的磁盘空间
    # 关闭将无法使用 撤回 回复 get_msg 等上下文相关功能
    enable: true
````

> 注1: 开启密码加密后程序将在每次启动时要求输入解密密钥, 密钥错误会导致登录时提示密码错误.
> 解密后密码的哈希将储存在内存中，用于自动重连等功能. 所以此加密并不能防止内存读取.
> 解密密钥在使用完成后并不会留存在内存中, 所以可用相对简单的字符串作为密钥

> 注2: 对于不需要的通信方式，你可以使用注释将其停用(推荐)，或者添加配置 `disabled: true` 将其关闭

> 注3: 分片发送为原酷Q发送长消息的老方案, 发送速度更优/兼容性更好，但在有发言频率限制的群里，可能无法发送。关闭后将优先使用新方案, 能发送更长的消息, 但发送速度更慢，在部分老客户端将无法解析.

> 注4：关闭心跳服务可能引起断线，请谨慎关闭

> 注5：关于MIME扫描， 详见[MIME](file.md#MIME)

### 环境变量

go-cqhttp 配置文件可以使用占位符来读取**环境变量**的值。

```yaml
account: # 账号相关
  uin: ${CQ_UIN} # 读取环境变量 CQ_UIN
  password: ${CQ_PWD:123456} # 当 CQ_PWD 为空时使用默认值 123456
```

## 在线状态

| 状态 | 值 |
| -----|----|
| 在线  |  0 |
| 离开 | 1 |
| 隐身 | 2 |
| 忙 | 3 |
| 听歌中 | 4 |
| 星座运势 | 5 |
| 今日天气 | 6 |
| 遇见春天 | 7 |
| Timi中 | 8 |
| 吃鸡中 | 9 |
| 恋爱中 | 10 |
| 汪汪汪 | 11 |
| 干饭中 | 12 |
| 学习中 | 13 |
| 熬夜中 | 14 |
| 打球中 | 15 |
| 信号弱 | 16 |
| 在线学习 | 17 |
| 游戏中 | 18 |
| 度假中 | 19 |
| 追剧中 | 20 |
| 健身中 | 21 |

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
| 4   | 企点           | 只能登录企点账号或企点子账号                                        |

> 注意, 根据协议的不同, 各类消息有所限制

## 自定义服务器IP

> 某些海外服务器使用默认地址可能会存在链路问题，此功能可以指定 go-cqhttp 连接哪些地址以达到最优化.

将文件 `address.txt` 创建到 `go-cqhttp` 工作目录, 并键入 `IP:PORT` 以换行符为分割即可.

示例:

````
1.1.1.1:53
1.1.2.2:8899
````

## 云函数部署

使用CustomRuntime进行部署， bootstrap 文件在 `scripts/bootstrap` 中已给出。
在部署前，请在本地完成登录，并将 `config.yml` ， `device.json` ，`bootstrap` 和 `go-cqhttp`
一起打包。

在触发器中创建一个API网关触发器，并启用集成响应，创建完成后即可通过api网关访问go-cqhttp(建议配置 AccessToken)。

> scripts/bootstrap 中使用的工作路径为 /tmp, 这个目录最大能容下500M文件, 如需长期使用，
> 请挂载文件存储(CFS).
