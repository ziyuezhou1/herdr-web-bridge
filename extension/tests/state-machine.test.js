"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { TaskStateMachine } = require("../src/state-machine.js");

function signals(overrides = {}) {
  return Object.assign({ running: false, error: false, assistantCount: 2, visible: false, focused: false, confident: true }, overrides);
}

test("ChatGPT initial history establishes idle without completion", () => {
  let now = 1000;
  const events = [];
  const machine = new TaskStateMachine({ now: () => now, onState: (event) => events.push(event) });
  machine.observe(signals({ assistantCount: 5 }));
  now += 5000;
  machine.observe(signals({ assistantCount: 5 }));
  assert.deepEqual(events.map((event) => event.state), ["idle"]);
});

test("running becomes done_unread only after new output and 800 ms stability", () => {
  let now = 1000;
  const events = [];
  const machine = new TaskStateMachine({ now: () => now, onState: (event) => events.push(event) });
  machine.observe(signals());
  now = 1100;
  machine.observe(signals({ running: true }));
  now = 1500;
  machine.observe(signals({ assistantCount: 3 }));
  assert.equal(machine.nextCheckDelay(signals({ assistantCount: 3 })), 800);
  now = 2299;
  machine.observe(signals({ assistantCount: 3 }));
  assert.equal(events.at(-1).state, "running");
  now = 2300;
  machine.observe(signals({ assistantCount: 3 }));
  assert.deepEqual(events.map((event) => event.state), ["idle", "running", "done_unread"]);
  machine.observe(signals({ assistantCount: 3 }));
  assert.equal(events.filter((event) => event.state === "done_unread").length, 1);
});

test("visible and focused page clears unread after 1500 ms", () => {
  let now = 1000;
  const events = [];
  const machine = new TaskStateMachine({ now: () => now, onState: (event) => events.push(event) });
  machine.observe(signals());
  now = 1100;
  machine.observe(signals({ running: true }));
  now = 2000;
  machine.observe(signals({ assistantCount: 3 }));
  now = 2800;
  machine.observe(signals({ assistantCount: 3 }));
  now = 3000;
  machine.observe(signals({ assistantCount: 3, visible: true, focused: true }));
  now = 4499;
  machine.observe(signals({ assistantCount: 3, visible: true, focused: true }));
  assert.equal(events.at(-1).state, "done_unread");
  now = 4500;
  machine.observe(signals({ assistantCount: 3, visible: true, focused: true }));
  assert.equal(events.at(-1).state, "viewed");
});

test("completion watched continuously becomes viewed without unread state", () => {
  let now = 1000;
  const events = [];
  const machine = new TaskStateMachine({ now: () => now, onState: (event) => events.push(event) });
  machine.observe(signals({ visible: true, focused: true }));
  now = 1100;
  machine.observe(signals({ running: true, visible: true, focused: true }));
  now = 2000;
  machine.observe(signals({ assistantCount: 3, visible: true, focused: true }));
  now = 3500;
  machine.observe(signals({ assistantCount: 3, visible: true, focused: true }));
  assert.equal(events.at(-1).state, "viewed");
  assert.equal(events.some((event) => event.state === "done_unread"), false);
});

test("missing new assistant structure degrades to unknown instead of guessing completion", () => {
  let now = 1000;
  const events = [];
  const machine = new TaskStateMachine({ now: () => now, onState: (event) => events.push(event) });
  machine.observe(signals());
  now = 1100;
  machine.observe(signals({ running: true }));
  now = 2000;
  machine.observe(signals({ assistantCount: 2 }));
  now = 2800;
  machine.observe(signals({ assistantCount: 2 }));
  assert.equal(events.at(-1).state, "unknown");
  assert.equal(events.some((event) => event.state === "done_unread"), false);
});

test("explicit failure and insufficient selectors produce error and unknown", () => {
  let now = 1000;
  const events = [];
  const machine = new TaskStateMachine({ now: () => now, onState: (event) => events.push(event) });
  machine.observe(signals());
  now += 1;
  machine.observe(signals({ error: true }));
  now += 1;
  machine.observe(signals({ confident: false }));
  assert.deepEqual(events.map((event) => event.state), ["idle", "error", "unknown"]);
});
