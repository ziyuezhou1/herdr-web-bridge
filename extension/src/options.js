"use strict";

document.getElementById("extension-id").textContent = chrome.runtime.id;
document.getElementById("retry").addEventListener("click", check);
check();

function check() {
  const status = document.getElementById("status");
  status.className = "";
  status.textContent = "检查中…";
  chrome.runtime.sendMessage({ type: "connection_status", payload: {} }, (response) => {
    if (chrome.runtime.lastError || !response || !response.ok) {
      status.className = "error";
      status.textContent = chrome.runtime.lastError ? chrome.runtime.lastError.message : (response && response.error ? response.error.message : "未连接");
      return;
    }
    status.className = "ok";
    status.textContent = "已连接";
  });
}

