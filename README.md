# go-cqhttp
使用 [mirai](https://github.com/mamoe/mirai) 以及 [MiraiGo](https://github.com/Mrs4s/MiraiGo) 开发的cqhttp golang原生实现, 并在[cqhttp原版](https://github.com/richardchien/coolq-http-api)的基础上做了部分修改和拓展.
文档暂时可查看 `docs` 目录， 目前还在撰写中.

测试版可前往 Release 下载

# 兼容性

#### 接口
- [x] HTTP API
- [x] 反向HTTP POST
- [x] 正向Websocket
- [x] 反向Websocket

#### 拓展支持
> 拓展API可前往 [文档](docs/cqhttp.md) 查看
- [x] HTTP POST多点上报
- [x] 反向WS多点连接 
- [x] 修改群名
- [x] 消息撤回事件
- [x] 解析/发送 回复消息
- [x] 解析/发送 合并转发
- [ ] 使用代理请求网络图片

#### 实现
<details>
<summary>已实现CQ码</summary>

- [CQ:image]
- [CQ:record]
- [CQ:video]
- [CQ:face]
- [CQ:at]
- [CQ:share]
- [CQ:reply]
- [CQ:forward]
- [CQ:node]

</details>

<details>
<summary>已实现API</summary>

##### 注意: 部分API实现与CQHTTP原版略有差异，请参考文档
| API                      | 功能                                                         |
| ------------------------ | ------------------------------------------------------------ |
| /get_login_info          | [获取登录号信息](https://cqhttp.cc/docs/4.15/#/API?id=get_login_info-获取登录号信息) |
| /get_friend_list         | [获取好友列表](https://cqhttp.cc/docs/4.15/#/API?id=get_friend_list-获取好友列表) |
| /get_group_list          | [获取群列表](https://cqhttp.cc/docs/4.15/#/API?id=get_group_list-获取群列表) |
| /get_group_info          | [获取群信息](https://cqhttp.cc/docs/4.15/#/API?id=get_group_info-获取群信息) |
| /get_group_member_list   | [获取群成员列表](https://cqhttp.cc/docs/4.15/#/API?id=get_group_member_list-获取群成员列表) |
| /get_group_member_info   | [获取群成员信息](https://cqhttp.cc/docs/4.15/#/API?id=get_group_member_info-获取群成员信息) |
| /send_msg                | [发送消息](https://cqhttp.cc/docs/4.15/#/API?id=send_msg-发送消息) |
| /send_group_msg          | [发送群消息](https://cqhttp.cc/docs/4.15/#/API?id=send_group_msg-发送群消息) |
| /send_private_msg        | [发送私聊消息](https://cqhttp.cc/docs/4.15/#/API?id=send_private_msg-发送私聊消息) |
| /delete_msg              | [撤回信息](https://cqhttp.cc/docs/4.15/#/API?id=delete_msg-撤回消息) |
| /set_friend_add_request  | [处理加好友请求](https://cqhttp.cc/docs/4.15/#/API?id=set_friend_add_request-处理加好友请求) |
| /set_group_add_request   | [处理加群请求/邀请](https://cqhttp.cc/docs/4.15/#/API?id=set_group_add_request-处理加群请求／邀请) |
| /set_group_card          | [设置群名片(群备注)](https://cqhttp.cc/docs/4.15/#/API?id=set_group_card-设置群名片（群备注）) |
| /set_group_special_title | [设置群组专属头衔](https://cqhttp.cc/docs/4.15/#/API?id=set_group_special_title-设置群组专属头衔) |
| /set_group_kick          | [群组T人](https://cqhttp.cc/docs/4.15/#/API?id=set_group_kick-群组踢人) |
| /set_group_ban           | [群组单人禁言](https://cqhttp.cc/docs/4.15/#/API?id=set_group_ban-群组单人禁言) |
| /set_group_whole_ban     | [群组全员禁言](https://cqhttp.cc/docs/4.15/#/API?id=set_group_whole_ban-群组全员禁言) |
| /set_group_leave         | [退出群组](https://cqhttp.cc/docs/4.15/#/API?id=set_group_leave-退出群组) |
| /set_group_name          | 设置群组名(拓展API)                                         |
| /get_image               | 获取图片信息(拓展API)                                        |
| /get_group_msg           | 获取群组消息(拓展API)                                        |
| /can_send_image          | [检查是否可以发送图片](https://cqhttp.cc/docs/4.15/#/API?id=can_send_image-检查是否可以发送图片) |
| /can_send_record         | [检查是否可以发送语音](https://cqhttp.cc/docs/4.15/#/API?id=can_send_record-检查是否可以发送语音) |
| /get_status              | [获取插件运行状态](https://cqhttp.cc/docs/4.15/#/API?id=get_status-获取插件运行状态) |
| /get_version_info        | [获取 酷Q 及 CQHTTP插件的版本信息](https://cqhttp.cc/docs/4.15/#/API?id=get_version_info-获取-酷q-及-cqhttp-插件的版本信息) |

</details>

<details>
<summary>已实现Event</summary>

##### 注意: 部分Event数据与CQHTTP原版略有差异，请参考文档
| Event                                                        |
| ------------------------------------------------------------ |
| [私聊信息](https://cqhttp.cc/docs/4.15/#/Post?id=私聊消息)   |
| [群消息](https://cqhttp.cc/docs/4.15/#/Post?id=群消息)       |
| [群消息撤回(拓展Event)](docs/cqhttp.md#群消息撤回)           |
| [好友消息撤回(拓展Event)](docs/cqhttp.md#好友消息撤回)       |
| [群内提示事件(拓展Event)(龙王等事件)](docs/cqhttp.md#群内戳一戳)           |
| [群管理员变动](https://cqhttp.cc/docs/4.15/#/Post?id=群管理员变动) |
| [群成员减少](https://cqhttp.cc/docs/4.15/#/Post?id=群成员减少) |
| [群成员增加](https://cqhttp.cc/docs/4.15/#/Post?id=群成员增加) |
| [群禁言](https://cqhttp.cc/docs/4.15/#/Post?id=群禁言)       |
| [群文件上传](https://cqhttp.cc/docs/4.15/#/Post?id=群文件上传) |
| [加好友请求](https://cqhttp.cc/docs/4.15/#/Post?id=加好友请求) |
| [加群请求/邀请](https://cqhttp.cc/docs/4.15/#/Post?id=加群请求／邀请) |

</details>

# 关于ISSUE

以下ISSUE会被直接关闭
- 提交BUG不使用Template
- 询问已知问题
- 提问找不到重点
- 重复提问

> 请注意, 开发者并没有义务回复您的问题. 您应该具备基本的提问技巧。

# 性能

在关闭数据库的情况下, 加载25个好友128个群运行24小时后内存使用为10MB左右. 开启数据库后内存使用将根据消息量增加10-20MB, 如果系统内存小于128M建议关闭数据库使用. 
