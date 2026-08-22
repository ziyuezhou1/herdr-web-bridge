(function initStatusQueue(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function statusQueueFactory() {
  "use strict";

  function shouldFallbackStatus({ healthy, notificationsEnabled, previousState, state, eventId, lastFallbackEventId }) {
    if (healthy || !notificationsEnabled || !eventId || lastFallbackEventId === eventId) return false;
    return (previousState === "running" && state === "done_unread") || state === "error";
  }

  return { shouldFallbackStatus };
});

