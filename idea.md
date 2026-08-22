# Herdr Web Bridge（Windows MVP）工程任务书

你现在位于一个新的、独立的项目目录中。请从零开发一个名为 **Herdr Web Bridge** 的 Windows 工具，用于把 Microsoft Edge 中的网页任务与 Herdr 项目工作区连接起来。

不要修改 Herdr 或 Herdr Plus 的源代码，不要向 GitHub 推送，不要覆盖用户现有配置。所有安装和配置修改必须可审计、可卸载、可回滚。

---

## 一、用户需求

用户使用 Herdr 按项目文件夹管理工作，例如：

```text
T:\VirtualDNA
T:\简历优化
T:\毕业论文
T:\中医药AI
```

同时会在 Microsoft Edge 中打开：

* ChatGPT 对话；
* 自己开发的简历优化工具；
* 其他项目相关网页。

需要实现以下工作流：

```text
在 Edge 中打开网页
→ 将当前网页绑定到某个 Herdr 项目文件夹
→ Herdr 项目中出现网页快捷入口
→ 从 Herdr 点击后，优先跳转到已打开的网页标签
→ 若标签不存在，则重新打开网页
→ 网页开始生成、完成、出错时，状态同步到 Herdr
→ 网页完成但尚未查看时，由 Herdr 发系统通知
→ 用户回到网页查看后，清除 Herdr 中的未读状态
```

第一版只需要支持：

1. Windows 10/11；
2. Microsoft Edge；
3. ChatGPT；
4. 用户自己的网页工具；
5. 通用网页的手动绑定与手动状态测试。

Chrome 可以保持代码兼容，但不作为第一版必须验收的平台。

---

## 二、先做环境审计，不要直接写代码

首先执行并记录：

```powershell
herdr --version
herdr status
herdr workspace list
herdr api schema --json
herdr plugin list
herdr plugin action list --plugin cloudmanic.herdr-plus
herdr plugin config-dir cloudmanic.herdr-plus
```

同时检查：

```powershell
go version
node --version
npm --version
git --version
```

把结果写入：

```text
docs/environment-audit.md
```

注意：

* 某条命令不存在时，不要猜测；
* 以当前电脑安装的 Herdr CLI 和 `herdr api schema --json` 为准；
* 记录 Herdr 版本、支持的 API、Herdr Plus 是否已安装；
* 不要自动安装或升级 Herdr；
* 不要自动安装 Herdr Plus；
* 缺少依赖时，继续完成能够完成的部分，并在报告中列出阻塞项。

然后创建：

```text
PLAN.md
```

列出阶段、文件结构、验收标准和风险，再开始编码。

---

## 三、冻结的 MVP 范围

第一版必须实现以下功能，不要在这些功能通过之前扩展范围。

### 1. 浏览器扩展

开发一个 Manifest V3 Edge 扩展。

扩展弹窗至少包含：

```text
当前网页标题
当前网页地址
检测到的网站类型
当前绑定的 Herdr 项目
选择 Herdr 项目
绑定
解除绑定
测试“运行中”
测试“已完成”
测试“出错”
打开对应 Herdr 工作区
```

绑定流程：

1. 扩展读取当前标签页 URL 和标题；
2. 通过 Native Messaging 向本地程序请求 Herdr 工作区列表；
3. 显示工作区名称和文件夹路径；
4. 用户选择一个工作区；
5. 保存绑定；
6. 为该项目生成 Herdr 网页快捷入口。

绑定的长期标识必须是：

```text
项目文件夹绝对路径
```

不要只保存临时的 `workspace_id`。Herdr 工作区 ID可以作为运行时缓存，但每次同步前应重新根据文件夹路径解析当前工作区。

---

### 2. Windows 本地桥接程序

使用 **Go** 开发一个单文件可执行程序：

```text
herdr-web-bridge.exe
```

它同时承担：

1. Edge Native Messaging Host；
2. Herdr CLI 调用器；
3. 绑定配置管理；
4. 本地 IPC 服务；
5. 命令行入口。

至少支持以下命令：

```powershell
herdr-web-bridge doctor
herdr-web-bridge list-bindings
herdr-web-bridge list-workspaces
herdr-web-bridge open --binding <binding_id>
herdr-web-bridge notify-test
herdr-web-bridge install-status
```

`doctor` 必须检查：

```text
Herdr CLI 是否存在
Herdr 是否正在运行
能否读取 workspace list
能否发送 Herdr 测试通知
Herdr Plus 是否存在
Native Messaging manifest 是否注册
Edge 扩展是否已连接
本地 IPC 是否可用
绑定配置是否有效
```

输出中不得泄露用户浏览内容、Cookie、Token 或其他凭据。

---

### 3. Native Messaging

扩展与本地程序使用 Edge Native Messaging 通信。

必须遵循：

```text
UTF-8 JSON
消息前加本机字节序的 32 位长度
stdout 只能输出 Native Messaging 数据帧
日志只能写 stderr 或日志文件
```

扩展 service worker 使用：

```javascript
chrome.runtime.connectNative(...)
```

content script 不得直接调用 Native Messaging。消息路径必须是：

```text
content script
→ extension service worker
→ Native Messaging Host
→ Herdr
```

Native Messaging Host manifest 的 `allowed_origins` 只能包含本扩展的 ID。

不要运行网页传来的任意命令。

---

## 四、建议的项目结构

采用类似结构：

```text
herdr-web-bridge/
├── README.md
├── PLAN.md
├── go.mod
├── package.json
├── docs/
│   ├── environment-audit.md
│   ├── ARCHITECTURE.md
│   ├── QUICKSTART_WINDOWS.md
│   ├── SECURITY.md
│   ├── TEST_REPORT.md
│   └── KNOWN_LIMITATIONS.md
├── extension/
│   ├── manifest.json
│   ├── src/
│   │   ├── background.ts
│   │   ├── content.ts
│   │   ├── popup.ts
│   │   ├── options.ts
│   │   ├── protocol.ts
│   │   ├── state-machine.ts
│   │   └── adapters/
│   │       ├── chatgpt.ts
│   │       ├── custom-tool.ts
│   │       └── generic.ts
│   ├── popup.html
│   ├── options.html
│   ├── icons/
│   └── tests/
├── cmd/
│   └── herdr-web-bridge/
│       └── main.go
├── internal/
│   ├── bindings/
│   ├── herdr/
│   ├── ipc/
│   ├── native/
│   ├── protocol/
│   ├── quickactions/
│   ├── security/
│   └── logging/
├── installer/
│   ├── install.ps1
│   ├── uninstall.ps1
│   ├── register-native-host.ps1
│   └── native-host-manifest.json.template
├── examples/
│   ├── custom-tool-integration.html
│   └── custom-tool-integration.js
├── scripts/
│   ├── build.ps1
│   ├── test.ps1
│   └── package.ps1
└── dist/
```

允许根据实际工程需要调整，但必须保持：

* 扩展与本地程序分离；
* 协议定义集中；
* 网站 adapter 分离；
* 安装、卸载可重复执行；
* 测试与产品代码分离。

---

## 五、绑定数据模型

绑定配置建议存放在：

```text
%LOCALAPPDATA%\HerdrWebBridge\bindings.json
```

数据结构至少包括：

```json
{
  "schemaVersion": 1,
  "bindings": [
    {
      "id": "稳定的UUID",
      "projectPath": "T:\\VirtualDNA",
      "projectLabel": "VirtualDNA",
      "url": "https://chatgpt.com/c/...",
      "urlMatch": "exact",
      "pageTitle": "VirtualDNA Run 3",
      "adapter": "chatgpt",
      "notificationsEnabled": true,
      "createdAt": "ISO-8601时间",
      "updatedAt": "ISO-8601时间",
      "lastState": "idle",
      "lastEventId": ""
    }
  ]
}
```

要求：

* 使用原子写入；
* 写入前生成备份；
* 对路径进行 Windows 规范化；
* 支持盘符大小写差异；
* 不保存网页正文；
* 不保存登录凭据；
* 不保存 Cookie；
* 不保存 ChatGPT 消息内容；
* URL 查询参数可能含敏感信息，日志中默认只记录 origin、pathname 和哈希后的 binding ID。

---

## 六、Herdr 工作区解析

每次需要更新 Herdr 时：

1. 调用：

```powershell
herdr workspace list
```

2. 解析 JSON；
3. 用规范化后的绝对路径匹配 `projectPath`；
4. 找到当前对应的 `workspace_id`；
5. 再调用 Herdr 状态或通知命令。

多个工作区使用同一个目录时：

1. 优先当前聚焦的工作区；
2. 其次优先 label 与 `projectLabel` 一致的工作区；
3. 仍然冲突时，不要静默选择，返回 `ambiguous_workspace`。

不得把某次获取到的 workspace ID永久写死。

---

## 七、Herdr 状态同步

第一版使用 workspace metadata，不要创建伪终端，也不要使用 `pane report-agent` 冒充真正的 CLI Agent。

状态映射：

| 网页状态          | Herdr 显示       |
| ------------- | -------------- |
| `idle`        | 清除网页状态         |
| `running`     | ` 正在生成：网页名称` |
| `done_unread` | `✅ 等待查看：网页名称`  |
| `viewed`      | 清除未读状态         |
| `error`       | `⚠️ 网页出错：网页名称` |
| `unknown`     | ` 状态未知：网页名称` |

使用类似命令，但必须先根据本机 Herdr schema 验证参数：

```powershell
herdr workspace report-metadata <workspace_id> `
  --source herdr-web-bridge `
  --token web_status="✅ 等待查看：VirtualDNA ChatGPT" `
  --seq <单调递增序号> `
  --ttl-ms 86400000
```

清除状态：

```powershell
herdr workspace report-metadata <workspace_id> `
  --source herdr-web-bridge `
  --clear-token web_status `
  --seq <单调递增序号>
```

要求：

* `source` 固定为 `herdr-web-bridge`；
* 每个 binding 维护单调递增的 seq；
* 状态文本控制在 80 个字符以内；
* 同一个完成事件只能通知一次；
* 重复 DOM 变化不得产生通知风暴；
* 状态同步失败时重试，采用有上限的指数退避；
* Herdr 不在线时不要无限重试。

---

## 八、Herdr 通知

当状态从：

```text
running → done_unread
```

时，调用：

```powershell
herdr notification show "网页任务已完成" `
  --body "VirtualDNA：ChatGPT 已生成结果，等待查看" `
  --sound done
```

发生错误时：

```powershell
herdr notification show "网页任务出错" `
  --body "简历优化：生成失败，请返回网页检查" `
  --sound request
```

通知规则：

* 只有绑定的网页才能发通知；
* 初次安装或初次打开历史对话时不得误报完成；
* 一个 `eventId` 只能通知一次；
* 用户正在看当前网页时，完成后不必发送“未查看”通知，可直接转为 `viewed`；
* Herdr 通知调用失败时，允许扩展使用 `chrome.notifications` 做兜底；
* Herdr 恢复后重新同步最终状态，但不要重复通知。

---

## 九、从 Herdr 一键回到网页

第一版使用 Herdr Plus 的项目级 Quick Actions。

每个绑定在项目目录下生成一个 TOML 文件：

```text
<projectPath>\.herdr-plus\quick-actions\
```

文件名必须经过安全 slug 处理，例如：

```text
open-web-virtualdna-chatgpt.toml
```

Quick Action 不要直接拼接不可信 URL 到任意 Shell 命令中。它应调用：

```powershell
herdr-web-bridge.exe open --binding <binding_id>
```

由本地程序根据绑定 ID读取受信任配置中的 URL。

`open` 的行为：

1. 尝试连接正在运行的桥接 IPC；
2. 通知 Edge 扩展查找对应 binding；
3. 扩展查询现有标签页；
4. 找到匹配标签页时：

   * 激活对应标签；
   * 激活对应浏览器窗口；
5. 没找到时：

   * 创建新标签；
   * 打开绑定 URL；
6. 扩展未连接或 IPC 不可用时：

   * 使用 Windows 默认浏览器打开 URL；
   * 返回清晰的 fallback 状态。

不要依赖网页标题匹配。优先使用：

```text
binding ID
精确 URL
规范化 URL
已记录的 tabId 仅作为临时缓存
```

Herdr Plus 未安装时：

* 不自动安装；
* 浏览器绑定仍应保存；
* 状态通知仍应工作；
* UI 显示“尚未生成 Herdr 快捷入口”；
* `doctor` 输出建议安装命令；
* 不要把缺少 Herdr Plus 当成整个桥接程序失败。

---

## 十、本地 IPC

为了让 Herdr Quick Action 可以命令浏览器扩展聚焦标签页，需要本地程序提供仅限当前用户访问的 IPC。

Windows 优先采用命名管道，例如：

```text
\\.\pipe\herdr-web-bridge-<当前用户标识>
```

要求：

* 仅当前用户可访问；
* 限制消息长度；
* 使用 JSON schema；
* 不接受任意命令；
* 不接受任意文件路径；
* 不接受任意 Shell；
* 命令白名单仅包括：

```text
open_binding
list_bindings
ping
status
```

Native Messaging Host 与 CLI 模式可以使用同一个 Go 可执行程序，但必须正确区分：

```text
浏览器启动的 Native Host 模式
用户执行的 CLI 模式
IPC Broker 模式
```

如单进程架构在 Manifest V3 生命周期下不可靠，可以采用：

```text
常驻 broker
+ Native Messaging adapter
+ CLI client
```

但不要引入 Electron 或大型运行时。

第一版只要求支持一个活动 Edge 用户配置文件。把多浏览器、多配置文件列入 `KNOWN_LIMITATIONS.md`。

---

## 十一、浏览器扩展权限

遵循最小权限原则。

建议权限：

```json
{
  "permissions": [
    "storage",
    "tabs",
    "nativeMessaging",
    "notifications",
    "scripting"
  ]
}
```

ChatGPT 使用明确的 host permission。

自定义网站使用 `optional_host_permissions`，只在用户绑定具体 origin 时请求权限。

不得申请：

```text
全部网站永久读取权限
浏览历史权限
下载历史权限
Cookie 权限
剪贴板读取权限
```

只有绑定的网站可以启动状态监控。

---

## 十二、ChatGPT adapter

单独实现：

```text
extension/src/adapters/chatgpt.ts
```

目标是识别：

```text
idle
running
done_unread
viewed
error
unknown
```

实现原则：

1. 使用 `MutationObserver`；
2. 使用多个信号，不要只依赖一个 CSS selector；
3. 可以参考：

   * 停止生成按钮；
   * 重试按钮；
   * assistant 消息数量变化；
   * 流式区域状态；
   * 页面可见性；
   * 页面焦点；
4. 不读取和传输回答正文；
5. 不发送用户输入；
6. 不发送对话内容；
7. 只发送：

   * binding ID；
   * 状态；
   * 页面标题；
   * URL；
   * 时间；
   * 简短的非正文状态说明。

状态机要求：

```text
初次加载历史对话
→ idle
→ 不通知

检测到本轮开始生成
→ running

running 状态消失并稳定至少 800ms
且确认出现了新的 assistant 输出结构
→ done_unread

标签页可见且窗口聚焦持续至少 1500ms
→ viewed

出现明确失败或重试状态
→ error
```

必须防止：

* 网页初次加载时把已有回答误认为刚完成；
* 页面重渲染导致重复通知；
* ChatGPT 页面结构轻微变化后疯狂通知；
* 用户正在当前标签页观看时仍产生未读提醒；
* selector 失效时继续武断判断。

selector 信号不足时进入：

```text
unknown
```

并在扩展 UI 中显示 adapter health，而不是猜测成功。

为 adapter 编写 DOM fixture 测试和状态机测试。

---

## 十三、自定义网页工具 adapter

为用户自己开发的简历工具定义稳定接口。

示例页面代码：

```javascript
window.postMessage(
  {
    source: "herdr-web-bridge",
    version: 1,
    type: "task-status",
    taskId: "resume-2026-001",
    state: "running",
    title: "简历优化",
    message: "正在生成岗位定制简历"
  },
  window.location.origin
);
```

完成：

```javascript
window.postMessage(
  {
    source: "herdr-web-bridge",
    version: 1,
    type: "task-status",
    taskId: "resume-2026-001",
    state: "done",
    title: "简历优化",
    message: "PDF 已生成"
  },
  window.location.origin
);
```

错误：

```javascript
window.postMessage(
  {
    source: "herdr-web-bridge",
    version: 1,
    type: "task-status",
    taskId: "resume-2026-001",
    state: "error",
    title: "简历优化",
    message: "PDF 生成失败"
  },
  window.location.origin
);
```

content script 必须验证：

```text
event.source === window
event.origin === location.origin
source 字段完全匹配
version 支持
type 在白名单中
state 在白名单中
taskId 长度受限
title/message 长度受限
当前 origin 已由用户授权和绑定
```

页面不得通过消息传递：

* 命令；
* 本地路径；
* 可执行文件；
* PowerShell；
* Herdr workspace ID；
* Native Messaging Host 名称。

提供：

```text
examples/custom-tool-integration.html
examples/custom-tool-integration.js
```

以及中文接入说明。

---

## 十四、通用网页 adapter

`generic.ts` 不尝试自动理解任意网页。

它只提供：

* 绑定网页；
* 从 Herdr 打开网页；
* 手动测试运行状态；
* 手动标记完成；
* 手动清除状态。

不要通过页面标题中出现数字就武断判断“有通知”。

---

## 十五、安全要求

必须写入：

```text
docs/SECURITY.md
```

至少覆盖：

1. 网页内容不可信；
2. Native Messaging 消息需要 schema 验证；
3. Quick Action 不得执行网页提供的命令；
4. 只能使用 binding ID查询本地受信任配置；
5. 路径规范化；
6. URL scheme 白名单；
7. 仅允许：

   * `https:`
   * 用户明确允许的 `http://localhost`
   * 用户明确允许的 `http://127.0.0.1`
8. 默认拒绝：

   * `file:`
   * `javascript:`
   * `data:`
   * `ftp:`
   * `shell:`
   * 自定义协议；
9. 日志脱敏；
10. 不保存 Cookie、Token、消息正文；
11. 命名管道访问控制；
12. Native Host `allowed_origins` 限制；
13. 通知去重；
14. 配置文件原子写入和备份；
15. 卸载时不删除用户项目文件，除非明确确认。

禁止使用以下不安全方式：

```text
Invoke-Expression
eval
将网页文本直接拼接为 cmd/powershell
开放无鉴权的 0.0.0.0 HTTP 服务
允许网页指定任意 executable
允许网页指定任意 projectPath
```

---

## 十六、安装和卸载

实现：

```powershell
.\installer\install.ps1
.\installer\uninstall.ps1
```

安装脚本负责：

1. 检查 Windows；
2. 检查 Herdr；
3. 检查 Edge；
4. 安装 `herdr-web-bridge.exe` 到用户目录；
5. 创建配置目录；
6. 注册 Edge Native Messaging Host；
7. 安装或输出扩展加载目录；
8. 验证扩展 ID；
9. 写入 `allowed_origins`；
10. 执行端到端连接测试；
11. 不需要管理员权限；
12. 不修改系统级注册表，优先使用当前用户范围；
13. 所有修改记录到安装日志。

扩展 ID需要稳定。

优先研究并采用可靠的稳定 ID方案；若无法安全自动化，则安装脚本接受：

```powershell
-ExtensionId <id>
```

并在 `QUICKSTART_WINDOWS.md` 中给出清晰的 Edge“加载解压缩扩展”步骤。

卸载脚本负责：

* 注销 Native Messaging Host；
* 删除本程序安装目录；
* 保留或询问是否保留 bindings；
* 默认不删除项目中的 `.herdr-plus/quick-actions/` 文件；
* 提供 `-RemoveGeneratedQuickActions` 显式选项；
* 不删除 Herdr；
* 不删除 Herdr Plus；
* 不删除项目文件。

---

## 十七、测试要求

### Go 单元测试

至少覆盖：

```text
Native Messaging 长度前缀读写
畸形 JSON
超长消息
路径规范化
盘符大小写
workspace 匹配
workspace 冲突
binding 原子写入
binding ID查找
URL scheme 验证
Herdr CLI 参数构造
Quick Action 文件名清理
Quick Action 命令转义
通知去重
seq 单调递增
IPC 白名单
日志脱敏
```

运行：

```powershell
go test ./...
go test -race ./...
go vet ./...
```

Windows 环境下 `-race` 不可用或失败时，记录原因，不要伪造通过。

### 扩展测试

使用 Vitest 或等价轻量测试框架，至少覆盖：

```text
ChatGPT 初次加载不通知
running → done_unread
重复 Mutation 不重复通知
可见并聚焦后变为 viewed
error 状态
unknown 状态
自定义 postMessage schema
非法 origin
非法 state
超长字段
service worker 重连
Native Host 断开
通知 fallback
```

### 集成测试

实现可重复运行的测试：

```text
mock Herdr CLI
mock Native Messaging
mock IPC
mock ChatGPT DOM fixture
自定义工具测试页面
```

提供：

```powershell
.\scripts\test.ps1
```

---

## 十八、人工验收场景

必须完成并记录以下验收结果。

### 场景 A：绑定 ChatGPT

1. Herdr 中存在 `VirtualDNA` 工作区；
2. Edge 打开一个 ChatGPT 对话；
3. 扩展选择 `VirtualDNA`；
4. 点击绑定；
5. `bindings.json` 出现绑定；
6. 项目目录出现 Quick Action 文件；
7. Herdr Plus 能看到该快捷入口。

### 场景 B：从 Herdr 返回网页

1. ChatGPT 标签已经打开；
2. 在 Herdr 中运行对应 Quick Action；
3. Edge 窗口被激活；
4. 已有标签被聚焦；
5. 不额外打开重复标签。

再关闭该标签并运行 Quick Action：

1. Edge 打开新标签；
2. 进入正确 URL。

### 场景 C：ChatGPT 完成通知

1. 在绑定的 ChatGPT 对话中发送测试请求；
2. 生成期间 Herdr 显示 `正在生成`；
3. 切换到其他窗口；
4. ChatGPT 完成；
5. Herdr 显示 `等待查看`；
6. 系统收到 Herdr 通知；
7. 同一个结果只通知一次；
8. 回到网页并查看；
9. Herdr 状态被清除。

### 场景 D：自定义网页工具

1. 打开示例网页；
2. 绑定到“简历优化”项目；
3. 页面发送 `running`；
4. Herdr 显示运行状态；
5. 页面发送 `done`；
6. Herdr 发通知；
7. 页面发送 `error`；
8. Herdr 发请求关注通知。

### 场景 E：故障恢复

验证：

```text
Herdr 关闭
Edge 扩展断开
Native Host 崩溃
浏览器重启
Herdr 重启
项目 workspace ID变化
绑定配置保留
Quick Action 仍能打开网页
```

故障时应降级，不得崩溃或通知风暴。

---

## 十九、交付物

最终必须生成：

```text
README.md
PLAN.md
docs/environment-audit.md
docs/ARCHITECTURE.md
docs/QUICKSTART_WINDOWS.md
docs/SECURITY.md
docs/KNOWN_LIMITATIONS.md
docs/TEST_REPORT.md
dist/herdr-web-bridge.exe
dist/edge-extension/
installer/install.ps1
installer/uninstall.ps1
examples/custom-tool-integration.html
examples/custom-tool-integration.js
```

并生成压缩包：

```text
release/Herdr_Web_Bridge_Windows_MVP.zip
```

压缩包中包含：

* 本地程序；
* Edge 扩展；
* 安装脚本；
* 卸载脚本；
* 快速开始；
* 安全说明；
* 已知限制；
* 测试报告。

---

## 二十、完成标准

只有以下条件全部满足，才能声明 MVP 完成：

* [ ] 能从 Edge 把当前网页绑定到 Herdr 项目文件夹；
* [ ] 绑定配置可持久化；
* [ ] Herdr 中存在可运行的项目网页快捷入口；
* [ ] 快捷入口优先聚焦已有标签；
* [ ] 没有已有标签时可以打开网页；
* [ ] ChatGPT 可以识别一次完整的运行到完成过程；
* [ ] 自定义网页工具可以稳定上报状态；
* [ ] 完成状态能写入 Herdr workspace metadata；
* [ ] 完成时能通过 Herdr 发通知；
* [ ] 查看网页后可以清除未读状态；
* [ ] 不传输或保存 ChatGPT 正文；
* [ ] 不执行网页提供的任意命令；
* [ ] 安装和卸载可重复执行；
* [ ] 自动测试通过；
* [ ] 人工验收结果写入 TEST_REPORT；
* [ ] 生成最终 ZIP。

---

## 二十一、执行规则

1. 先审计，再计划，再实现；
2. 不要只给设计方案，要实际创建代码和文件；
3. 不要修改 Herdr、Herdr Plus 或用户其他项目的源代码；
4. 不要自动安装、升级或卸载第三方工具；
5. 不要执行破坏性命令；
6. 不要开放公网端口；
7. 不要使用网页正文作为通知内容；
8. 不要为了宣称完成而跳过失败测试；
9. 对所有未验证的 Herdr API，以本机 `herdr api schema --json` 为准；
10. 发现 ChatGPT DOM 无法稳定识别时，进入 `unknown`，不要伪造正确状态；
11. 第一版不要加入 Claude、Gemini、飞书、Notion 等额外 adapter；
12. 第一版不要开发 Electron 桌面端；
13. 第一版不要加入云同步、账号系统或遥测；
14. 每完成一个阶段，更新 `PLAN.md` 和 `TEST_REPORT.md`；
15. 最终回复必须列出：

    * 实际完成内容；
    * 尚未完成内容；
    * 测试通过情况；
    * 安装命令；
    * 生成文件路径；
    * 已知风险；
    * 下一步最小建议。
