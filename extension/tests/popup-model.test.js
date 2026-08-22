"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const popupModel = require("../src/popup-model.js");

test("empty workspace response produces an actionable warning", () => {
  const summary = popupModel.workspaceSummary([]);
  assert.equal(summary.bindableCount, 0);
  assert.match(summary.placeholder, /未读取到/);
  assert.match(summary.warning, /list-workspaces/);
});

test("workspaces without resolvable public paths remain visible but unbindable", () => {
  const summary = popupModel.workspaceSummary([
    { label: "VirtualDNA", pathAvailable: false },
    { label: "简历优化", pathAvailable: false }
  ]);
  assert.equal(summary.bindableCount, 0);
  assert.match(summary.warning, /pane cwd/);
});

test("workspace path source and failure reason are auditable", () => {
  assert.match(
    popupModel.workspaceOptionLabel({ label: "Plain", pathAvailable: true, projectPath: "D:\\Plain", pathSource: "pane_cwd" }),
    /\[pane cwd\]/
  );
  assert.match(
    popupModel.workspacePathHint({ pathAvailable: true, projectPath: "D:\\Plain", pathSource: "pane_cwd" }),
    /普通工作区/
  );
  assert.match(
    popupModel.workspacePathHint({ pathAvailable: false, pathReason: "ambiguous_pane_cwd" }),
    /多个窗格位于不同目录/
  );
  assert.match(
    popupModel.workspacePathHint({ pathAvailable: false, pathReason: "pane_list_unavailable" }),
    /pane list/
  );
});

test("native host and Herdr failures receive distinct repair guidance", () => {
  assert.match(
    popupModel.bridgeErrorMessage(new Error("Specified native messaging host not found.")),
    /installer\\install\.ps1/
  );
  assert.match(
    popupModel.bridgeErrorMessage(new Error("找不到指定的原生消息传递主机。")),
    /installer\\install\.ps1/
  );
  assert.match(
    popupModel.bridgeErrorMessage(Object.assign(new Error("herdr command failed"), { code: "herdr_unavailable" })),
    /herdr workspace list/
  );
});

test("workspace selector stays inspectable when no workspace is bindable", () => {
  const empty = popupModel.bindingControls({ busy: false, isBound: false, workspace: null });
  assert.equal(empty.workspaceDisabled, false);
  assert.equal(empty.bindDisabled, true);

  const pathless = popupModel.bindingControls({
    busy: false,
    isBound: false,
    workspace: { label: "VirtualDNA", pathAvailable: false }
  });
  assert.equal(pathless.workspaceDisabled, false);
  assert.equal(pathless.bindDisabled, true);

  const bindable = popupModel.bindingControls({
    busy: false,
    isBound: false,
    workspace: { pathAvailable: true, projectPath: "T:\\VirtualDNA" }
  });
  assert.equal(bindable.workspaceDisabled, false);
  assert.equal(bindable.bindDisabled, false);
});
