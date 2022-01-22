# 说明

完全放弃命令行的登录方式，采用全web登录方式。

#### 添加了web控制台。管理和查看状态更方便了。

### ps: 配置信息里面，记得自行修改token。以及管理admin的密码。

## 默认端口`8080`。默认用户名 `admin`，默认密码 `admin`

## 使用说明

+ 切换说明
  > 默认启动并不会启动该模块。该模块启动方式（二选一）：
  >
  > + 启动时添加参数: `gocqhttp -webui`
  >
  > + 添加环境变量: `GOCQWEBUI=1`
  >
  > + ps: 要启用该模块，需要在编译的时候启动cgo+gcc。不使用该模块的话，关闭cgo，直接用交叉编译就行了

+ docker-compose方式

> 找到项目内的 `docker-compose.yaml` 文件。放入到你的服务器上去 例如 `/home/scjtqs/qq/docker-compose.yaml`
>
> `cd /home/scjtqs/qq` 进去当前目录。
>
> `docker-compose up -d`启动项目。这样就可以了。浏览器打开 http://服务器ip:9999 进行登录，默认的用户密码: `admin` `admin`
>

+ 自己编译使用：

> 编译需要开启cgo。用到了sqlite，来做后台的sql存储。其他的参考docker的方式的编译和启动。
>

## 功能

- [x] 一套后台管理系统+web方式登录qq
- [x] 配置账号密码进行登录
- [x] qr二维码扫码登录
- [x] web控制配置信息
- [x] 好友列表+操作按钮
- [x] 群列表+操作按钮
- [x] web日志查看
- [x] web消息发送（cq码调试）
- [x] 聊天记录列表查看
- [ ] ...
