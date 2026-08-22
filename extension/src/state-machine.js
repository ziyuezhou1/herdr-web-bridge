(function initStateMachine(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function stateMachineFactory() {
  "use strict";

  class TaskStateMachine {
    constructor(options = {}) {
      this.now = options.now || (() => Date.now());
      this.onState = options.onState || (() => {});
      this.completionStableMs = options.completionStableMs || 800;
      this.viewStableMs = options.viewStableMs || 1500;
      this.state = null;
      this.baselineAssistantCount = null;
      this.lastStableAssistantCount = 0;
      this.generationAssistantCount = 0;
      this.generationId = "";
      this.completionSince = 0;
      this.activeSince = 0;
    }

    observe(signals) {
      const now = this.now();
      const assistantCount = Number.isInteger(signals.assistantCount) ? signals.assistantCount : 0;
      if (this.baselineAssistantCount === null) {
        this.baselineAssistantCount = assistantCount;
        this.lastStableAssistantCount = assistantCount;
        this.generationAssistantCount = assistantCount;
      }

      if (!signals.confident) {
        this.completionSince = 0;
        return this.transition("unknown", "", "adapter_signals_insufficient");
      }

      if (signals.error) {
        const eventId = this.generationId || this.newEventId(now);
        return this.transition("error", eventId, "explicit_retry_or_failure_signal");
      }

      if (signals.running) {
        this.completionSince = 0;
        this.activeSince = 0;
        if (this.state !== "running") {
          this.generationAssistantCount = this.lastStableAssistantCount;
          this.generationId = this.newEventId(now);
          return this.transition("running", this.generationId, "generation_signal_present");
        }
        return null;
      }

      if (this.state === "running") {
        const hasNewAssistant = assistantCount > this.generationAssistantCount;
        if (!this.completionSince) this.completionSince = now;
        if (!hasNewAssistant) {
          if (now - this.completionSince >= this.completionStableMs) {
            return this.transition("unknown", this.generationId, "completion_without_new_assistant_structure");
          }
          return null;
        }
        const active = Boolean(signals.visible && signals.focused);
        if (active && !this.activeSince) this.activeSince = now;
        if (!active) this.activeSince = 0;
        if (now - this.completionSince < this.completionStableMs) return null;
        if (active) {
          if (now - this.activeSince >= this.viewStableMs) {
            this.lastStableAssistantCount = assistantCount;
            return this.transition("viewed", this.generationId, "completed_while_viewed");
          }
          return null;
        }
        this.lastStableAssistantCount = assistantCount;
        return this.transition("done_unread", this.generationId, "new_assistant_structure_stable");
      }

      if (this.state === "done_unread") {
        const active = Boolean(signals.visible && signals.focused);
        if (active && !this.activeSince) this.activeSince = now;
        if (!active) this.activeSince = 0;
        if (active && now - this.activeSince >= this.viewStableMs) {
          this.lastStableAssistantCount = assistantCount;
          return this.transition("viewed", this.generationId, "page_visible_and_focused");
        }
        return null;
      }

      if (this.state === null) {
        this.lastStableAssistantCount = assistantCount;
        return this.transition("idle", "", "history_baseline_loaded");
      }
      if (this.state === "idle" || this.state === "viewed" || this.state === "error") {
        this.lastStableAssistantCount = assistantCount;
      }
      return null;
    }

    transition(next, eventId, reason) {
      if (this.state === next) return null;
      this.state = next;
      const event = { state: next, eventId, reason, at: new Date(this.now()).toISOString() };
      this.onState(event);
      return event;
    }

    nextCheckDelay(signals) {
      const now = this.now();
      if (this.state === "running" && this.completionSince) {
        let due = this.completionSince + this.completionStableMs;
        if (signals.visible && signals.focused && this.activeSince) {
          due = Math.max(due, this.activeSince + this.viewStableMs);
        }
        return Math.max(0, due - now);
      }
      if (this.state === "done_unread" && signals.visible && signals.focused && this.activeSince) {
        return Math.max(0, this.activeSince + this.viewStableMs - now);
      }
      return null;
    }

    newEventId(now) {
      return `chatgpt-${now}-${Math.random().toString(16).slice(2, 10)}`;
    }
  }

  return { TaskStateMachine };
});
