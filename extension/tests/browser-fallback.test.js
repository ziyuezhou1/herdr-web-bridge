"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

global.HerdrBridge = require("../src/protocol.js");
const { findBoundTab } = require("../src/tab-matcher.js");
const { fallbackOptions } = require("../src/fallback-notification.js");

test("tab matching prefers cached exact binding without title matching", () => {
  const tabs = [
    { id: 1, title: "same", url: "https://example.com/other" },
    { id: 2, title: "unrelated title", url: "https://example.com/task#section" },
    { id: 3, title: "same", url: "https://example.com/task" }
  ];
  assert.equal(findBoundTab(tabs, { url: "https://example.com/task" }, 3).id, 3);
  assert.equal(findBoundTab(tabs, { url: "https://example.com/task" }, 99).id, 3);
});

test("fallback notification uses only bounded status text", () => {
  const options = fallbackOptions({ title: "T".repeat(100), message: "M".repeat(300) }, "icon.svg");
  assert.equal(options.title.length, 80);
  assert.equal(options.message.length, 240);
  assert.equal(options.iconUrl, "icon.svg");
});

test("optional origin patterns allow HTTPS or localhost only and omit ports", () => {
  assert.equal(global.HerdrBridge.safeOriginPattern("https://tool.example:8443/run"), "https://tool.example/*");
  assert.equal(global.HerdrBridge.safeOriginPattern("http://localhost:3000/run"), "http://localhost/*");
  assert.throws(() => global.HerdrBridge.safeOriginPattern("http://tool.example/run"), /HTTPS/);
});
