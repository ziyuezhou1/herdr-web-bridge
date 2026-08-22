"use strict";

const taskId = "resume-2026-001";
document.querySelectorAll("[data-state]").forEach((button) => {
  button.addEventListener("click", () => {
    const state = button.dataset.state;
    const messages = {
      running: "正在生成岗位定制简历",
      done: "PDF 已生成",
      error: "PDF 生成失败",
      idle: "等待新任务"
    };
    window.postMessage({
      source: "herdr-web-bridge",
      version: 1,
      type: "task-status",
      taskId,
      state,
      title: "简历优化",
      message: messages[state]
    }, window.location.origin);
    document.getElementById("status").textContent = `已发送：${state}`;
  });
});
