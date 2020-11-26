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
- [x] 使用代理请求网络图片

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
- [CQ:gift]
- [CQ:redbag]
- [CQ:tts]
- [CQ:music]

</details>

<details>
<summary>已实现API</summary>

##### 注意: 部分API实现与CQHTTP原版略有差异，请参考文档
| API                      | 功能                                                                                                                                                                                                         |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| /get_login_info          | [获取登录号信息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_login_info-%E8%8E%B7%E5%8F%96%E7%99%BB%E5%BD%95%E5%8F%B7%E4%BF%A1%E6%81%AF)                                   |
| /get_friend_list         | [获取好友列表](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_friend_list-%E8%8E%B7%E5%8F%96%E5%A5%BD%E5%8F%8B%E5%88%97%E8%A1%A8)                                             |
| /get_group_list          | [获取群列表](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_group_list-%E8%8E%B7%E5%8F%96%E7%BE%A4%E5%88%97%E8%A1%A8)                                                         |
| /get_group_info          | [获取群信息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_group_info-%E8%8E%B7%E5%8F%96%E7%BE%A4%E4%BF%A1%E6%81%AF)                                                         |
| /get_group_member_list   | [获取群成员列表](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_group_member_list-%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%88%90%E5%91%98%E5%88%97%E8%A1%A8)                            |
| /get_group_member_info   | [获取群成员信息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_group_member_info-%E8%8E%B7%E5%8F%96%E7%BE%A4%E6%88%90%E5%91%98%E4%BF%A1%E6%81%AF)                            |
| /send_msg                | [发送消息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#send_msg-%E5%8F%91%E9%80%81%E6%B6%88%E6%81%AF)                                                                          |
| /send_group_msg          | [发送群消息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#send_group_msg-%E5%8F%91%E9%80%81%E7%BE%A4%E6%B6%88%E6%81%AF)                                                         |
| /send_private_msg        | [发送私聊消息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#send_private_msg-%E5%8F%91%E9%80%81%E7%A7%81%E8%81%8A%E6%B6%88%E6%81%AF)                                            |
| /delete_msg              | [撤回信息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#delete_msg-%E6%92%A4%E5%9B%9E%E6%B6%88%E6%81%AF)                                                                        |
| /set_friend_add_request  | [处理加好友请求](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_friend_add_request-%E5%A4%84%E7%90%86%E5%8A%A0%E5%A5%BD%E5%8F%8B%E8%AF%B7%E6%B1%82)                           |
| /set_group_add_request   | [处理加群请求/邀请](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_add_request-%E5%A4%84%E7%90%86%E5%8A%A0%E7%BE%A4%E8%AF%B7%E6%B1%82%E9%82%80%E8%AF%B7)                |
| /set_group_card          | [设置群名片(群备注)](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_card-%E8%AE%BE%E7%BD%AE%E7%BE%A4%E5%90%8D%E7%89%87%E7%BE%A4%E5%A4%87%E6%B3%A8)                      |
| /set_group_special_title | [设置群组专属头衔](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_special_title-%E8%AE%BE%E7%BD%AE%E7%BE%A4%E7%BB%84%E4%B8%93%E5%B1%9E%E5%A4%B4%E8%A1%94)               |
| /set_group_kick          | [群组T人](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_kick-%E7%BE%A4%E7%BB%84%E8%B8%A2%E4%BA%BA)                                                                     |
| /set_group_ban           | [群组单人禁言](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_ban-%E7%BE%A4%E7%BB%84%E5%8D%95%E4%BA%BA%E7%A6%81%E8%A8%80)                                               |
| /set_group_whole_ban     | [群组全员禁言](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_whole_ban-%E7%BE%A4%E7%BB%84%E5%85%A8%E5%91%98%E7%A6%81%E8%A8%80)                                         |
| /set_group_leave         | [退出群组](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_leave-%E9%80%80%E5%87%BA%E7%BE%A4%E7%BB%84)                                                                   |
| /set_group_name          | [设置群组名](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_group_name-%E8%AE%BE%E7%BD%AE%E7%BE%A4%E5%90%8D)                                                                  |
| /set_restart             | [重启go-cqhttp](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#set_restart-%E9%87%8D%E5%90%AF-onebot-%E5%AE%9E%E7%8E%B0)                                                          |
| /get_image               | [获取图片信息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_image-%E8%8E%B7%E5%8F%96%E5%9B%BE%E7%89%87)                                                                     |
| /get_msg                 | [获取消息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_msg-%E8%8E%B7%E5%8F%96%E6%B6%88%E6%81%AF)                                                                           |
| /can_send_image          | [检查是否可以发送图片](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#can_send_image-%E6%A3%80%E6%9F%A5%E6%98%AF%E5%90%A6%E5%8F%AF%E4%BB%A5%E5%8F%91%E9%80%81%E5%9B%BE%E7%89%87)  |
| /can_send_record         | [检查是否可以发送语音](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#can_send_record-%E6%A3%80%E6%9F%A5%E6%98%AF%E5%90%A6%E5%8F%AF%E4%BB%A5%E5%8F%91%E9%80%81%E8%AF%AD%E9%9F%B3) |
| /get_status              | [获取插件运行状态](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_status-%E8%8E%B7%E5%8F%96%E8%BF%90%E8%A1%8C%E7%8A%B6%E6%80%81)                                              |
| /get_version_info        | [获取 酷Q 及 CQHTTP插件的版本信息](https://github.com/howmanybots/onebot/blob/master/v11/specs/api/public.md#get_version_info-%E8%8E%B7%E5%8F%96%E7%89%88%E6%9C%AC%E4%BF%A1%E6%81%AF)                        |

</details>

<details>
<summary>已实现Event</summary>

##### 注意: 部分Event数据与CQHTTP原版略有差异，请参考文档
| Event                                                                                                                                                |
| ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| [私聊信息](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/message.md#%E7%A7%81%E8%81%8A%E6%B6%88%E6%81%AF)                        |
| [群消息](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/message.md#%E7%BE%A4%E6%B6%88%E6%81%AF)                                   |
| [群消息撤回(拓展Event)](docs/cqhttp.md#群消息撤回)                                                                                                   |
| [好友消息撤回(拓展Event)](docs/cqhttp.md#好友消息撤回)                                                                                               |
| [群内提示事件(拓展Event)(龙王等事件)](docs/cqhttp.md#群内戳一戳)                                                                                     |
| [群管理员变动](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/notice.md#%E7%BE%A4%E7%AE%A1%E7%90%86%E5%91%98%E5%8F%98%E5%8A%A8)   |
| [群成员减少](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/notice.md#%E7%BE%A4%E6%88%90%E5%91%98%E5%87%8F%E5%B0%91)              |
| [群成员增加](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/notice.md#%E7%BE%A4%E6%88%90%E5%91%98%E5%A2%9E%E5%8A%A0)              |
| [群禁言](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/notice.md#%E7%BE%A4%E7%A6%81%E8%A8%80)                                    |
| [群文件上传](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/notice.md#%E7%BE%A4%E6%96%87%E4%BB%B6%E4%B8%8A%E4%BC%A0)              |
| [加好友请求](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/request.md#%E5%8A%A0%E5%A5%BD%E5%8F%8B%E8%AF%B7%E6%B1%82)             |
| [加群请求/邀请](https://github.com/howmanybots/onebot/blob/master/v11/specs/event/request.md#%E5%8A%A0%E7%BE%A4%E8%AF%B7%E6%B1%82%E9%82%80%E8%AF%B7) |

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
