"use strict";

const popupModel = globalThis.HerdrPopupModel;

const elements = {
  connection: document.getElementById("connection"),
  title: document.getElementById("page-title"),
  url: document.getElementById("page-url"),
  adapter: document.getElementById("adapter"),
  workspace: document.getElementById("workspace"),
  workspacePath: document.getElementById("workspace-path"),
  notifications: document.getElementById("notifications"),
  bind: document.getElementById("bind"),
  unbind: document.getElementById("unbind"),
  openWorkspace: document.getElementById("open-workspace"),
  bindingState: document.getElementById("binding-state"),
  adapterHealth: document.getElementById("adapter-health"),
  quickAction: document.getElementById("quick-action"),
  message: document.getElementById("message")
};

let context = null;
let busy = false;

load().catch(showError);
elements.workspace.addEventListener("change", updateWorkspacePath);
elements.bind.addEventListener("click", bindPage);
elements.unbind.addEventListener("click", unbindPage);
elements.openWorkspace.addEventListener("click", openWorkspace);
document.querySelectorAll("[data-state]").forEach((button) => button.addEventListener("click", () => testState(button.dataset.state)));

async function load() {
  setBusy(true);
  context = await send("popup_context");
  elements.title.textContent = context.tab.title || "无标题网页";
  elements.title.title = context.tab.title || "";
  elements.url.textContent = context.tab.url;
  elements.url.title = context.tab.url;
  elements.adapter.value = context.binding ? context.binding.adapter : context.detectedAdapter;
  const workspaceState = renderWorkspaces(context.workspaces);
  renderBinding(context.binding);
  elements.connection.textContent = `本地桥接已连接 · ${workspaceState.bindableCount} 个可绑定工作区`;
  if (workspaceState.warning) showWarning(workspaceState.warning);
  setBusy(false);
}

function renderWorkspaces(workspaces) {
  const items = Array.isArray(workspaces) ? workspaces : [];
  const summary = popupModel.workspaceSummary(items);
  elements.workspace.replaceChildren(new Option(summary.placeholder, ""));
  items.forEach((workspace, index) => {
    const option = new Option(popupModel.workspaceOptionLabel(workspace), String(index));
    elements.workspace.add(option);
  });
  if (context.binding) {
    const index = items.findIndex((workspace) => workspace.projectPath && workspace.projectPath.toLowerCase() === context.binding.projectPath.toLowerCase());
    if (index >= 0) elements.workspace.value = String(index);
  }
  updateWorkspacePath();
  return summary;
}

function renderBinding(binding) {
  const isBound = Boolean(binding);
  elements.bindingState.textContent = isBound ? `已绑定 · ${binding.lastState}` : "未绑定";
  elements.bindingState.classList.toggle("bound", isBound);
  const health = !isBound ? "未开始监控" : (binding.lastState === "unknown" ? "unknown · selector 信号不足" : `正常 · ${binding.lastState}`);
  elements.adapterHealth.textContent = health;
  elements.adapterHealth.classList.toggle("unknown", isBound && binding.lastState === "unknown");
  elements.unbind.disabled = !isBound;
  elements.openWorkspace.disabled = !isBound;
  elements.adapter.disabled = isBound;
  document.querySelectorAll("[data-state]").forEach((button) => { button.disabled = !isBound; });
  if (isBound) elements.notifications.checked = binding.notificationsEnabled;
  updateActionState();
}

async function bindPage() {
  const workspace = selectedWorkspace();
  if (!workspace) return showError(new Error("请选择可绑定的 Herdr 工作区"));
  if (!workspace.pathAvailable || !workspace.projectPath) {
    return showError(new Error("所选工作区没有可安全解析的项目路径，暂时无法绑定"));
  }
  setBusy(true);
  try {
    const result = await send("bind_page", {
      projectPath: workspace.projectPath,
      projectLabel: workspace.label,
      url: context.tab.url,
      pageTitle: context.tab.title || "网页",
      adapter: elements.adapter.value,
      notificationsEnabled: elements.notifications.checked
    });
    context.binding = result.binding;
    renderBinding(context.binding);
    elements.quickAction.textContent = result.quickActionGenerated ? "已生成 Herdr Plus 快捷入口。" : "绑定已保存；尚未生成 Herdr 快捷入口。";
    showMessage("网页已绑定到项目。");
  } catch (error) {
    showError(error);
  } finally {
    setBusy(false);
  }
}

async function unbindPage() {
  if (!context.binding) return;
  setBusy(true);
  try {
    const result = await send("unbind_page", { bindingId: context.binding.id });
    context.binding = null;
    renderBinding(null);
    elements.quickAction.textContent = result.warning || "";
    showMessage("绑定已解除。");
  } catch (error) {
    showError(error);
  } finally {
    setBusy(false);
  }
}

async function openWorkspace() {
  if (!context.binding) return;
  try {
    await send("focus_workspace", { bindingId: context.binding.id });
    showMessage("已请求 Herdr 聚焦对应工作区。");
  } catch (error) {
    showError(error);
  }
}

async function testState(state) {
  if (!context.binding) return;
  try {
    await send("manual_status", {
      bindingId: context.binding.id,
      state,
      pageTitle: context.tab.title || "网页",
      url: context.tab.url
    });
    context.binding.lastState = state;
    renderBinding(context.binding);
    showMessage(`已发送状态：${state}`);
  } catch (error) {
    showError(error);
  }
}

function selectedWorkspace() {
  if (!context || elements.workspace.value === "") return null;
  return context.workspaces[Number(elements.workspace.value)] || null;
}

function updateWorkspacePath() {
  const workspace = selectedWorkspace();
  elements.workspacePath.textContent = popupModel.workspacePathHint(workspace);
  updateActionState();
}

function setBusy(value) {
  busy = value;
  if (busy) elements.connection.textContent = "正在连接本地桥接…";
  updateActionState();
}

function updateActionState() {
  const isBound = Boolean(context && context.binding);
  const workspace = selectedWorkspace();
  const controls = popupModel.bindingControls({ busy, isBound, workspace });
  elements.workspace.disabled = controls.workspaceDisabled;
  elements.bind.disabled = controls.bindDisabled;
}

function showMessage(message) {
  elements.message.classList.remove("error", "warning");
  elements.message.textContent = message;
}

function showWarning(message) {
  elements.message.classList.remove("error");
  elements.message.classList.add("warning");
  elements.message.textContent = message;
}

function showError(error) {
  elements.connection.textContent = "本地桥接不可用或请求失败";
  elements.message.classList.remove("warning");
  elements.message.classList.add("error");
  elements.message.textContent = popupModel.bridgeErrorMessage(error);
  setBusy(false);
}

function send(type, payload = {}) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage({ type, payload }, (response) => {
      const lastError = chrome.runtime.lastError;
      if (lastError) return reject(new Error(lastError.message));
      if (!response || !response.ok) {
        const error = new Error(response && response.error ? response.error.message : "扩展请求失败");
        if (response && response.error) error.code = response.error.code;
        return reject(error);
      }
      resolve(response.result);
    });
  });
}
