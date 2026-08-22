"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { shouldFallbackStatus } = require("../src/status-queue.js");

test("disconnected running-to-done and error states receive one local fallback", () => {
  const base = { healthy: false, notificationsEnabled: true, previousState: "running", eventId: "event-1" };
  assert.equal(shouldFallbackStatus(Object.assign({}, base, { state: "done_unread" })), true);
  assert.equal(shouldFallbackStatus(Object.assign({}, base, { state: "error" })), true);
  assert.equal(shouldFallbackStatus(Object.assign({}, base, { state: "done_unread", lastFallbackEventId: "event-1" })), false);
});

test("healthy or notification-disabled state never invokes Edge fallback", () => {
  const base = { notificationsEnabled: true, previousState: "running", state: "done_unread", eventId: "event-1" };
  assert.equal(shouldFallbackStatus(Object.assign({}, base, { healthy: true })), false);
  assert.equal(shouldFallbackStatus(Object.assign({}, base, { healthy: false, notificationsEnabled: false })), false);
});
