# 使用说明

1. 设置配置信息：
    + 如果使用扫码登录，就不需要配置`account_uin`和`account_password`。
    + 如果要使用账号密码登录，就设置`account_uin`和`account_password`吧。账号密码登录可能会报 `0xa`的错误，多重试一下就可以了。
    + 默认开启了http的5700端口。具体配置请在`servers_http`、`servers_ws`、`servers_ws-reverse`配置中配置。采用了json压缩的字符串的配置形式。照着默认的demo改就行了。
    + 字段对应`go-cqhttp`的yaml字段。

2. 配置完成后，在`dashboard`中点击`登录`。
    + 登录过程中可能遇到`扫码`、`输入手机验证码`等输入操作。

3. 登录成功，就可以放心使用了。再次登录的话会读取token实现快速登录。重新启动程序也会自动尝试登录。
    + Just for fun!~