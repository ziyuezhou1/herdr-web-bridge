(function initProtocol(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function protocolFactory() {
  "use strict";

  const PROTOCOL_VERSION = 1;
  const HOST_NAME = "com.herdr_web_bridge";
  const EXTENSION_ID = "pphgcjjepkodhghpncncnmikafkdjdjd";
  const STATES = new Set(["idle", "running", "done_unread", "viewed", "error", "unknown"]);

  function requestId() {
    if (globalThis.crypto && typeof globalThis.crypto.randomUUID === "function") {
      return globalThis.crypto.randomUUID();
    }
    return `req-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  }

  function makeRequest(type, payload = {}) {
    if (!/^[a-z_]{1,64}$/.test(type)) throw new Error("Invalid request type");
    return { version: PROTOCOL_VERSION, id: requestId(), type, payload };
  }

  function assertResponse(message) {
    if (!message || message.version !== PROTOCOL_VERSION || typeof message.type !== "string") {
      throw new Error("Invalid native host response");
    }
    return message;
  }

  function normalizeUrl(raw) {
    const url = new URL(raw);
    url.hash = "";
    url.hostname = url.hostname.toLowerCase();
    if ((url.protocol === "https:" && url.port === "443") || (url.protocol === "http:" && url.port === "80")) {
      url.port = "";
    }
    if (url.pathname.length > 1) url.pathname = url.pathname.replace(/\/+$/, "");
    return url.toString();
  }

  function urlsMatch(left, right) {
    if (left === right) return true;
    try {
      return normalizeUrl(left) === normalizeUrl(right);
    } catch {
      return false;
    }
  }

  function siteAdapter(rawUrl) {
    try {
      const url = new URL(rawUrl);
      if (url.protocol === "https:" && url.hostname === "chatgpt.com") return "chatgpt";
      return "generic";
    } catch {
      return "generic";
    }
  }

  function safeOriginPattern(rawUrl) {
    const url = new URL(rawUrl);
    const local = url.hostname === "localhost" || url.hostname === "127.0.0.1";
    if (url.protocol !== "https:" && !(url.protocol === "http:" && local)) {
      throw new Error("Only HTTPS and explicit localhost HTTP bindings are allowed");
    }
    // Chromium match patterns do not include a port component. The injected
    // script remains inert unless the exact page URL resolves to a binding.
    return `${url.protocol}//${url.hostname}/*`;
  }

  return {
    PROTOCOL_VERSION,
    HOST_NAME,
    EXTENSION_ID,
    STATES,
    makeRequest,
    assertResponse,
    normalizeUrl,
    urlsMatch,
    siteAdapter,
    safeOriginPattern
  };
});
