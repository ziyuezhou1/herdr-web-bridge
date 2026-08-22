(function initFallbackNotification(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function fallbackNotificationFactory() {
  "use strict";

  function fallbackOptions(notification, iconUrl) {
    return {
      type: "basic",
      iconUrl,
      title: String(notification.title || "Herdr Web Bridge").slice(0, 80),
      message: String(notification.message || "网页任务状态已更新").slice(0, 240),
      priority: 1
    };
  }

  async function showFallbackNotification(notificationsApi, notification, iconUrl) {
    return notificationsApi.create(`herdr-web-bridge-${Date.now()}`, fallbackOptions(notification, iconUrl));
  }

  return { fallbackOptions, showFallbackNotification };
});

