# 频道相关API

> 注意: QQ频道功能目前还在测试阶段, go-cqhttp 也在适配的初期阶段, 以下 `API` `Event` 的字段名可能存在错误并均有可能在后续版本修改/添加/删除.
> 目前仅供开发者测试以及适配使用

QQ频道相关功能的事件以及API

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

## API

### 获取频道系统内BOT的资料

终结点: `/get_guild_service_profile`

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `nickname`    | string | 昵称      |
| `tiny_id`     | uint64 | 自身的ID   |
| `avatar_url`  | string | 头像链接   |

### 获取频道列表

终结点: `/get_guild_list`

**响应数据**

正常情况下响应 `GuildInfo` 数组, 未加入任何频道响应 `null`

GuildInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `guild_id`    | uint64 | 频道ID      |
| `guild_name`     | string | 频道名称   |
| `guild_display_id`  | int64 | 频道显示ID, 公测后可能作为搜索ID使用  |

### 通过访客获取频道元数据

终结点: `/get_guild_meta_by_guest`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | uint64 | 频道ID |

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `guild_id`    | uint64 | 频道ID      |
| `guild_name`     | string | 频道名称   |
| `guild_profile`  | string | 频道简介  |
| `create_time`  | int64 | 创建时间  |
| `max_member_count`  | int64 | 频道人数上限  |
| `max_robot_count`  | int64 | 频道BOT数上限  |
| `max_admin_count`  | int64 | 频道管理员人数上限  |
| `member_count`  | int64 | 已加入人数  |
| `owner_id`  | uint64 | 创建者ID  |

### 获取子频道列表

终结点: `/get_guild_channel_list`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | uint64 | 频道ID |
| `no_cache` | bool  | 是否无视缓存 |

**响应数据**

正常情况下响应 `ChannelInfo` 数组, 未找到任何子频道响应 `null`

ChannelInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `owner_guild_id`    | uint64 | 所属频道ID      |
| `channel_id`     | uint64 | 子频道ID   |
| `channel_type`     | int32 | 子频道类型   |
| `channel_name`  | string | 之频道名称  |
| `create_time`  | int64 | 创建时间  |
| `creator_id`  | int64 | 创建者QQ号  |
| `creator_tiny_id`  | uint64 | 创建者ID  |
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

### 获取频道成员列表

终结点: `/get_guild_members`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | uint64 | 频道ID |

**响应数据**

> 注意: 类型内无任何成员将返回 `null`

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `members`    | []GuildMemberInfo | 普通成员列表      |
| `bots`     | []GuildMemberInfo | 机器人列表  |
| `admins`  | []GuildMemberInfo | 管理员列表  |

GuildMemberInfo:

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `tiny_id`    | uint64 | 成员ID      |
| `title`     | string | 成员头衔 |
| `nickname`  | string | 成员昵称  |
| `role`  | int32 | 成员权限  |

### 发送信息到子频道

终结点: `/send_guild_channel_msg`

**参数**

| 字段       | 类型  | 说明 |
| ---------- | ----- | ---- |
| `guild_id` | uint64 | 频道ID |
| `channel_id` | uint64 | 子频道ID |
| `message` | Message | 消息, 与原有消息类型相同 |

**响应数据**

| 字段          | 类型  | 说明       |
| ------------- | ----- | ---------- |
| `message_id`    | string | 消息ID     |

## 事件

### 收到频道消息

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `message`       | 上报类型       |
| `message_type` | string | `guild` | 消息类型       |
| `sub_type` | string | `channel` | 消息子类型       |
| `guild_id`    | uint64  |                | 频道ID           |
| `channel_id`    | uint64  |                | 子频道ID           |
| `user_id`     | uint64  |                | 消息发送者ID   |
| `message_id`     | string  |                | 消息ID  |
| `sender`     | Sender  |                | 发送者  |
| `message`     | Message  |                | 消息内容  |

### 频道消息表情贴更新

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `message_reactions_updated` | 消息类型       |
| `guild_id`    | uint64  |                | 频道ID           |
| `channel_id`    | uint64  |                | 子频道ID           |
| `user_id`     | uint64  |                | 操作者ID  |
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
| `guild_id`    | uint64  |                | 频道ID           |
| `channel_id`    | uint64  |                | 子频道ID           |
| `user_id`     | uint64  |                | 操作者ID  |
| `operator_id`     | uint64  |                | 操作者ID  |
| `old_info`     | ChannelInfo  |        | 更新前的频道信息  |
| `new_info`     | ChannelInfo  |        | 更新后的频道信息  |

### 子频道创建

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `channel_created` | 消息类型       |
| `guild_id`    | uint64  |                | 频道ID           |
| `channel_id`    | uint64  |                | 子频道ID           |
| `user_id`     | uint64  |                | 操作者ID  |
| `operator_id`     | uint64  |                | 操作者ID  |
| `channel_info`     | ChannelInfo  |        | 频道信息  |

### 子频道删除

**上报数据**

| 字段          | 类型   | 可能的值       | 说明           |
| ------------- | ------ | -------------- | -------------- |
| `post_type`   | string | `notice`       | 上报类型       |
| `notice_type` | string | `channel_destroyed` | 消息类型       |
| `guild_id`    | uint64  |                | 频道ID           |
| `channel_id`    | uint64  |                | 子频道ID           |
| `user_id`     | uint64  |                | 操作者ID  |
| `operator_id`     | uint64  |                | 操作者ID  |
| `channel_info`     | ChannelInfo  |        | 频道信息  |