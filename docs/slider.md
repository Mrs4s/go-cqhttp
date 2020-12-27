# 滑块验证码

由于TX最新的限制, 所有协议在陌生设备/IP登录时都有可能被要求通过滑块验证码, 否则将会出现 `当前上网环境异常` 的错误. 目前我们准备了两个临时方案应对该验证码.

> 如果您有一台运行Windows的PC/Server 并且不会抓包操作, 我们建议直接使用方案B

## 方案A: 自行抓包

由于滑块验证码和QQ本体的协议独立, 我们无法直接处理并提交. 需要在浏览器通过后抓包并获取 `Ticket` 提交.

该方案为具体的抓包教程, 如果您已经知道如何在浏览器中抓包. 可以略过接下来的文档并直接抓取 `cap_union_new_verify` 的返回值, 提取 `Ticket` 并在命令行提交.

首先获取滑块验证码的地址, 并在浏览器中打开. 这里以 *Microsoft Edge* 浏览器为例, *Chrome* 同理. 

![image.png](https://i.loli.net/2020/12/27/yXdomOnQ8tkauMe.png)

首先选择 `1` 并提取链接在浏览器中打开

![image.png](https://i.loli.net/2020/12/27/HYhmZv1wARMV7Uq.png)

![image.png](https://i.loli.net/2020/12/27/otk9Hz7lBCaRFMV.png)

此时不要滑动验证码, 首先按下 `F12` (键盘右上角退格键上方) 打开 *开发者工具*

![image.png](https://i.loli.net/2020/12/27/JDioadLPwcKWpt1.png)

点击 `Network` 选项卡 (在某些浏览器它可能叫做 `网络`)

![image.png](https://i.loli.net/2020/12/27/qEzTB5jrDZUWSwp.png)

点开 `Filter` (箭头) 按钮以确定您能看到下面的工具栏, 勾选 `Preserve log`(红框)

此时可以滑动并通过验证码

![image.png](https://i.loli.net/2020/12/27/Id4hxzyDprQuF2G.png)

回到 *开发者工具*, 我们可以看到已经有了一个请求.

![image.png](https://i.loli.net/2020/12/27/3C6Y2XVKBRv1z9E.png)

此时如果有多个请求, 请不要慌张. 看到上面的 `Filter` 没? 此时在 `Filter` 输入框中输入 `cap_union_new`, 就应该只剩一个请求了.

然后点击该请求. 点开 `Preview` 选项卡 (箭头):  

![image.png](https://i.loli.net/2020/12/27/P1VtxRWpjY8524Z.png)

此时就能看到一个标准的 `JSON`, 复制 `ticket` 字段并回到 `go-cqhttp` 粘贴. 即可通过滑块验证.

如果您看到这里还是不会如何操作, 没关系! 我们还准备了方案B.

## 方案B: 使用专用工具

此方案需要您有一台可以操作的 `Windows` 电脑.

首先下载工具:  [蓝奏云](https://wws.lanzous.com/i2vn0jrofte) [Google Drive](https://drive.google.com/file/d/1peMDHqgP8AgWBVp5vP-cfhcGrb2ksSrE/view?usp=sharing)

解压并打开工具: 

![image.png](https://i.loli.net/2020/12/27/winG4SkxhgLoNDZ.png)

打开 `go-cqhttp` 并选择 `2`:

![image.png](https://i.loli.net/2020/12/27/yXdomOnQ8tkauMe.png)

复制 `ID` 并前往工具粘贴:

![image.png](https://i.loli.net/2020/12/27/fIwXx5nN9r8Zbc7.png)

![image.png](https://i.loli.net/2020/12/27/WZsTCyGwSjc9mb5.png)

点击 `OK` 并处理滑块, 完成即可登录成功. (OK可能反应稍微慢点, 请不要多次点击)

![image.png](https://i.loli.net/2020/12/27/UnvAuxreijYzgLC.png)

