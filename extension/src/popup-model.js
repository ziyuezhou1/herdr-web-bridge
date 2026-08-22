(function initPopupModel(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrPopupModel = api;
})(typeof globalThis !== "undefined" ? globalThis : this, function popupModelFactory() {
  "use strict";

  function workspaceSummary(workspaces) {
    const items = Array.isArray(workspaces) ? workspaces : [];
    const bindableCount = items.filter((workspace) => workspace && workspace.pathAvailable && workspace.projectPath).length;
    if (items.length === 0) {
      return {
        bindableCount,
        placeholder: "未读取到 Herdr 工作区",
        warning: "本地桥已连接，但 Herdr 没有返回工作区。请在普通 PowerShell 中运行 list-workspaces 诊断。"
      };
    }
    if (bindableCount === 0) {
      return {
        bindableCount,
        placeholder: "工作区缺少可绑定的项目路径",
        warning: "Herdr 返回了工作区，但未能从 worktree.checkout_path 或唯一的 pane cwd 解析项目路径。"
      };
    }
    return {
      bindableCount,
      placeholder: "选择可解析项目路径的工作区",
      warning: ""
    };
  }

  function workspaceOptionLabel(workspace) {
    const item = workspace || {};
    if (!item.pathAvailable || !item.projectPath) return `${item.label || "未命名工作区"} — 路径不可用`;
    const source = item.pathSource === "pane_cwd" ? "pane cwd" : "worktree";
    return `${item.label || "未命名工作区"} — ${item.projectPath} [${source}]`;
  }

  function workspacePathHint(workspace) {
    if (!workspace) return "路径来源优先级：worktree.checkout_path，其次为唯一的 Herdr pane cwd。";
    if (workspace.pathAvailable && workspace.projectPath) {
      const source = workspace.pathSource === "pane_cwd" ? "Herdr pane cwd（普通工作区）" : "Herdr worktree.checkout_path";
      return `${workspace.projectPath} · 来源：${source}`;
    }
    switch (workspace.pathReason) {
      case "ambiguous_pane_cwd":
        return "该工作区的多个窗格位于不同目录，无法安全确定项目路径。请让窗格使用同一项目目录后重试。";
      case "pane_list_unavailable":
        return "无法读取 Herdr pane list，因此不能解析普通工作区路径。";
      case "invalid_worktree_path":
        return "Herdr 返回的 worktree.checkout_path 不是有效的绝对 Windows 路径。";
      default:
        return "该工作区既没有 worktree.checkout_path，也没有可用的 pane cwd。";
    }
  }

  function bridgeErrorMessage(error) {
    const message = error && error.message ? error.message : String(error || "扩展请求失败");
    const code = error && error.code ? error.code : "";
    if (/native messaging host.*not found|specified native messaging host|native host (is )?not connected|找不到.*(?:本机|原生).*消息|本地桥.*(?:未注册|未连接)/i.test(message)) {
      return "Native Messaging Host 未注册或尚未连接。请在普通 PowerShell 中重新运行 installer\\install.ps1，然后在 edge://extensions 点击重新加载。";
    }
    if (code === "herdr_unavailable" || /herdr command failed|permissiondenied|拒绝访问/i.test(message)) {
      return "本地桥已连接，但 Herdr CLI 无法读取工作区。请确认 Herdr 正在运行，并用同一 Windows 用户执行 herdr workspace list。";
    }
    return message;
  }

  function bindingControls({ busy, isBound, workspace }) {
    const canBindSelection = Boolean(workspace && workspace.pathAvailable && workspace.projectPath);
    return {
      workspaceDisabled: Boolean(busy || isBound),
      bindDisabled: Boolean(busy || isBound || !canBindSelection)
    };
  }

  return { workspaceSummary, workspaceOptionLabel, workspacePathHint, bridgeErrorMessage, bindingControls };
});
