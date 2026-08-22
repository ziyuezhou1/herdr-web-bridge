(function initTabMatcher(root, factory) {
  const api = factory(root.HerdrBridge || {});
  if (typeof module === "object" && module.exports) module.exports = api;
  root.HerdrBridge = Object.assign(root.HerdrBridge || {}, api);
})(typeof globalThis !== "undefined" ? globalThis : this, function tabMatcherFactory(shared) {
  "use strict";

  function findBoundTab(tabs, binding, cachedTabId) {
    const usable = (tabs || []).filter((tab) => tab && typeof tab.url === "string");
    if (Number.isInteger(cachedTabId)) {
      const cached = usable.find((tab) => tab.id === cachedTabId && shared.urlsMatch(tab.url, binding.url));
      if (cached) return cached;
    }
    const exact = usable.find((tab) => tab.url === binding.url);
    if (exact) return exact;
    return usable.find((tab) => shared.urlsMatch(tab.url, binding.url)) || null;
  }

  return { findBoundTab };
});

