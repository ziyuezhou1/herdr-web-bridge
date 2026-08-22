(function initNativeClient(root, factory) {
  const api = factory(root.HerdrBridge || {});
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function nativeClientFactory(shared) {
  "use strict";

  class NativeClient {
    constructor(runtime, options = {}) {
      this.runtime = runtime;
      this.hostName = options.hostName || shared.HOST_NAME || "com.herdr_web_bridge";
      this.makeRequest = options.makeRequest || shared.makeRequest;
      this.assertResponse = options.assertResponse || shared.assertResponse || ((value) => value);
      const timerHost = options.timerHost || globalThis;
      this.setTimer = options.setTimer || ((callback, delay) => timerHost.setTimeout(callback, delay));
      this.clearTimer = options.clearTimer || ((timer) => timerHost.clearTimeout(timer));
      this.timeoutMs = options.timeoutMs || 5000;
      this.reconnectDelays = options.reconnectDelays || [250, 1000, 3000, 10000];
      this.pending = new Map();
      this.eventHandlers = new Set();
      this.port = null;
      this.healthy = false;
      this.reconnectTimer = null;
      this.reconnectAttempt = 0;
      this.stopped = false;
      this.onConnected = options.onConnected || (() => {});
      this.onDisconnected = options.onDisconnected || (() => {});
    }

    start() {
      this.stopped = false;
      this.connect();
    }

    stop() {
      this.stopped = true;
      if (this.reconnectTimer) this.clearTimer(this.reconnectTimer);
      if (this.port) this.port.disconnect();
      this.rejectPending(new Error("Native client stopped"));
      this.port = null;
      this.healthy = false;
    }

    connect() {
      if (this.stopped || this.port) return;
      try {
        const port = this.runtime.connectNative(this.hostName);
        this.port = port;
        port.onMessage.addListener((message) => this.handleMessage(message));
        port.onDisconnect.addListener(() => this.handleDisconnect(port));
        this.onConnected();
      } catch (error) {
        this.scheduleReconnect(error);
      }
    }

    call(type, payload = {}) {
      if (!this.port) this.connect();
      if (!this.port) return Promise.reject(new Error("Native host is not connected"));
      const request = this.makeRequest(type, payload);
      return new Promise((resolve, reject) => {
        const timer = this.setTimer(() => {
          this.pending.delete(request.id);
          reject(new Error(`Native request timed out: ${type}`));
        }, this.timeoutMs);
        this.pending.set(request.id, { resolve, reject, timer });
        try {
          this.port.postMessage(request);
        } catch (error) {
          this.pending.delete(request.id);
          this.clearTimer(timer);
          this.healthy = false;
          reject(error);
        }
      });
    }

    onEvent(handler) {
      this.eventHandlers.add(handler);
      return () => this.eventHandlers.delete(handler);
    }

    handleMessage(rawMessage) {
      let message;
      try {
        message = this.assertResponse(rawMessage);
      } catch {
        return;
      }
      this.reconnectAttempt = 0;
      this.healthy = true;
      const pending = this.pending.get(message.id);
      if (pending) {
        this.pending.delete(message.id);
        this.clearTimer(pending.timer);
        if (message.ok) pending.resolve(message.result);
        else {
          const error = new Error(message.error && message.error.message ? message.error.message : "Native host request failed");
          error.code = message.error && message.error.code ? message.error.code : "native_error";
          pending.reject(error);
        }
        return;
      }
      for (const handler of this.eventHandlers) handler(message);
    }

    handleDisconnect(port) {
      if (this.port !== port) return;
      // Accessing lastError prevents an expected disconnect warning in Chromium.
      const lastError = this.runtime.lastError;
      this.port = null;
      this.healthy = false;
      this.rejectPending(new Error(lastError && lastError.message ? lastError.message : "Native host disconnected"));
      this.onDisconnected(lastError || null);
      this.scheduleReconnect(lastError);
    }

    rejectPending(error) {
      for (const pending of this.pending.values()) {
        this.clearTimer(pending.timer);
        pending.reject(error);
      }
      this.pending.clear();
    }

    scheduleReconnect() {
      if (this.stopped || this.reconnectTimer) return;
      const index = Math.min(this.reconnectAttempt, this.reconnectDelays.length - 1);
      const delay = this.reconnectDelays[index];
      this.reconnectAttempt += 1;
      this.reconnectTimer = this.setTimer(() => {
        this.reconnectTimer = null;
        this.connect();
      }, delay);
    }
  }

  return { NativeClient };
});
