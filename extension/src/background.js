"use strict";

importScripts("protocol.js", "native-client.js", "tab-matcher.js", "fallback-notification.js", "status-queue.js");

const HB = globalThis.HerdrBridge;
const nativeClient = new HB.NativeClient(chrome.runtime, {
  hostName: HB.HOST_NAME,
  onConnected: () => {
    nativeClient.call("hello", { extensionId: chrome.runtime.id })
      .then(() => flushPendingStatuses())
      .catch(() => {});
  }
});

nativeClient.onEvent((message) => {
  if (message.type === "open_binding" && message.ok && message.result) {
    focusOrOpenBinding(message.result).catch(() => {});
  }
});
nativeClient.start();

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleRuntimeMessage(message, sender)
    .then((result) => sendResponse({ ok: true, result }))
    .catch((error) => sendResponse({ ok: false, error: { code: error.code || "extension_error", message: error.message } }));
  return true;
});

async function handleRuntimeMessage(message, sender) {
  if (!message || typeof message.type !== "string") throw new Error("Invalid extension message");
  switch (message.type) {
    case "popup_context":
      return popupContext();
    case "bind_page":
      return bindPage(message.payload);
    case "unbind_page":
      return unbindPage(message.payload);
    case "manual_status":
      return reportStatus(Object.assign({}, message.payload, {
        eventId: message.payload.eventId || `manual-${message.payload.bindingId}-${Date.now()}`,
        reason: "manual_popup_test"
      }));
    case "focus_workspace":
      return nativeClient.call("focus_workspace", { bindingId: message.payload.bindingId });
    case "binding_for_url":
      return nativeClient.call("binding_for_url", { url: message.payload.url });
    case "content_status":
      if (!sender.tab || !sender.tab.url || !HB.urlsMatch(sender.tab.url, message.payload.url)) {
        throw new Error("Content status sender URL mismatch");
      }
      return reportStatus(message.payload);
    case "connection_status":
      return nativeClient.call("ping", {});
    default:
      throw new Error("Extension message type is not allowed");
  }
}

async function popupContext() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab || !tab.url) throw new Error("No active web page");
  const [bindingResult, workspaceResult] = await Promise.all([
    nativeClient.call("binding_for_url", { url: tab.url }),
    nativeClient.call("list_workspaces", {})
  ]);
  return {
    tab: { id: tab.id, title: tab.title || "", url: tab.url },
    detectedAdapter: HB.siteAdapter(tab.url),
    binding: bindingResult.binding || null,
    workspaces: workspaceResult.workspaces || []
  };
}

async function bindPage(payload) {
  if (!payload || !payload.url) throw new Error("Binding payload is incomplete");
  if (payload.adapter !== "chatgpt") await ensureOriginAccess(payload.url);
  const result = await nativeClient.call("bind_page", payload);
  if (result.binding) {
    await chrome.storage.local.set({
      [`binding:${result.binding.id}`]: {
        url: result.binding.url,
        adapter: result.binding.adapter,
        projectLabel: result.binding.projectLabel,
        notificationsEnabled: result.binding.notificationsEnabled
      }
    });
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (tab && tab.id) {
      if (payload.adapter !== "chatgpt") {
        await injectContent(tab.id);
      } else {
        await chrome.tabs.sendMessage(tab.id, { type: "herdr_binding_refresh" }).catch(() => {});
      }
    }
  }
  return result;
}

async function unbindPage(payload) {
  const result = await nativeClient.call("unbind_page", { bindingId: payload.bindingId });
  await chrome.storage.local.remove([
    `binding:${payload.bindingId}`,
    `last-state:${payload.bindingId}`,
    `fallback-event:${payload.bindingId}`,
    `pending-status:${payload.bindingId}`
  ]);
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tab && tab.id) await chrome.tabs.sendMessage(tab.id, { type: "herdr_binding_refresh" }).catch(() => {});
  return result;
}

async function ensureOriginAccess(rawUrl) {
  const pattern = HB.safeOriginPattern(rawUrl);
  const granted = await chrome.permissions.request({ origins: [pattern] });
  if (!granted) throw new Error("Origin permission was not granted");
  const id = `hwb-${originHash(new URL(rawUrl).origin)}`;
  const existing = await chrome.scripting.getRegisteredContentScripts({ ids: [id] });
  if (existing.length === 0) {
    await chrome.scripting.registerContentScripts([{
      id,
      matches: [pattern],
      js: ["src/protocol.js", "src/state-machine.js", "src/adapters/chatgpt.js", "src/adapters/custom-tool.js", "src/adapters/generic.js", "src/content.js"],
      runAt: "document_idle",
      persistAcrossSessions: true
    }]);
  }
}

async function injectContent(tabId) {
  await chrome.scripting.executeScript({
    target: { tabId },
    files: ["src/protocol.js", "src/state-machine.js", "src/adapters/chatgpt.js", "src/adapters/custom-tool.js", "src/adapters/generic.js", "src/content.js"]
  });
}

async function reportStatus(payload) {
  const stateKey = `last-state:${payload.bindingId}`;
  const bindingKey = `binding:${payload.bindingId}`;
  const fallbackKey = `fallback-event:${payload.bindingId}`;
  const pendingKey = `pending-status:${payload.bindingId}`;
  const local = await chrome.storage.local.get([stateKey, bindingKey, fallbackKey]);
  const previousState = local[stateKey] || "idle";
  try {
    const result = await nativeClient.call("report_status", payload);
    if (result.fallbackNotification && result.notification) {
      await HB.showFallbackNotification(chrome.notifications, result.notification, chrome.runtime.getURL("icons/bridge.svg"));
    }
    await chrome.storage.local.set({ [stateKey]: payload.state });
    await chrome.storage.local.remove(pendingKey);
    return result;
  } catch (error) {
    const bindingInfo = local[bindingKey] || {};
    const shouldFallback = HB.shouldFallbackStatus({
      healthy: nativeClient.healthy,
      notificationsEnabled: bindingInfo.notificationsEnabled,
      previousState,
      state: payload.state,
      eventId: payload.eventId,
      lastFallbackEventId: local[fallbackKey]
    });
    const queuedPayload = Object.assign({}, payload);
    let fallbackNotification = null;
    if (shouldFallback) {
      fallbackNotification = payload.state === "error"
        ? { title: "网页任务出错", message: `${bindingInfo.projectLabel || "绑定项目"}：请返回网页检查` }
        : { title: "网页任务已完成", message: `${bindingInfo.projectLabel || "绑定项目"}：网页结果等待查看` };
      queuedPayload.notificationHandled = true;
    }
    const persisted = {
      [stateKey]: payload.state,
      [pendingKey]: { payload: queuedPayload, queuedAt: new Date().toISOString() }
    };
    if (shouldFallback) persisted[fallbackKey] = payload.eventId;
    await chrome.storage.local.set(persisted);
    if (fallbackNotification) {
      await HB.showFallbackNotification(chrome.notifications, fallbackNotification, chrome.runtime.getURL("icons/bridge.svg")).catch(() => {});
    }
    throw error;
  }
}

async function flushPendingStatuses() {
  const all = await chrome.storage.local.get(null);
  const entries = Object.entries(all).filter(([key]) => key.startsWith("pending-status:"));
  for (const [, pending] of entries) {
    if (!pending || !pending.payload) continue;
    try {
      await reportStatus(pending.payload);
    } catch {
      return;
    }
  }
}

async function focusOrOpenBinding(event) {
  const key = `tab:${event.bindingId}`;
  const cached = await chrome.storage.session.get(key);
  const tabs = await chrome.tabs.query({});
  const tab = HB.findBoundTab(tabs, { url: event.url, urlMatch: event.urlMatch }, cached[key]);
  if (tab) {
    await chrome.tabs.update(tab.id, { active: true });
    await chrome.windows.update(tab.windowId, { focused: true });
    await chrome.storage.session.set({ [key]: tab.id });
    return { status: "focused", tabId: tab.id };
  }
  const created = await chrome.tabs.create({ url: event.url, active: true });
  await chrome.storage.session.set({ [key]: created.id });
  return { status: "opened", tabId: created.id };
}

function originHash(origin) {
  let hash = 2166136261;
  for (let index = 0; index < origin.length; index += 1) {
    hash ^= origin.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(16);
}
