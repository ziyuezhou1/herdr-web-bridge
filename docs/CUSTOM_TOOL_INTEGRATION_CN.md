# 自定义网页工具接入

网页只需要向自身 origin 发送状态事件；不要引用 Herdr workspace ID、项目路径或本地命令。

```javascript
window.postMessage({
  source: "herdr-web-bridge",
  version: 1,
  type: "task-status",
  taskId: "resume-2026-001",
  state: "running",
  title: "简历优化",
  message: "正在生成岗位定制简历"
}, window.location.origin);
```

`state` 只允许 `idle`、`running`、`done`、`error`。`taskId` 最长 128 字符，`title` 最长 80 字符，`message` 最长 160 字符。扩展验证 `event.source === window`、事件 origin、绑定 origin、字段白名单和长度；正文不会转发。

在 Edge 打开工具，选择扩展中的“自定义网页工具”，完成绑定并批准该 origin 权限。可直接打开 `examples/custom-tool-integration.html` 所在的 localhost HTTPS/HTTP 开发服务进行测试；普通 `file:` 页面会被拒绝。

