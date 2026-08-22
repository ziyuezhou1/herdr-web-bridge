(function initChatGPTAdapter(root, factory) {
  const api = factory(root.HerdrBridge || {});
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function chatGPTFactory(shared) {
  "use strict";

  const SELECTORS = {
    stop: ["[data-testid='stop-button']", "button[aria-label*='Stop']", "button[aria-label*='停止']"],
    retry: ["[data-testid*='retry']", "button[aria-label*='Retry']", "button[aria-label*='重试']"],
    streaming: ["[data-is-streaming='true']", "[data-testid='conversation-turn-streaming']"],
    assistant: ["[data-message-author-role='assistant']", "article[data-turn='assistant']"],
    anchor: ["main", "[role='main']", "[data-testid^='conversation-turn-']"],
    composer: ["#prompt-textarea", "[data-testid='composer']", "textarea", "div[contenteditable='true']"]
  };

  function any(documentLike, selectors) {
    return selectors.some((selector) => Boolean(documentLike.querySelector(selector)));
  }

  function count(documentLike, selectors) {
    let maximum = 0;
    for (const selector of selectors) {
      const nodes = documentLike.querySelectorAll(selector);
      maximum = Math.max(maximum, nodes ? nodes.length : 0);
    }
    return maximum;
  }

  function collectChatGPTSignals(documentLike, windowLike) {
    const stop = any(documentLike, SELECTORS.stop);
    const streaming = any(documentLike, SELECTORS.streaming);
    const retry = any(documentLike, SELECTORS.retry);
    const assistantCount = count(documentLike, SELECTORS.assistant);
    const anchor = any(documentLike, SELECTORS.anchor);
    const composer = any(documentLike, SELECTORS.composer);
    return {
      running: stop || streaming,
      error: retry && !stop && !streaming,
      assistantCount,
      visible: documentLike.visibilityState === "visible",
      focused: typeof documentLike.hasFocus === "function" ? documentLike.hasFocus() : Boolean(windowLike && windowLike.document && windowLike.document.hasFocus()),
      confident: anchor && (assistantCount > 0 || stop || streaming || retry || composer)
    };
  }

  class ChatGPTAdapter {
    constructor(documentLike, windowLike, onState, options = {}) {
      this.document = documentLike;
      this.window = windowLike;
      this.timer = null;
      const StateMachine = shared.TaskStateMachine || options.TaskStateMachine;
      this.machine = new StateMachine(Object.assign({}, options, { onState }));
      this.observer = null;
    }

    start() {
      this.sample();
      const Observer = this.window.MutationObserver;
      this.observer = new Observer(() => this.schedule());
      this.observer.observe(this.document.documentElement, { childList: true, subtree: true, attributes: true });
      this.document.addEventListener("visibilitychange", () => this.schedule());
      this.window.addEventListener("focus", () => this.schedule());
      this.window.addEventListener("blur", () => this.schedule());
    }

    schedule() {
      if (this.timer) this.window.clearTimeout(this.timer);
      this.timer = this.window.setTimeout(() => this.sample(), 100);
    }

    sample() {
      this.timer = null;
      const signals = collectChatGPTSignals(this.document, this.window);
      const result = this.machine.observe(signals);
      const nextDelay = this.machine.nextCheckDelay(signals);
      if (nextDelay !== null && !this.timer) {
        this.timer = this.window.setTimeout(() => this.sample(), Math.max(1, nextDelay));
      }
      return result;
    }

    stop() {
      if (this.observer) this.observer.disconnect();
      if (this.timer) this.window.clearTimeout(this.timer);
    }
  }

  return { SELECTORS, collectChatGPTSignals, ChatGPTAdapter };
});
