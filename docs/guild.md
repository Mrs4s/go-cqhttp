# 频道相关API

> 注意: QQ频道功能目前还在测试阶段, go-cqhttp 也在适配的初期阶段, 以下 `API` `Event` 的字段名可能存在错误并均有可能在后续版本修改/添加/删除.
> 目前仅供开发者测试以及适配使用

QQ频道相关功能的事件以及API

> 注意, 最新文档已经移动到 [go-cqhttp-docs](https://github.com/ishkong/go-cqhttp-docs), 当前文档只做兼容性保留, 所以内容可能有不足.

## 命名说明

API以及字段相关命名均为参考QQ官方命名或相似产品命名规则, 由于QQ频道的账号系统独立于QQ本体, 所以各个 `ID` 并不能和QQ通用.也无法通过 `tiny_id` 获取到 `QQ号`

下表为常见字段命名说明

| 命名         | 说明                 |
| ------------ | -------------------- |
| `tiny_id`    | 在频道系统中代表用户ID, 与QQ号并不通用 |
| `guild_id`   | 频道ID               |
| `channel_id` | 子频道ID             |

> 所有频道相关事件的 `user_id` 均为 `tiny_id`

## 特殊说明

- 由于频道的限制, 目前无法通过图片摘要查询到频道图片消息的详细信息, 所以通过频道消息收到的图片均会下载完整文件到 `images/guild-images`. (群图片转发不受此限制)
- 由于无法通过 `GlobalID` 放下频道消息的ID, 所以所有频道消息的 `message_id` 均为 `string` 类型
- `send_msg` API将无法发送频道消息
- `get_msg` API暂时无法获取频道消息
- `reply` 等消息类型暂不支持解析
- `at` 消息的 `target` 依然使用 `qq` 字段, 以保证一致性. 但内容为 `tiny_id`
- 所有事件的 `self_id` 均为 BOT 的QQ号. `tiny_id` 将放在 `self_tiny_id` 字段
- 遵循我们一贯的原则, 将不会支持主动加频道/主动拉人/红包相关消息类型
- 频道相关的API仅能在 `Android Phone` 和 `iPad` 协议上使用.
- 由于频道相关ID的数据类型均为 `uint64` , 为保证不超过某些语言的安全值范围, 在 `v1.0.0-beta8-fix3` 以后, 所有ID相关数据将转换为 `string` 类型, API调用 `uint64`
  或 `string` 均可接受.
- 为保证一致性, 所有频道接口返回的 `用户ID` 均命名为 `tiny_id`, 所有频道相关接口的 `用户ID` 入参均命名为 `user_id`

## API

### 获取频道系统内BOT的资料

终结点: `/get_guild_service_profile`

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `nickname`    | string | 昵称      |
| `tiny_id`     | string | 自身的ID   |
| `avatar_url`  | string | 头像链接   |

### 获取频道列表

终结点: `/get_guild_list`

**响应数据**

正常情况下响应 `GuildInfo` 数组, 未加入任何频道响应 `null`

GuildInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `guild_id`    | string | 频道ID      |
| `guild_name`     | string | 频道名称   |
| `guild_display_id`  | int64 | 频道显示ID, 公测后可能作为搜索ID使用  |

### 通过访客获取频道元数据

终结点: `/get_guild_meta_by_guest`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | string | 频道ID |

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `guild_id`    | string | 频道ID      |
| `guild_name`     | string | 频道名称   |
| `guild_profile`  | string | 频道简介  |
| `create_time`  | int64 | 创建时间  |
| `max_member_count`  | int64 | 频道人数上限  |
| `max_robot_count`  | int64 | 频道BOT数上限  |
| `max_admin_count`  | int64 | 频道管理员人数上限  |
| `member_count`  | int64 | 已加入人数  |
| `owner_id`  | string | 创建者ID  |

### 获取子频道列表

终结点: `/get_guild_channel_list`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | string | 频道ID |
| `no_cache` | bool  | 是否无视缓存 |

**响应数据**

正常情况下响应 `ChannelInfo` 数组, 未找到任何子频道响应 `null`

ChannelInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `owner_guild_id`    | string | 所属频道ID      |
| `channel_id`     | string | 子频道ID   |
| `channel_type`     | int32 | 子频道类型   |
| `channel_name`  | string | 子频道名称  |
| `create_time`  | int64 | 创建时间  |
| `creator_tiny_id`  | string | 创建者ID  |
| `talk_permission`  | int32 | 发言权限类型  |
| `visible_type`  | int32 | 可视性类型  |
| `current_slow_mode`  | int32 | 当前启用的慢速模式Key  |
| `slow_modes`  | []SlowModeInfo | 频道内可用慢速模式类型列表|

SlowModeInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `slow_mode_key`    | int32 | 慢速模式Key   |
| `slow_mode_text`     | string | 慢速模式说明   |
| `speak_frequency`  | int32 | 周期内发言频率限制  |
| `slow_mode_circle`  | int32 | 单位周期时间, 单位秒 |

已知子频道类型列表

| 类型          |  说明       |
| ------------- | ---------- |
| 1    | 文字频道  |
| 2     | 语音频道  |
| 5  |  直播频道  |
| 7  |  主题频道  |

### 获取频道成员列表

终结点: `/get_guild_member_list`

> 由于频道人数较多(数万), 请尽量不要全量拉取成员列表, 这将会导致严重的性能问题
>
> 尽量使用 `get_guild_member_profile` 接口代替全量拉取

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | string | 频道ID |
| `next_token` | string | 翻页Token |

> `next_token` 为空的情况下, 将返回第一页的数据, 并在返回值附带下一页的 `token`

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `members`    | []GuildMemberInfo | 成员列表   |
| `finished`    | bool | 是否最终页   |
| `next_token`    | string | 翻页Token   |

GuildMemberInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `tiny_id`    | string | 成员ID   |
| `title`    | string | 成员头衔   |
| `nickname`    | string | 成员昵称   |
| `role_id`    | string | 所在权限组ID   |
| `role_name`    | string | 所在权限组名称   |

> 默认情况下频道管理员的权限组ID为 `2`, 部分频道可能会另行创建, 需手动判断
>
> 此接口仅展现最新的权限组, 获取用户加入的所有权限组请使用 `get_guild_member_profile` 接口

### 单独获取频道成员信息

终结点: `/get_guild_member_profile`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | string | 频道ID |
| `user_id` | string | 用户ID |

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `tiny_id`    | string | 用户ID     |
| `nickname`    | string | 用户昵称     |
| `avatar_url`    | string | 头像地址     |
| `join_time`    | int64 | 加入时间     |
| `roles`    | []RoleInfo | 加入的所有权限组    |

RoleInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `role_id`    | string | 权限组ID     |
| `role_name`   | string | 权限组名称     |

### 发送信息到子频道

终结点: `/send_guild_channel_msg`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | string | 频道ID |
| `channel_id` | string | 子频道ID |
| `message` | Message | 消息, 与原有消息类型相同 |

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `message_id`    | string | 消息ID     |

### 获取话题频道帖子

终结点: `/get_topic_channel_feeds`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | string | 频道ID |
| `channel_id` | string | 子频道ID |

**响应数据**

返回 `FeedInfo` 数组

FeedInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `id`   | string | 帖子ID     |
| `channel_id`    | string | 子频道ID     |
| `guild_id`   | string | 频道ID     |
| `create_time`   | int64 | 发帖时间     |
| `title`   | string | 帖子标题     |
| `sub_title`   | string | 帖子副标题  |
| `poster_info`   | PosterInfo | 发帖人信息  |
| `resource`   | ResourceInfo | 媒体资源信息  |
| `resource.images`   | []FeedMedia | 帖子附带的图片列表 |
| `resource.videos`   | []FeedMedia | 帖子附带的视频列表 |
| `contents`   | []FeedContent | 帖子内容 |

PosterInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `tiny_id`   | string | 发帖人ID     |
| `nickname`    | string | 发帖人昵称     |
| `icon_url`   | string | 发帖人头像链接   |

FeedMedia:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `file_id`   | string | 媒体ID     |
| `pattern_id`    | string |   控件ID?(不确定)   |
| `url`   | string | 媒体链接   |
| `height`   | int32 | 媒体高度  |
| `width`   | int32 | 媒体宽度  |

FeedContent:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `type`   | string |  内容类型    |
| `data`    | Data |   内容数据   |

#### 内容类型列表:

|  类型  | 说明       |
|  ----- | ---------- |
| `text` |  文本   |
| `face` |  表情   |
| `at` |  At  |
| `url_quote` |  链接引用   |
| `channel_quote` |  子频道引用  |

#### 内容类型对应数据列表:

- `text`

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `text`   | string |  文本内容    |

- `face`

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `id`   | string |  表情ID    |

- `at`

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `id`   | string |  目标ID    |
| `qq`   | string |  目标ID, 为确保和 `array message` 的一致性保留    |

- `url_quote`

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `display_text`   | string |  显示文本    |
| `url`   | string |  链接    |

- `channel_quote`

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `display_text`   | string |  显示文本    |
| `guild_id`   | string |  频道ID    |
| `channel_id`   | string |  子频道ID    |

## 事件

### 收到频道消息

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `message`       | 上报类型       |
| `message_type` | string | `guild` | 消息类型       |
| `sub_type` | string | `channel` | 消息子类型       |
| `guild_id`    | string  |                | 频道ID           |
| `channel_id`    | string  |                | 子频道ID           |
| `user_id`     | string  |                | 消息发送者ID   |
| `message_id`     | string  |                | 消息ID  |
| `sender`     | Sender  |                | 发送者  |
| `message`     | Message  |                | 消息内容  |

> 注: 此处的 `Sender` 对象为保证一致性, `user_id` 为 `uint64` 类型, 并添加了 `string` 类型的 `tiny_id` 字段

### 频道消息表情贴更新

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `message_reactions_updated` | 消息类型       |
| `guild_id`    | string  |                | 频道ID           |
| `channel_id`    | string  |                | 子频道ID           |
| `user_id`     | string  |                | 操作者ID  |
| `message_id`     | string  |                | 消息ID  |
| `current_reactions`     | []ReactionInfo  |                | 当前消息被贴表情列表  |

ReactionInfo:

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `emoji_id` | string | 表情ID |
| `emoji_index` | int32 | 表情对应数值ID |
| `emoji_type` | int32 | 表情类型 |
| `emoji_name` | string | 表情名字 |
| `count` | int32 | 当前表情被贴数量 |
| `clicked` | bool | BOT是否点击 |

### 子频道信息更新

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `channel_updated` | 消息类型       |
| `guild_id`    | string  |                | 频道ID           |
| `channel_id`    | string  |                | 子频道ID           |
| `user_id`     | string  |                | 操作者ID  |
| `operator_id`     | string  |                | 操作者ID  |
| `old_info`     | ChannelInfo  |        | 更新前的频道信息  |
| `new_info`     | ChannelInfo  |        | 更新后的频道信息  |

### 子频道创建

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `channel_created` | 消息类型       |
| `guild_id`    | string  |                | 频道ID           |
| `channel_id`    | string  |                | 子频道ID           |
| `user_id`     | string  |                | 操作者ID  |
| `operator_id`     | string  |                | 操作者ID  |
| `channel_info`     | ChannelInfo  |        | 频道信息  |

### 子频道删除

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `channel_destroyed` | 消息类型       |
| `guild_id`    | string  |                | 频道ID           |
| `channel_id`    | string  |                | 子频道ID           |
| `user_id`     | string  |                | 操作者ID  |
| `operator_id`     | string  |                | 操作者ID  |
| `channel_info`     | ChannelInfo  |        | 频道信息  |