# 拓展API

由于部分 api 原版 CQHTTP 并未实现，go-cqhttp 修改并增加了一些拓展 api .

<details>
<summary>目录</summary>
<p>

##### CQCode
- [图片](#图片)
- [回复](#回复)
- [红包](#红包)
- [戳一戳](#戳一戳)
- [礼物](#礼物)
- [合并转发](#合并转发)
- [合并转发消息节点](#合并转发消息节点)
- [XML 消息](#xml-消息)
- [JSON 消息](#json-消息)
- [cardimage](#cardimage)
- [文本转语音](#文本转语音)
- [图片](#图片)

##### API
- [设置群名](#设置群名)
- [设置群头像](#设置群头像)
- [获取图片信息](#获取图片信息)
- [获取消息](#获取消息)
- [获取合并转发内容](#获取合并转发内容)
- [发送合并转发(群)](#发送合并转发群)
- [获取中文分词](#获取中文分词)
- [图片OCR](#图片ocr)
- [获取群系统消息](#获取群文件系统信息)
- [获取群文件系统信息](#获取群文件系统信息)
- [获取群根目录文件列表](#获取群根目录文件列表)
- [获取群子目录文件列表](#获取群子目录文件列表)
- [获取群文件资源链接](#获取群文件资源链接)
- [获取状态](#获取状态)
- [获取群@全体成员剩余次数](#获取群全体成员剩余次数)
- [下载文件到缓存目录](#下载文件到缓存目录)
- [获取群消息历史记录](#获取群消息历史记录)
- [设置群名](#设置群名)
- [获取用户VIP信息](#获取用户vip信息)
- [发送群公告](#发送群公告)
- [设置精华消息](#设置精华消息)
- [移出精华消息](#移出精华消息)
- [获取精华消息列表](#获取精华消息列表)
- [重载事件过滤器](#重载事件过滤器)

##### 事件
- [群消息撤回](#群消息撤回)
- [好友消息撤回](#好友消息撤回)
- [好友戳一戳](#好友戳一戳)
- [群内戳一戳](#群内戳一戳)
- [群红包运气王提示](#群红包运气王提示)
- [群成员荣誉变更提示](#群成员荣誉变更提示)
- [群成员名片更新](#群成员名片更新)
- [接收到离线文件](#接收到离线文件)
- [群精华消息](#精华消息)

</p>
</details>

## CQCode

### 图片

Type : `image`

范围: **发送/接收**

参数:

| 参数名  | 可能的值        | 说明                                                                   |
| ------- | --------------- | ---------------------------------------------------------------------- |
| `file`  | -               | 图片文件名                                                             |
| `type`  | `flash`，`show` | 图片类型，`flash` 表示闪照，`show` 表示秀图，默认普通图片              |
| `url`   | -               | 图片 URL                                                               |
| `cache` | `0` `1`         | 只在通过网络 URL 发送时有效，表示是否使用已缓存的文件，默认 `1`        |
| `id`    | -               | 发送秀图时的特效id，默认为40000                                        |
| `c`     | `2` `3`         | 通过网络下载图片时的线程数, 默认单线程. (在资源不支持并发时会自动处理) |

可用的特效ID:

| id    | 类型 |
| ----- | ---- |
| 40000 | 普通 |
| 40001 | 幻影 |
| 40002 | 抖动 |
| 40003 | 生日 |
| 40004 | 爱你 |
| 40005 | 征友 |

示例: `[CQ:image,file=http://baidu.com/1.jpg,type=show,id=40004]`

> 注意：图片总大小不能超过30MB，gif总帧数不能超过300帧

### 回复

Type : `reply`

范围: **发送/接收**

> 注意: 如果id存在则优先处理id

参数:

| 参数名 | 类型   | 说明                                                |
| ------ | ------ | --------------------------------------------------- |
| `id`   | int    | 回复时所引用的消息id, 必须为本群消息.               |
| `text` | string | 自定义回复的信息                                    |
| `qq`   | int64  | 自定义回复时的自定义QQ, 如果使用自定义信息必须指定. |
| `time` | int64  | 可选. 自定义回复时的时间, 格式为Unix时间 |
| `seq` | int64  | 起始消息序号, 可通过 `get_msg` 获得 |



示例: `[CQ:reply,id=123456]` 
\
自定义回复示例: `[CQ:reply,text=Hello World,qq=10086,time=3376656000,seq=5123]`

### 音乐分享 <Badge text="发"/>

```json
{
    "type": "music",
    "data": {
        "type": "163",
        "id": "28949129"
    }
}
```

```
[CQ:music,type=163,id=28949129]
```

| 参数名 | 收  | 发  | 可能的值   | 说明                             |
| ------ | --- | --- | ---------- | -------------------------------- |
| `type` |     | ✓   | `qq` `163` | 分别表示使用 QQ 音乐、网易云音乐 |
| `id`   |     | ✓   | -          | 歌曲 ID                          |

### 音乐自定义分享 <Badge text="发"/>

```json
{
    "type": "music",
    "data": {
        "type": "custom",
        "url": "http://baidu.com",
        "audio": "http://baidu.com/1.mp3",
        "title": "音乐标题"
    }
}
```

```
[CQ:music,type=custom,url=http://baidu.com,audio=http://baidu.com/1.mp3,title=音乐标题]
```

| 参数名    | 收  | 发  | 可能的值                 | 说明                                                  |
| --------- | --- | --- | ------------------------ | ----------------------------------------------------- |
| `type`    |     | ✓   | `custom`                 | 表示音乐自定义分享                                    |
| `subtype` |     | ✓   | `qq,163,migu,kugou,kuwo` | 表示分享类型，不填写发送为xml卡片，推荐填写提高稳定性 |
| `url`     |     | ✓   | -                        | 点击后跳转目标 URL                                    |
| `audio`   |     | ✓   | -                        | 音乐 URL                                              |
| `title`   |     | ✓   | -                        | 标题                                                  |
| `content` |     | ✓   | -                        | 内容描述                                              |
| `image`   |     | ✓   | -                        | 图片 URL                                              |

### 红包

Type: `redbag`

范围: **接收**

参数:

| 参数名  | 类型   | 说明        |
| ------- | ------ | ----------- |
| `title` | string | 祝福语/口令 |

示例: `[CQ:redbag,title=恭喜发财]`

### 戳一戳

> 注意：发送戳一戳消息无法撤回，返回的 `message id`  恒定为 `0`

Type: `poke`

范围: **发送(仅群聊)**

参数:

| 参数名 | 类型  | 说明         |
| ------ | ----- | ------------ |
| `qq`   | int64 | 需要戳的成员 |

示例: `[CQ:poke,qq=123456]`

### 礼物

> 注意：仅支持免费礼物,发送群礼物消息无法撤回,返回的 `message id`  恒定为 `0`

Type: `gift`

范围: **发送(仅群聊,接收的时候不是CQ码)**

参数:

| 参数名 | 类型  | 说明           |
| ------ | ----- | -------------- |
| `qq`   | int64 | 接收礼物的成员 |
| `id`   | int   | 礼物的类型     |

目前支持的礼物ID:

| id  | 类型       |
| --- | ---------- |
| 0   | 甜Wink     |
| 1   | 快乐肥宅水 |
| 2   | 幸运手链   |
| 3   | 卡布奇诺   |
| 4   | 猫咪手表   |
| 5   | 绒绒手套   |
| 6   | 彩虹糖果   |
| 7   | 坚强       |
| 8   | 告白话筒   |
| 9   | 牵你的手   |
| 10  | 可爱猫咪   |
| 11  | 神秘面具   |
| 12  | 我超忙的   |
| 13  | 爱心口罩   |



示例: `[CQ:gift,qq=123456,id=8]`

 ### 合并转发

Type: `forward`

范围: **接收**

参数:

| 参数名 | 类型   | 说明                                                          |
| ------ | ------ | ------------------------------------------------------------- |
| `id`   | string | 合并转发ID, 需要通过 `/get_forward_msg` API获取转发的具体内容 |

示例: `[CQ:forward,id=xxxx]`

### 合并转发消息节点

Type: `node`

范围: **发送**

参数:

| 参数名    | 类型    | 说明           | 特殊说明                                                                               |
| --------- | ------- | -------------- | -------------------------------------------------------------------------------------- |
| `id`      | int32   | 转发消息id     | 直接引用他人的消息合并转发,  实际查看顺序为原消息发送顺序 **与下面的自定义消息二选一** |
| `name`    | string  | 发送者显示名字 | 用于自定义消息 (自定义消息并合并转发，实际查看顺序为自定义消息段顺序)                  |
| `uin`     | int64   | 发送者QQ号     | 用于自定义消息                                                                         |
| `content` | message | 具体消息       | 用于自定义消息                                                                         |
| `seq`     | message | 具体消息       | 用于自定义消息                                                                         |

特殊说明: **需要使用单独的API `/send_group_forward_msg` 发送，并且由于消息段较为复杂，仅支持Array形式入参。 如果引用消息和自定义消息同时出现，实际查看顺序将取消息段顺序.  另外按 [CQHTTP](https://git.io/JtxtN) 文档说明, `data` 应全为字符串, 但由于需要接收`message` 类型的消息, 所以 *仅限此Type的content字段* 支持Array套娃**

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
            "content": "我是自定义消息",
            "seq": "5123",
            "time": "3376656000"
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
### 短视频消息

Type: `video`

范围: **发送/接收**

参数:

| 参数名  | 类型    | 说明                                                                   |
| ------- | ------- | ---------------------------------------------------------------------- |
| `file`  | string  | 支持http和file发送                                                     |
| `cover` | string  | 视频封面，支持http，file和base64发送，格式必须为jpg                    |
| `c`     | `2` `3` | 通过网络下载视频时的线程数, 默认单线程. (在资源不支持并发时会自动处理) |
示例: `[CQ:image,file=file:///C:\\Users\Richard\Pictures\1.mp4]`

### XML 消息

Type: `xml`

范围: **发送/接收**

参数:

| 参数名  | 类型   | 说明                                      |
| ------- | ------ | ----------------------------------------- |
| `data`  | string | xml内容，xml中的value部分，记得实体化处理 |
| `resid` | int32  | 可以不填                                  |

示例: `[CQ:xml,data=xxxx]`

#### 一些xml样例

#### ps:重要：xml中的value部分，记得html实体化处理后，再打加入到cq码中

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

### JSON 消息

Type: `json`

范围: **发送/接收**

参数:

| 参数名  | 类型   | 说明                                            |
| ------- | ------ | ----------------------------------------------- |
| `data`  | string | json内容，json的所有字符串记得实体化处理        |
| `resid` | int32  | 默认不填为0，走小程序通道，填了走富文本通道发送 |

json中的字符串需要进行转义：

>","=> `&#44;`

>"&"=> `&amp;`

>"["=> `&#91;`

>"]"=> `&#93;`

否则无法正确得到解析

示例json 的cq码：
```test
[CQ:json,data={"app":"com.tencent.miniapp"&#44;"desc":""&#44;"view":"notification"&#44;"ver":"0.0.0.1"&#44;"prompt":"&#91;应用&#93;"&#44;"appID":""&#44;"sourceName":""&#44;"actionData":""&#44;"actionData_A":""&#44;"sourceUrl":""&#44;"meta":{"notification":{"appInfo":{"appName":"全国疫情数据统计"&#44;"appType":4&#44;"appid":1109659848&#44;"iconUrl":"http:\/\/gchat.qpic.cn\/gchatpic_new\/719328335\/-2010394141-6383A777BEB79B70B31CE250142D740F\/0"}&#44;"data":&#91;{"title":"确诊"&#44;"value":"80932"}&#44;{"title":"今日确诊"&#44;"value":"28"}&#44;{"title":"疑似"&#44;"value":"72"}&#44;{"title":"今日疑似"&#44;"value":"5"}&#44;{"title":"治愈"&#44;"value":"60197"}&#44;{"title":"今日治愈"&#44;"value":"1513"}&#44;{"title":"死亡"&#44;"value":"3140"}&#44;{"title":"今**亡"&#44;"value":"17"}&#93;&#44;"title":"中国加油，武汉加油"&#44;"button":&#91;{"name":"病毒：SARS-CoV-2，其导致疾病命名 COVID-19"&#44;"action":""}&#44;{"name":"传染源：新冠肺炎的患者。无症状感染者也可能成为传染源。"&#44;"action":""}&#93;&#44;"emphasis_keyword":""}}&#44;"text":""&#44;"sourceAd":""}]
```


### cardimage
一种xml的图片消息（装逼大图）

ps: xml 接口的消息都存在风控风险，请自行兼容发送失败后的处理（可以失败后走普通图片模式）

Type: `cardimage`

范围: **发送**

参数:

| 参数名      | 类型   | 说明                                  |
| ----------- | ------ | ------------------------------------- |
| `file`      | string | 和image的file字段对齐，支持也是一样的 |
| `minwidth`  | int64  | 默认不填为400，最小width              |
| `minheight` | int64  | 默认不填为400，最小height             |
| `maxwidth`  | int64  | 默认不填为500，最大width              |
| `maxheight` | int64  | 默认不填为1000，最大height            |
| `source`    | string | 分享来源的名称，可以留空              |
| `icon`      | string | 分享来源的icon图标url，可以留空       |


示例cardimage 的cq码：
```test
[CQ:cardimage,file=https://i.pixiv.cat/img-master/img/2020/03/25/00/00/08/80334602_p0_master1200.jpg]
```

### 文本转语音

> 注意：通过TX的TTS接口，采用的音源与登录账号的性别有关

Type: `tts`

范围: **发送(仅群聊)**

参数:

| 参数名 | 类型   | 说明 |
| ------ | ------ | ---- |
| `text` | string | 内容 |

示例: `[CQ:tts,text=这是一条测试消息]`

## API

### 设置群名

终结点: `/set_group_name`  

**参数** 

| 字段         | 类型   | 说明 |
| ------------ | ------ | ---- |
| `group_id`   | int64  | 群号 |
| `group_name` | string | 新名 |

### 设置群头像

终结点: `/set_group_portrait`  

**参数** 

| 字段       | 类型   | 说明                     |
| ---------- | ------ | ------------------------ |
| `group_id` | int64  | 群号                     |
| `file`     | string | 图片文件名               |
| `cache`    | int    | 表示是否使用已缓存的文件 |

[1]`file` 参数支持以下几种格式：

- 绝对路径，例如 `file:///C:\\Users\Richard\Pictures\1.png`，格式使用 [`file` URI](https://tools.ietf.org/html/rfc8089)
- 网络 URL，例如 `http://i1.piimg.com/567571/fdd6e7b6d93f1ef0.jpg`
- Base64 编码，例如 `base64://iVBORw0KGgoAAAANSUhEUgAAABQAAAAVCAIAAADJt1n/AAAAKElEQVQ4EWPk5+RmIBcwkasRpG9UM4mhNxpgowFGMARGEwnBIEJVAAAdBgBNAZf+QAAAAABJRU5ErkJggg==`

[2]`cache`参数: 通过网络 URL 发送时有效，`1`表示使用缓存，`0`关闭关闭缓存，默认 为`1`

[3] 目前这个API在登录一段时间后因cookie失效而失效，请考虑后使用

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

### 获取消息

终结点: `/get_msg` 

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
| `message`    | message | 消息内容   |

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

````json5
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
                "time": 1595694393 //  可选
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
| `messages` | forward node[] | 自定义转发消息, 具体看 [CQCode](https://github.com/Mrs4s/go-cqhttp/blob/master/docs/cqhttp.md#%E5%90%88%E5%B9%B6%E8%BD%AC%E5%8F%91%E6%B6%88%E6%81%AF%E8%8A%82%E7%82%B9) |

响应数据
    
| 字段         | 类型   | 说明   |
| ------------ | ------ | ------ |
| `message_id` | string | 消息id |

### 获取中文分词

终结点: `/.get_word_slices`  

**参数** 

| 字段      | 类型   | 说明 |
| --------- | ------ | ---- |
| `content` | string | 内容 |

**响应数据**

| 字段     | 类型     | 说明 |
| -------- | -------- | ---- |
| `slices` | string[] | 词组 |

### 设置精华消息

终结点: `/set_essence_msg`

**参数**

| 字段         | 类型  | 说明   |
| ------------ | ----- | ------ |
| `message_id` | int32 | 消息ID |

**响应数据**

无

### 移出精华消息

终结点: `/delete_essence_msg`

**参数**

| 字段         | 类型  | 说明   |
| ------------ | ----- | ------ |
| `message_id` | int32 | 消息ID |

**响应数据**

无

### 获取精华消息列表

终结点: `/get_essence_msg_list`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `group_id` | int64 | 群号 |

**响应数据**

响应内容为 JSON 数组，每个元素如下：

| 字段名          | 数据类型 | 说明         |
| --------------- | -------- | ------------ |
| `sender_id`     | int64    | 发送者QQ 号  |
| `sender_nick`   | string   | 发送者昵称   |
| `sender_time`   | int64    | 消息发送时间 |
| `operator_id`   | int64    | 操作者QQ 号  |
| `operator_nick` | string   | 操作者昵称   |
| `operator_time` | int64    | 精华设置时间 |
| `message_id`    | int32    | 消息ID       |

### 图片OCR

> 注意: 目前图片OCR接口仅支持接受的图片

终结点: `/ocr_image` 

**参数** 

| 字段    | 类型   | 说明   |
| ------- | ------ | ------ |
| `image` | string | 图片ID |

**响应数据**

| 字段       | 类型            | 说明    |
| ---------- | --------------- | ------- |
| `texts`    | TextDetection[] | OCR结果 |
| `language` | string          | 语言    |

**TextDetection**

| 字段          | 类型    | 说明   |
| ------------- | ------- | ------ |
| `text`        | string  | 文本   |
| `confidence`  | int32   | 置信度 |
| `coordinates` | vector2 | 坐标   |


### 获取群系统消息

终结点: `/get_group_system_msg`

**响应数据**

| 字段               | 类型             | 说明         |
| ------------------ | ---------------- | ------------ |
| `invited_requests` | InvitedRequest[] | 邀请消息列表 |
| `join_requests`    | JoinRequest[]    | 进群消息列表 |

 > 注意: 如果列表不存在任何消息, 将返回 `null`

 **InvitedRequest**

| 字段           | 类型   | 说明              |
| -------------- | ------ | ----------------- |
| `request_id`   | int64  | 请求ID            |
| `invitor_uin`  | int64  | 邀请者            |
| `invitor_nick` | string | 邀请者昵称        |
| `group_id`     | int64  | 群号              |
| `group_name`   | string | 群名              |
| `checked`      | bool   | 是否已被处理      |
| `actor`        | int64  | 处理者, 未处理为0 |

  **JoinRequest**

| 字段             | 类型   | 说明              |
| ---------------- | ------ | ----------------- |
| `request_id`     | int64  | 请求ID            |
| `requester_uin`  | int64  | 请求者ID          |
| `requester_nick` | string | 请求者昵称        |
| `message`        | string | 验证消息          |
| `group_id`       | int64  | 群号              |
| `group_name`     | string | 群名              |
| `checked`        | bool   | 是否已被处理      |
| `actor`          | int64  | 处理者, 未处理为0 |

### 获取群文件系统信息

终结点: `/get_group_file_system_info`

**参数** 

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `group_id` | int64 | 群号 |

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `file_count`  | int32 | 文件总数   |
| `limit_count` | int32 | 文件上限   |
| `used_space`  | int64 | 已使用空间 |
| `total_space` | int64 | 空间上限   |

### 获取群根目录文件列表

> `File` 和 `Folder` 对象信息请参考最下方

终结点: `/get_group_root_files`

**参数** 

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `group_id` | int64 | 群号 |

**响应数据**

| 字段      | 类型     | 说明       |
| --------- | -------- | ---------- |
| `files`   | File[]   | 文件列表   |
| `folders` | Folder[] | 文件夹列表 |

### 获取群子目录文件列表

> `File` 和 `Folder` 对象信息请参考最下方

终结点: `/get_group_files_by_folder`

**参数** 

| 字段        | 类型   | 说明                        |
| ----------- | ------ | --------------------------- |
| `group_id`  | int64  | 群号                        |
| `folder_id` | string | 文件夹ID 参考 `Folder` 对象 |

**响应数据**

| 字段      | 类型     | 说明       |
| --------- | -------- | ---------- |
| `files`   | File[]   | 文件列表   |
| `folders` | Folder[] | 文件夹列表 |

### 获取群文件资源链接

> `File` 和 `Folder` 对象信息请参考最下方

终结点: `/get_group_file_url`

**参数** 

| 字段       | 类型   | 说明                      |
| ---------- | ------ | ------------------------- |
| `group_id` | int64  | 群号                      |
| `file_id`  | string | 文件ID 参考 `File` 对象   |
| `busid`    | int32  | 文件类型 参考 `File` 对象 |

**响应数据**

| 字段  | 类型   | 说明         |
| ----- | ------ | ------------ |
| `url` | string | 文件下载链接 |

 **File**

| 字段             | 类型   | 说明                   |
| ---------------- | ------ | ---------------------- |
| `file_id`        | string | 文件ID                 |
| `file_name`      | string | 文件名                 |
| `busid`          | int32  | 文件类型               |
| `file_size`      | int64  | 文件大小               |
| `upload_time`    | int64  | 上传时间               |
| `dead_time`      | int64  | 过期时间,永久文件恒为0 |
| `modify_time`    | int64  | 最后修改时间           |
| `download_times` | int32  | 下载次数               |
| `uploader`       | int64  | 上传者ID               |
| `uploader_name`  | string | 上传者名字             |

 **Folder**

| 字段               | 类型   | 说明       |
| ------------------ | ------ | ---------- |
| `folder_id`        | string | 文件夹ID   |
| `folder_name`      | string | 文件名     |
| `create_time`      | int64  | 创建时间   |
| `creator`          | int64  | 创建者     |
| `creator_name`     | string | 创建者名字 |
| `total_file_count` | int32  | 子文件数量 |

### 上传群文件

终结点: `/upload_group_file`

**参数** 

| 字段       | 类型   | 说明                      |
| ---------- | ------ | ------------------------- |
| `group_id` | int64  | 群号                      |
| `file`     | string |  本地文件路径       |
| `name`     | string | 储存名称         |
| `folder`   | string | 父目录ID           |

> 在不提供 `folder` 参数的情况下默认上传到根目录
> 只能上传本地文件, 需要上传 `http` 文件的话请先调用 `download_file` API下载

### 获取状态

终结点: `/get_status`

**响应数据**

| 字段              | 类型       | 说明                            |
| ----------------- | ---------- | ------------------------------- |
| `app_initialized` | bool       | 原 `CQHTTP` 字段, 恒定为 `true` |
| `app_enabled`     | bool       | 原 `CQHTTP` 字段, 恒定为 `true` |
| `plugins_good`    | bool       | 原 `CQHTTP` 字段, 恒定为 `true` |
| `app_good`        | bool       | 原 `CQHTTP` 字段, 恒定为 `true` |
| `online`          | bool       | 表示BOT是否在线                 |
| `good`            | bool       | 同 `online`                     |
| `stat`            | Statistics | 运行统计                        |

**Statistics**


| 字段               | 类型   | 说明             |
| ------------------ | ------ | ---------------- |
| `packet_received`  | uint64 | 收到的数据包总数 |
| `packet_sent`      | uint64 | 发送的数据包总数 |
| `packet_lost`      | uint32 | 数据包丢失总数   |
| `message_received` | uint64 | 接受信息总数     |
| `message_sent`     | uint64 | 发送信息总数     |
| `disconnect_times` | uint32 | TCP链接断开次数  |
| `lost_times`       | uint32 | 账号掉线次数     |

> 注意: 所有统计信息都将在重启后重制

### 获取群@全体成员剩余次数

终结点: `/get_group_at_all_remain`

**参数** 

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `group_id` | int64 | 群号 |

**响应数据**

| 字段                            | 类型  | 说明                              |
| ------------------------------- | ----- | --------------------------------- |
| `can_at_all`                    | bool  | 是否可以@全体成员                 |
| `remain_at_all_count_for_group` | int16 | 群内所有管理当天剩余@全体成员次数 |
| `remain_at_all_count_for_uin`   | int16 | BOT当天剩余@全体成员次数          |

### 下载文件到缓存目录

终结点: `/download_file`

**参数** 

| 字段           | 类型            | 说明         |
| -------------- | --------------- | ------------ |
| `url`          | string          | 链接地址     |
| `thread_count` | int32           | 下载线程数   |
| `headers`      | string or array | 自定义请求头 |

**`headers`格式:**

字符串:

```
User-Agent=YOUR_UA[\r\n]Referer=https://www.baidu.com
```

> `[\r\n]` 为换行符, 使用http请求时请注意编码

JSON数组:

```
[
    "User-Agent=YOUR_UA",
    "Referer=https://www.baidu.com",
]
```

**响应数据**

| 字段   | 类型   | 说明                 |
| ------ | ------ | -------------------- |
| `file` | string | 下载文件的*绝对路径* |

> 通过这个API下载的文件能直接放入CQ码作为图片或语音发送
> 调用后会阻塞直到下载完成后才会返回数据，请注意下载大文件时的超时

### 获取群消息历史记录

终结点：`/get_group_msg_history`

**参数** 

| 字段          | 类型  | 说明                                |
| ------------- | ----- | ----------------------------------- |
| `message_seq` | int64 | 起始消息序号, 可通过 `get_msg` 获得 |
| `group_id`    | int64 | 群号                                |

**响应数据**

| 字段       | 类型      | 说明                       |
| ---------- | --------- | -------------------------- |
| `messages` | []Message | 从起始序号开始的前19条消息 |

> 不提供起始序号将默认获取最新的消息

### 获取当前账号在线客户端列表

终结点：`/get_online_clients`

**参数** 

| 字段       | 类型 | 说明         |
| ---------- | ---- | ------------ |
| `no_cache` | bool | 是否无视缓存 |

**响应数据**

| 字段      | 类型     | 说明           |
| --------- | -------- | -------------- |
| `clients` | []Device | 在线客户端列表 |

**Device**

| 字段          | 类型   | 说明     |
| ------------- | ------ | -------- |
| `app_id`      | int64  | 客户端ID |
| `device_name` | string | 设备名称 |
| `device_kind` | string | 设备类型 |

### 检查链接安全性

终结点：`/check_url_safely`

**参数** 

| 字段       | 类型   | 说明                      |
| ---------- | ------ | ------------------------- |
| `url` | string  | 需要检查的链接  |

**响应数据**

| 字段        | 类型       | 说明            |
| ---------- | ---------- | ------------ |
| `level`    | int       |  安全等级, 1: 安全 2: 未知 3: 危险  |

### 获取用户VIP信息

终结点：`/_get_vip_info`

**参数**

| 字段名    | 数据类型 | 默认值 | 说明  |
| --------- | -------- | ------ | ----- |
| `user_id` | int64    |        | QQ 号 |

**响应数据**

| 字段               | 类型    | 说明         |
| ------------------ | ------- | ------------ |
| `user_id`          | int64   | QQ 号        |
| `nickname`         | string  | 用户昵称     |
| `level`            | int64   | QQ 等级      |
| `level_speed`      | float64 | 等级加速度   |
| `vip_level`        | string  | 会员等级     |
| `vip_growth_speed` | int64   | 会员成长速度 |
| `vip_growth_total` | int64   | 会员成长总值 |

### 发送群公告

终结点： `/_send_group_notice`

**参数**

| 字段名     | 数据类型 | 默认值 | 说明     |
| ---------- | -------- | ------ | -------- |
| `group_id` | int64    |        | 群号     |
| `content`  | string   |        | 公告内容 |

`该 API 没有响应数据`

### 删除好友

终结点: `/delete_friend`

**参数**

| 字段名     | 数据类型 | 默认值 | 说明     |
| ---------- | -------- | ------ | -------- |
| `id`       | int64    |        | 好友ID   |

`该 API 没有响应数据`

### 获取企点账号信息

> 该API只有企点协议可用

终结点: `/qidian_get_account_info`

**响应数据**

| 字段               | 类型    | 说明         |
| ------------------ | ------- | ------------ |
| `master_id`        | int64   | 父账号ID      |
| `ext_name`         | string  | 用户昵称     |
| `create_time`      | int64   | 账号创建时间  |

### 重载事件过滤器

终结点：`/reload_event_filter`

`该 API 无需参数也没有响应数据`


## 事件

### 群消息撤回

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `group_recall` | 消息类型       |
| `group_id`    | int64  |                | 群号           |
| `user_id`     | int64  |                | 消息发送者id   |
| `operator_id` | int64  |                | 操作者id       |
| `message_id`  | int64  |                | 被撤回的消息id |

### 好友消息撤回

**上报数据**

| 字段          | 类型   | 可能的值        | 说明           |
| ------------- | ------ | --------------- | -------------- |
| `post_type`   | string | `notice`        | 上报类型       |
| `notice_type` | string | `friend_recall` | 消息类型       |
| `user_id`     | int64  |                 | 好友id         |
| `message_id`  | int64  |                 | 被撤回的消息id |

## 好友戳一戳

**事件数据**

| 字段名        | 数据类型 | 可能的值 | 说明         |
| ------------- | -------- | -------- | ------------ |
| `post_type`   | string   | `notice` | 上报类型     |
| `notice_type` | string   | `notify` | 消息类型     |
| `sub_type`    | string   | `poke`   | 提示类型     |
| `self_id`     | int64    |          | BOT QQ 号    |
| `sender_id`   | int64    |          | 发送者 QQ 号 |
| `user_id`     | int64    |          | 发送者 QQ 号 |
| `target_id`   | int64    |          | 被戳者 QQ 号 |
| `time`        | int64    |          | 时间         |

### 群内戳一戳

> 注意：此事件无法在平板和手表协议上触发

**上报数据**

| 字段          | 类型   | 可能的值 | 说明     |
| ------------- | ------ | -------- | -------- |
| `post_type`   | string | `notice` | 上报类型 |
| `notice_type` | string | `notify` | 消息类型 |
| `group_id`    | int64  |          | 群号     |
| `sub_type`    | string | `poke`   | 提示类型 |
| `user_id`     | int64  |          | 发送者id |
| `target_id`   | int64  |          | 被戳者id |

### 群红包运气王提示

> 注意：此事件无法在平板和手表协议上触发

**上报数据**

| 字段          | 类型   | 可能的值     | 说明         |
| ------------- | ------ | ------------ | ------------ |
| `post_type`   | string | `notice`     | 上报类型     |
| `notice_type` | string | `notify`     | 消息类型     |
| `group_id`    | int64  |              | 群号         |
| `sub_type`    | string | `lucky_king` | 提示类型     |
| `user_id`     | int64  |              | 红包发送者id |
| `target_id`   | int64  |              | 运气王id     |

### 群成员荣誉变更提示

> 注意：此事件无法在平板和手表协议上触发

**上报数据**

| 字段          | 类型   | 可能的值                                                 | 说明     |
| ------------- | ------ | -------------------------------------------------------- | -------- |
| `post_type`   | string | `notice`                                                 | 上报类型 |
| `notice_type` | string | `notify`                                                 | 消息类型 |
| `group_id`    | int64  |                                                          | 群号     |
| `sub_type`    | string | `honor`                                                  | 提示类型 |
| `user_id`     | int64  |                                                          | 成员id   |
| `honor_type`  | string | `talkative:龙王` `performer:群聊之火` `emotion:快乐源泉` | 荣誉类型 |

### 群成员名片更新

> 注意: 此事件不保证时效性，仅在收到消息时校验卡片

**上报数据**

| 字段          | 类型   | 可能的值     | 说明     |
| ------------- | ------ | ------------ | -------- |
| `post_type`   | string | `notice`     | 上报类型 |
| `notice_type` | string | `group_card` | 消息类型 |
| `group_id`    | int64  |              | 群号     |
| `user_id`     | int64  |              | 成员id   |
| `card_new`    | int64  |              | 新名片   |
| `card_old`    | int64  |              | 旧名片   |

> PS: 当名片为空时 `card_xx` 字段为空字符串, 并不是昵称

### 接收到离线文件

**上报数据**

| 字段          | 类型   | 可能的值       | 说明     |
| ------------- | ------ | -------------- | -------- |
| `post_type`   | string | `notice`       | 上报类型 |
| `notice_type` | string | `offline_file` | 消息类型 |
| `user_id`     | int64  |                | 发送者id |
| `file`        | object |                | 文件数据 |

**file object**

| 字段   | 类型   | 可能的值 | 说明     |
| ------ | ------ | -------- | -------- |
| `name` | string |          | 文件名   |
| `size` | int64  |          | 文件大小 |
| `url`  | string |          | 下载链接 |

### 其他客户端在线状态变更

**上报数据**

| 字段          | 类型   | 可能的值        | 说明         |
| ------------- | ------ | --------------- | ------------ |
| `post_type`   | string | `notice`        | 上报类型     |
| `notice_type` | string | `client_status` | 消息类型     |
| `client`      | Device |                 | 客户端信息   |
| `online`      | bool   |                 | 当前是否在线 |

### 精华消息

**上报数据**

| 字段          | 类型   | 可能的值       | 说明                       |
| ------------- | ------ | -------------- | -------------------------- |
| `post_type`   | string | `notice`       | 上报类型                   |
| `notice_type` | string | `essence`      | 消息类型                   |
| `sub_type`    | string | `add`,`delete` | 添加为`add`,移出为`delete` |
| `sender_id`   | int64  |                | 消息发送者ID               |
| `operator_id` | int64  |                | 操作者ID                   |
| `message_id`  | int32  |                | 消息ID                     |
