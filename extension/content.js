if (window.__faultlineBridge) {
  /* already injected (manifest + extra hosts) */
} else {
  window.__faultlineBridge = true
  window.addEventListener('faultline:error', (ev) => {
    const payload = ev && ev.detail
    if (!payload) return
    try {
      chrome.runtime.sendMessage({ type: 'faultline-error', payload }, () => {
        void chrome.runtime.lastError
      })
    } catch (_) {
      /* extension reloaded, or Faultline is not running */
    }
  })
}
