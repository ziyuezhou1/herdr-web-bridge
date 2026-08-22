"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { validateCustomToolMessage } = require("../src/adapters/custom-tool.js");

const windowLike = {};
const locationLike = { origin: "https://tool.example" };

function event(data, overrides = {}) {
  return Object.assign({ source: windowLike, origin: locationLike.origin, data }, overrides);
}

function valid(overrides = {}) {
  return Object.assign({ source: "herdr-web-bridge", version: 1, type: "task-status", taskId: "resume-1", state: "running", title: "简历优化", message: "正在生成" }, overrides);
}

test("custom tool accepts the documented same-origin schema", () => {
  const result = validateCustomToolMessage(event(valid()), locationLike, locationLike.origin, windowLike);
  assert.equal(result.ok, true);
  assert.equal(result.value.state, "running");
});

test("custom tool rejects foreign origins and sources", () => {
  assert.equal(validateCustomToolMessage(event(valid(), { origin: "https://evil.example" }), locationLike, locationLike.origin, windowLike).ok, false);
  assert.equal(validateCustomToolMessage(event(valid(), { source: {} }), locationLike, locationLike.origin, windowLike).ok, false);
});

test("custom tool rejects invalid state, long fields, and command-shaped extras", () => {
  assert.equal(validateCustomToolMessage(event(valid({ state: "execute" })), locationLike, locationLike.origin, windowLike).ok, false);
  assert.equal(validateCustomToolMessage(event(valid({ taskId: "x".repeat(129) })), locationLike, locationLike.origin, windowLike).ok, false);
  assert.equal(validateCustomToolMessage(event(Object.assign(valid(), { command: "powershell" })), locationLike, locationLike.origin, windowLike).ok, false);
});

