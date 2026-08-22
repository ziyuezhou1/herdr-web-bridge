(function startContentScript() {
  "use strict";
  if (globalThis.__herdrWebBridgeContentLoaded) return;
  globalThis.__herdrWebBridgeContentLoaded = true;

  const HB = globalThis.HerdrBridge;
  let binding = null;
  let adapter = null;
  let activeSince = document.visibilityState === "visible" && document.hasFocus() ? Date.now() : 0;
  let customCompletionTimer = null;

  document.addEventListener("visibilitychange", updateActiveSince);
  window.addEventListener("focus", updateActiveSince);
  window.addEventListener("blur", updateActiveSince);

  chrome.runtime.onMessage.addListener((message) => {
    if (message && message.type === "herdr_binding_refresh") loadBinding();
  });

  loadBinding();

  function loadBinding() {
    if (adapter && typeof adapter.stop === "function") adapter.stop();
    if (customCompletionTimer) window.clearTimeout(customCompletionTimer);
    customCompletionTimer = null;
    adapter = null;
    window.removeEventListener("message", onCustomMessage);
    binding = null;
    return send({ type: "binding_for_url", payload: { url: location.href } })
    .then((result) => {
      binding = result.binding;
      if (!binding) return;
      if (binding.adapter === "chatgpt" && location.hostname === "chatgpt.com") {
        adapter = new HB.ChatGPTAdapter(document, window, reportAdapterState);
        adapter.start();
      } else if (binding.adapter === "custom-tool") {
        window.addEventListener("message", onCustomMessage);
      }
    })
    .catch(() => {});
  }

  function reportAdapterState(event) {
    if (!binding) return;
    if (!HB.urlsMatch(location.href, binding.url)) {
      loadBinding();
      return;
    }
    send({
      type: "content_status",
      payload: {
        bindingId: binding.id,
        state: event.state,
        eventId: event.eventId || "",
        pageTitle: document.title.slice(0, 160),
        url: location.href,
        reason: event.reason || "adapter_state"
      }
    }).catch(() => {});
  }

  function onCustomMessage(event) {
    if (!binding || binding.adapter !== "custom-tool") return;
    const validation = HB.validateCustomToolMessage(event, location, new URL(binding.url).origin, window);
    if (!validation.ok) return;
    if (customCompletionTimer) window.clearTimeout(customCompletionTimer);
    customCompletionTimer = null;
    if (validation.value.state === "done_unread" && activeSince) {
      const remaining = Math.max(0, 1500 - (Date.now() - activeSince));
      if (remaining > 0) {
        customCompletionTimer = window.setTimeout(() => {
          customCompletionTimer = null;
          if (activeSince && Date.now() - activeSince >= 1500) {
            validation.value.state = "viewed";
            validation.value.reason = "custom_tool_completed_while_viewed";
          }
          reportAdapterState(validation.value);
        }, remaining);
        return;
      }
      validation.value.state = "viewed";
      validation.value.reason = "custom_tool_completed_while_viewed";
    }
    reportAdapterState(validation.value);
  }

  function updateActiveSince() {
    const active = document.visibilityState === "visible" && document.hasFocus();
    if (active && !activeSince) activeSince = Date.now();
    if (!active) activeSince = 0;
  }

  function send(message) {
    return new Promise((resolve, reject) => {
      chrome.runtime.sendMessage(message, (response) => {
        const lastError = chrome.runtime.lastError;
        if (lastError) return reject(new Error(lastError.message));
        if (!response || !response.ok) {
          const error = new Error(response && response.error ? response.error.message : "Bridge message failed");
          if (response && response.error) error.code = response.error.code;
          return reject(error);
        }
        resolve(response.result);
      });
    });
  }
})();
