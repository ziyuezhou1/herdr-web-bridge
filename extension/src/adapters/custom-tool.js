(function initCustomToolAdapter(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function customToolFactory() {
  "use strict";

  const PAGE_SOURCE = "herdr-web-bridge";
  const PAGE_STATES = new Set(["running", "done", "error", "idle"]);

  function validText(value, maximum, required = false) {
    if (typeof value !== "string") return !required && value === undefined;
    return (!required || value.length > 0) && Array.from(value).length <= maximum;
  }

  function validateCustomToolMessage(event, locationLike, boundOrigin, windowLike = globalThis.window) {
    if (!event || !windowLike || event.source !== windowLike) {
      return { ok: false, reason: "invalid_source" };
    }
    if (event.origin !== locationLike.origin || event.origin !== boundOrigin) return { ok: false, reason: "invalid_origin" };
    const data = event.data;
    if (!data || typeof data !== "object" || Array.isArray(data)) return { ok: false, reason: "invalid_data" };
    const allowedKeys = new Set(["source", "version", "type", "taskId", "state", "title", "message"]);
    if (Object.keys(data).some((key) => !allowedKeys.has(key))) return { ok: false, reason: "unexpected_field" };
    if (data.source !== PAGE_SOURCE || data.version !== 1 || data.type !== "task-status") return { ok: false, reason: "invalid_envelope" };
    if (!PAGE_STATES.has(data.state)) return { ok: false, reason: "invalid_state" };
    if (!validText(data.taskId, 128, true) || !validText(data.title, 80) || !validText(data.message, 160)) {
      return { ok: false, reason: "field_too_long" };
    }
    const stateMap = { running: "running", done: "done_unread", error: "error", idle: "idle" };
    return {
      ok: true,
      value: {
        state: stateMap[data.state],
        eventId: `custom-${data.taskId}-${data.state}`,
        pageTitle: data.title || "",
        reason: data.state === "done" ? "custom_tool_reported_done" : `custom_tool_reported_${data.state}`
      }
    };
  }

  return { PAGE_SOURCE, PAGE_STATES, validateCustomToolMessage };
});
