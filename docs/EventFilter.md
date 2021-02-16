# 事件过滤器

在go-cqhttp同级目录下新建`filter.json`文件即可开启事件过滤器，启动时会读取该文件中定义的过滤规则（使用 JSON 编写），若文件不存在，或过滤规则语法错误，则不会启用事件过滤器。
事件过滤器会处理所有事件(包括心跳事件在内的元事件),请谨慎使用！！

注意: 与客户端建立连接的握手事件**不会**经过事件过滤器

## 示例

这节首先给出一些示例，演示过滤器的基本用法，下一节将给出具体语法说明。

### 过滤所有事件

```json
{
    ".not": {}
}
```

### 只上报以「!!」开头的消息

```json
{
    "raw_message": {
        ".regex": "^!!"
    }
}
```

### 只上报群组的非匿名消息

```json
{
    "message_type": "group",
    "anonymous": {
        ".eq": null
    }
}
```

### 只上报私聊或特定群组的非匿名消息

```json
{
    ".or": [
        {
            "message_type": "private"
        },
        {
            "message_type": "group",
            "group_id": {
                ".in": [
                    123456
                ]
            },
            "anonymous": {
                ".eq": null
            }
        }
    ]
}
```

### 只上报群组 11111、22222、33333 中不是用户 12345 发送的消息，以及用户 66666 发送的所有消息

```json
{
    ".or": [
        {
            "group_id": {
                ".in": [11111, 22222, 33333]
            },
            "user_id": {
                ".neq": 12345
            }
        },
        {
            "user_id": 66666
        }
    ]
}
```

### 一个更复杂的例子

```json
{
    ".or": [
        {
            "message_type": "private",
            "user_id": {
                ".not": {
                    ".in": [11111, 22222, 33333]
                },
                ".neq": 44444
            }
        },
        {
            "message_type": {
                ".regex": "group|discuss"
            },
            ".or": [
                {
                    "group_id": 12345
                },
                {
                    "raw_message": {
                        ".contains": "通知"
                    }
                }
            ]
        }
    ]
}
```

## 语法说明

过滤规则最外层是一个 JSON 对象，其中的键，如果以 `.`（点号）开头，则表示运算符，其值为运算符的参数，如果不以 `.` 开头，则表示对事件数据对象中相应键的过滤。过滤规则中任何一个对象，只有在它的所有项都匹配的情况下，才会让事件通过（等价于一个 `and` 运算）；其中，不以 `.` 开头的键，若其值不是对象，则只有在这个值和事件数据相应值相等的情况下，才会通过（等价于一个 `eq` 运算符）。

下面列出所有运算符（「要求的参数类型」是指运算符的键所对应的值的类型，「可作用于的类型」是指在过滤时事件对象相应值的类型）：

| 运算符      | 要求的参数类型             | 可作用于的类型                                        |
| ----------- | -------------------------- | ----------------------------------------------------- |
| `.not`      | object                     | 任何                                                  |
| `.and`      | object                     | 若参数中全为运算符，则任何；若不全为运算符，则 object |
| `.or`       | array（数组元素为 object） | 任何                                                  |
| `.eq`       | 任何                       | 任何                                                  |
| `.neq`      | 任何                       | 任何                                                  |
| `.in`       | string/array               | 若参数为 string，则 string；若参数为 array，则任何    |
| `.contains` | string                     | string                                                |
| `.regex`    | string                     | string                                                |


## 过滤时的事件数据对象

过滤器在go-cqhttp构建好事件数据后运行，各事件的数据字段见[OneBot标准]( https://github.com/howmanybots/onebot/blob/master/v11/specs/event/README.md )。

这里有几点需要注意：

- `message` 字段在运行过滤器时和上报信息类型相同（见 [消息格式]( https://github.com/howmanybots/onebot/blob/master/v11/specs/message/array.md )）
- `raw_message` 字段为未经**CQ码**处理的原始消息字符串，这意味着其中可能会出现形如 `[CQ:face,id=123]` 的 CQ 码
