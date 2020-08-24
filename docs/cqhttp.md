# 拓展API

由于部分 api 原版 CQHTTP 并未实现，go-cqhttp 修改并增加了一些拓展 api .

## CQCode

### 回复

Type : `reply`

范围: **发送/接收**

参数:

| 参数名 | 类型 | 说明                                  |
| ------ | ---- | ------------------------------------- |
| id     | int  | 回复时所引用的消息id, 必须为本群消息. |

示例: `[CQ:reply,id=123456]`

 ### 合并转发

Type: `forward`

范围: **接收**

参数:

| 参数名 | 类型   | 说明                                                         |
| ------ | ------ | ------------------------------------------------------------ |
| id     | string | 合并转发ID, 需要通过 `/get_forward_msg` API获取转发的具体内容 |

示例: `[CQ:forward,id=xxxx]`

### 合并转发消息节点

Type: `node`

范围: **发送**

参数:

| 参数名  | 类型    | 说明           | 特殊说明                                                     |
| ------- | ------- | -------------- | ------------------------------------------------------------ |
| id      | int32   | 转发消息id     | 直接引用他人的消息合并转发,  实际查看顺序为原消息发送顺序 **与下面的自定义消息二选一** |
| name    | string  | 发送者显示名字 | 用于自定义消息 (自定义消息并合并转发，实际查看顺序为自定义消息段顺序) |
| uin     | int64   | 发送者QQ号     | 用于自定义消息                                               |
| content | message | 具体消息       | 用于自定义消息 **不支持转发套娃，不支持引用回复**            |

特殊说明: **需要使用单独的API `/send_group_forward_msg` 发送，并且由于消息段较为复杂，仅支持Array形式入参。 如果引用消息和自定义消息同时出现，实际查看顺序将取消息段顺序.  另外按 [CQHTTP](https://cqhttp.cc/docs/4.15/#/Message?id=格式) 文档说明, `data` 应全为字符串, 但由于需要接收`message` 类型的消息, 所以 *仅限此Type的content字段* 支持Array套娃**

示例: 

直接引用消息合并转发:

````json
[
    {
        "type": "node",
        "data": {
            "id": "123"
        }
    },
    {
        "type": "node",
        "data": {
            "id": "456"
        }
    }
]
````

自定义消息合并转发:

````json
[
    {
        "type": "node",
        "data": {
            "name": "消息发送者A",
            "uin": "10086",
            "content": [
                {
                    "type": "text",
                    "data": {"text": "测试消息1"}
                }
            ]
        }
    },
    {
        "type": "node",
        "data": {
            "name": "消息发送者B",
            "uin": "10087",
            "content": "[CQ:image,file=xxxxx]测试消息2"
        }
    }
]
````

引用自定义混合合并转发:

````json
[
    {
        "type": "node",
        "data": {
            "name": "自定义发送者",
            "uin": "10086",
            "content": "我是自定义消息"
        }
    },
    {
        "type": "node",
        "data": {
            "id": "123"
        }
    }
]
````

### xml支持

Type: `xml`

范围: **发送**

参数:

| 参数名 | 类型   | 说明                                                         |
| ------ | ------ | ------------------------------------------------------------ |
| data     | string | xml内容，xml中的value部分，记得实体化处理|
| resid     | int32 | 可以不填|

示例: `[CQ:xml,data=xxxx]`

####一些xml样例
####ps:重要：xml中的value部分，记得html实体化处理后，再打加入到cq码中
#### qq音乐
```xml
<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="2" templateID="1" action="web" brief="&#91;分享&#93; 十年" sourceMsgId="0" url="https://i.y.qq.com/v8/playsong.html?_wv=1&amp;songid=4830342&amp;souce=qqshare&amp;source=qqshare&amp;ADTAG=qqshare" flag="0" adverSign="0" multiMsgFlag="0" ><item layout="2"><audio cover="http://imgcache.qq.com/music/photo/album_500/26/500_albumpic_89526_0.jpg" src="http://ws.stream.qqmusic.qq.com/C400003mAan70zUy5O.m4a?guid=1535153710&amp;vkey=D5315B8C0603653592AD4879A8A3742177F59D582A7A86546E24DD7F282C3ACF81526C76E293E57EA1E42CF19881C561275D919233333ADE&amp;uin=&amp;fromtag=3" /><title>十年</title><summary>陈奕迅</summary></item><source name="QQ音乐" icon="https://i.gtimg.cn/open/app_icon/01/07/98/56/1101079856_100_m.png" url="http://web.p.qq.com/qqmpmobile/aio/app.html?id=1101079856" action="app"  a_actionData="com.tencent.qqmusic" i_actionData="tencent1101079856://" appid="1101079856" /></msg>
```
#### 网易音乐
```xml
<?xml version='1.0' encoding='UTF-8' standalone='yes' ?><msg serviceID="2" templateID="1" action="web" brief="&#91;分享&#93; 十年" sourceMsgId="0" url="http://music.163.com/m/song/409650368" flag="0" adverSign="0" multiMsgFlag="0" ><item layout="2"><audio cover="http://p2.music.126.net/g-Qgb9ibk9Wp_0HWra0xQQ==/16636710440565853.jpg?param=90y90" src="https://music.163.com/song/media/outer/url?id=409650368.mp3" /><title>十年</title><summary>黄梦之</summary></item><source name="网易云音乐" icon="https://pic.rmb.bdstatic.com/911423bee2bef937975b29b265d737b3.png" url="http://web.p.qq.com/qqmpmobile/aio/app.html?id=1101079856" action="app" a_actionData="com.netease.cloudmusic" i_actionData="tencent100495085://" appid="100495085" /></msg>
```

#### 卡片消息1
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<msg serviceID="1">
<item><title>生死8秒！女司机高速急刹，他一个操作救下一车性命</title></item>
<source name="官方认证消息" icon="https://qzs.qq.com/ac/qzone_v5/client/auth_icon.png" action="" appid="-1" />
</msg>
```

#### 卡片消息2
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<msg serviceID="1">
<item layout="4">
<title>test title</title>
<picture cover="http://url.cn/5CEwIUy"/>
</item>
</msg>
```


## API

### 设置群名

终结点: `/set_group_name`  

**参数** 

| 字段     | 类型   | 说明 |
| -------- | ------ | ---- |
| group_id | int64  | 群号 |
| name     | string | 新名 |

### 获取图片信息

终结点: `/get_image`  

> 该接口为 CQHTTP 接口修改

参数

| 字段   | 类型   | 说明           |
| ------ | ------ | -------------- |
| `file` | string | 图片缓存文件名 |

响应数据

| 字段       | 类型   | 说明           |
| ---------- | ------ | -------------- |
| `size`     | int32  | 图片源文件大小 |
| `filename` | string | 图片文件原名   |
| `url`      | string | 图片下载地址   |

### 获取群消息

终结点: `/get_group_msg` 

参数

| 字段         | 类型  | 说明   |
| ------------ | ----- | ------ |
| `message_id` | int32 | 消息id |

响应数据

| 字段         | 类型    | 说明       |
| ------------ | ------- | ---------- |
| `message_id` | int32   | 消息id     |
| `real_id`    | int32   | 消息真实id |
| `sender`     | object  | 发送者     |
| `time`       | int32   | 发送时间   |
| `content`    | message | 消息内容   |

### 获取合并转发内容

终结点: `/get_forward_msg`

参数

| 字段         | 类型   | 说明   |
| ------------ | ------ | ------ |
| `message_id` | string | 消息id |

响应数据

| 字段       | 类型              | 说明     |
| ---------- | ----------------- | -------- |
| `messages` | forward message[] | 消息列表 |

响应示例

````json
{
    "data": {
        "messages": [
            {
                "content": "合并转发1",
                "sender": {
                    "nickname": "发送者A",
                    "user_id": 10086
                },
                "time": 1595694374
            },
            {
                "content": "合并转发2[CQ:image,file=xxxx,url=xxxx]",
                "sender": {
                    "nickname": "发送者B",
                    "user_id": 10087
                },
                "time": 1595694393
            }
        ]
    },
    "retcode": 0,
    "status": "ok"
}
````

### 发送合并转发(群)

终结点: `/send_group_forward_msg`

**参数** 

| 字段       | 类型           | 说明                         |
| ---------- | -------------- | ---------------------------- |
| `group_id` | int64          | 群号                         |
| `messages` | forward node[] | 自定义转发消息, 具体看CQCode |

### 

## 事件

#### 群消息撤回

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `group_recall` | 消息类型       |
| `group_id`    | int64  |                | 群号           |
| `user_id`     | int64  |                | 消息发送者id   |
| `operator_id` | int64  |                | 操作者id       |
| `message_id`  | int64  |                | 被撤回的消息id |

#### 好友消息撤回

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `friend_recall`| 消息类型       |
| `user_id`     | int64  |                | 好友id        |
| `message_id`  | int64  |                | 被撤回的消息id |

