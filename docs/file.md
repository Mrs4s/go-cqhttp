# 文件

go-cqhttp 默认生成的文件树如下所示:

```
.
├── go-cqhttp
├── config.yml
├── device.json
├── logs
│   └── xx-xx-xx.log
└── data
    ├── images
    │   └── xxxx.image
    └── levleldb
```

| 文件         | 用途                 |
| ------------ | -------------------- |
| go-cqhttp    | go-cqhttp 可执行文件 |
| config.yml | 运行配置文件         |
| device.json  | 虚拟设备配置文件     |
| logs         | 日志存放目录         |
| data         | 数据目录             |
| data/leveldb | 数据库目录           |
| data/images  | 图片缓存目录         |
| data/voices  | 语音缓存目录         |
| data/videos  | 视频缓存目录         |
| data/cache   | 发送图片缓存目录     |

## 图片缓存文件

出于性能考虑，go-cqhttp 并不会将图片源文件下载到本地，而是生成一个可以和 QQ 服务器对应的缓存文件 (.image)，该缓存文件结构如下:

| 偏移            | 类型     | 说明                 |
| --------------- | -------- | -------------------- |
| 0x00            | [16]byte | 图片源文件 MD5 HASH  |
| 0x10            | uint32   | 图片源文件大小       |
| 0x14            | string   | 图片原名(QQ内部ID) |
| 0x14 + 原名长度 | string   | 图片下载链接         |

# MIME

启用MINE检查可以及时发现媒体资源格式错误引起的上传失败(通常表现为，请求网页图片，但服务端返回404.html)

在配置文件中设置 `skip-mine-scan: false`后 ，go-cqhttp 会在上传媒体资源(视频暂不支持)前对MIME进行检查，
详细允许类型如下所示:

图片：
> image/bmp  
> image/gif  
> image/jpeg  
> image/png  
> image/webp  

语音:
> audio/aac  
> audio/aiff  
> audio/amr  
> audio/ape  
> audio/flac  
> audio/midi  
> audio/mp4  
> audio/mpeg  
> audio/ogg  
> audio/wav  
> audio/x-m4a  

