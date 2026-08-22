"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const roots = ["extension", "examples", "scripts"];
const files = [];

function collect(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const item = path.join(directory, entry.name);
    if (entry.isDirectory()) collect(item);
    else if (entry.isFile() && item.endsWith(".js")) files.push(item);
  }
}

for (const root of roots) {
  if (fs.existsSync(root)) collect(root);
}

let failed = false;
for (const file of files.sort()) {
  const result = spawnSync(process.execPath, ["--check", file], { stdio: "inherit" });
  if (result.status !== 0) failed = true;
}

if (failed) process.exit(1);
console.log(`JavaScript syntax passed for ${files.length} files.`);
