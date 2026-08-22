"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");

const manifest = JSON.parse(fs.readFileSync(path.join(__dirname, "..", "manifest.json"), "utf8"));

test("manifest keeps sensitive permissions absent and ChatGPT access explicit", () => {
  const prohibited = ["cookies", "history", "downloads", "clipboardRead", "<all_urls>"];
  const serializedPermissions = JSON.stringify({ permissions: manifest.permissions, hosts: manifest.host_permissions });
  prohibited.forEach((permission) => assert.equal(serializedPermissions.includes(permission), false, permission));
  assert.deepEqual(manifest.host_permissions, ["https://chatgpt.com/*"]);
  assert.equal(manifest.optional_host_permissions.includes("https://*/*"), true);
});

test("manifest public key derives the installer extension ID", () => {
  const der = Buffer.from(manifest.key, "base64");
  const digest = crypto.createHash("sha256").update(der).digest().subarray(0, 16);
  const alphabet = "abcdefghijklmnop";
  let id = "";
  for (const byte of digest) id += alphabet[byte >> 4] + alphabet[byte & 15];
  assert.equal(id, "pphgcjjepkodhghpncncnmikafkdjdjd");
});
