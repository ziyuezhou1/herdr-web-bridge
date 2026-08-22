"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { collectChatGPTSignals } = require("../src/adapters/chatgpt.js");

function fixtureDocument(name, visible = false, focused = false) {
  const html = fs.readFileSync(path.join(__dirname, "fixtures", name), "utf8");
  const matches = (selector) => {
    if (selector === "main") return /<main[\s>]/.test(html);
    if (selector === "[role='main']") return /role=["']main["']/.test(html);
    if (selector.includes("stop-button") || selector.includes("Stop") || selector.includes("停止")) return /stop-button|Stop generating|停止/.test(html);
    if (selector.includes("retry") || selector.includes("Retry") || selector.includes("重试")) return /retry-response|Retry|重试/.test(html);
    if (selector.includes("streaming")) return /data-is-streaming=["']true|conversation-turn-streaming/.test(html);
    if (selector.includes("conversation-turn-")) return /conversation-turn-/.test(html);
    if (selector.includes("prompt-textarea") || selector.includes("composer") || selector === "textarea" || selector.includes("contenteditable")) return /prompt-textarea|data-testid=["']composer|<textarea|contenteditable/.test(html);
    return false;
  };
  return {
    visibilityState: visible ? "visible" : "hidden",
    hasFocus: () => focused,
    querySelector: (selector) => matches(selector) ? {} : null,
    querySelectorAll: (selector) => {
      if (selector.includes("assistant")) return new Array((html.match(/data-message-author-role=["']assistant["']/g) || []).length).fill({});
      return [];
    }
  };
}

test("DOM fixtures expose independent running, done, error, and unknown signals", () => {
  const history = collectChatGPTSignals(fixtureDocument("chatgpt-history.html"), {});
  const fresh = collectChatGPTSignals(fixtureDocument("chatgpt-new.html"), {});
  const running = collectChatGPTSignals(fixtureDocument("chatgpt-running.html"), {});
  const done = collectChatGPTSignals(fixtureDocument("chatgpt-done.html"), {});
  const error = collectChatGPTSignals(fixtureDocument("chatgpt-error.html"), {});
  const unknown = collectChatGPTSignals(fixtureDocument("chatgpt-unknown.html"), {});
  assert.equal(history.assistantCount, 2);
  assert.equal(history.running, false);
  assert.equal(fresh.confident, true);
  assert.equal(fresh.assistantCount, 0);
  assert.equal(running.running, true);
  assert.equal(done.assistantCount, 3);
  assert.equal(error.error, true);
  assert.equal(unknown.confident, false);
});
