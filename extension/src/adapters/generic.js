(function initGenericAdapter(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function genericFactory() {
  "use strict";
  return {
    genericCapabilities: Object.freeze({ automaticDetection: false, manualStates: ["running", "done_unread", "viewed", "error", "idle"] })
  };
});
