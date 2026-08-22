"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { NativeClient } = require("../src/native-client.js");

class EventHook {
  constructor() { this.listeners = []; }
  addListener(listener) { this.listeners.push(listener); }
  emit(value) { this.listeners.forEach((listener) => listener(value)); }
}

class FakePort {
  constructor() { this.onMessage = new EventHook(); this.onDisconnect = new EventHook(); this.sent = []; }
  postMessage(message) { this.sent.push(message); }
  disconnect() { this.onDisconnect.emit(); }
}

function setup() {
  const ports = [];
  const timers = [];
  let id = 0;
  const runtime = {
    lastError: null,
    connectNative() { const port = new FakePort(); ports.push(port); return port; }
  };
  const client = new NativeClient(runtime, {
    makeRequest: (type, payload) => ({ version: 1, id: `id-${++id}`, type, payload }),
    assertResponse: (message) => message,
    setTimer: (callback) => { timers.push(callback); return timers.length; },
    clearTimer: () => {},
    reconnectDelays: [1]
  });
  return { client, runtime, ports, timers };
}

test("native client resolves responses and rejects all pending calls on disconnect", async () => {
  const { client, ports } = setup();
  client.start();
  const success = client.call("ping", {});
  const request = ports[0].sent[0];
  ports[0].onMessage.emit({ version: 1, id: request.id, type: "ping", ok: true, result: { status: "ok" } });
  assert.deepEqual(await success, { status: "ok" });

  const pending = client.call("list_bindings", {});
  ports[0].onDisconnect.emit();
  await assert.rejects(pending, /disconnected/i);
});

test("native client schedules service-worker reconnection", () => {
  const { client, ports, timers } = setup();
  client.start();
  assert.equal(ports.length, 1);
  ports[0].onDisconnect.emit();
  assert.equal(timers.length, 1);
  timers[0]();
  assert.equal(ports.length, 2);
});

test("default timer wrappers preserve the browser global receiver", () => {
  const ports = [];
  const timers = [];
  const cleared = [];
  const runtime = {
    lastError: null,
    connectNative() { const port = new FakePort(); ports.push(port); return port; }
  };
  const timerHost = {
    setTimeout(callback) {
      assert.equal(this, timerHost);
      timers.push(callback);
      return timers.length;
    },
    clearTimeout(timer) {
      assert.equal(this, timerHost);
      cleared.push(timer);
    }
  };
  const client = new NativeClient(runtime, {
    makeRequest: (type, payload) => ({ version: 1, id: "timer-test", type, payload }),
    assertResponse: (message) => message,
    timerHost,
    reconnectDelays: [1]
  });

  client.start();
  ports[0].onDisconnect.emit();
  assert.equal(timers.length, 1);
  client.stop();
  assert.deepEqual(cleared, [1]);
});
