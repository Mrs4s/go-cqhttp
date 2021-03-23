# Contributing to go-cqhttp

想要成为 go-cqhttp 的 Contributor? Awesome!

这个页面提供了一些 Tips ，可能对您的开发提供一些帮助.

## 开发环境准备

go-cqhttp 使用了 `golangci-lint` 检查可能的问题，规范代码风格，为了减少不必要的麻烦，
我们推荐在开发环境中安装 `golangci-lint` 工具.

```shell
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

在提交代码前运行 `golangci-lint` 检查你的代码:

```shell
golangci-lint run
```

**注意**: `golangci-lint` 需要 `diff` 工具，在 windows 环境中，你可能需要使用 `Git Bash` 运行。

## Pull requests

首先，为了方便项目管理，请将您的 PR 推送至**dev**分支。

### 检查 issue 列表

不管你是已经明确了要提交什么代码，还是正在寻找一个想法，你都应该先到 issue 列表看一下。
如果在 issue 中找到了感兴趣的，请在 issue 表明正在对这个 issue 进行开发。

### 项目结构

下面是 go-cqhttp 项目结构的简单介绍.

<table class="tg">
  <tr>
    <td>coolq</td>
    <td>
      包含与 MiraiGo 交互部分， CQ码解析等部分
    </td>
  </tr>
  <tr>
    <td>server</td>
    <td>
      包含 http，ws 通信的实现部分
    </td>
  </tr>
  <tr>
    <td>global</td>
    <td>
      一个<del>实用的</del>工具包
    </td>
  </tr>
  <tr>
    <td>docs</td>
    <td>
      使用教程与文档
    </td>
  </tr>
</table>

## 社区准则
为了让社区保持强大，不断发展，我们向整个社区提出了一些通用准则：

**友善**：对社区成员要礼貌，尊重和礼貌。 请不要在社区中发布任何有关种族歧视、性别歧视、
地域歧视、人格侮辱等言论。

**鼓励参与**：在社区中讲礼貌的每个人都受到欢迎，无论他们的贡献程度如何，
我们鼓励一切人参与(不一定需要提交代码) `go-cqhttp` 的开发。

**紧贴主题**：请避免主题外的讨论。当您更新或回复时， 可能会给大量人员发送邮件，
请牢记，没有人喜欢垃圾邮件。
