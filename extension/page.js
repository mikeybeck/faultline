(() => {
  if (window.__faultlineCapture) return
  window.__faultlineCapture = true

  const recent = new Map()

  function noisy(file, stack) {
    const s = `${file || ''} ${stack || ''}`.toLowerCase()
    return (
      s.includes('chrome-extension://') ||
      s.includes('moz-extension://') ||
      s.includes('safari-extension://')
    )
  }

  function shouldSend(key) {
    const now = Date.now()
    const last = recent.get(key) || 0
    if (now - last < 400) return false
    recent.set(key, now)
    if (recent.size > 200) {
      const cutoff = now - 5000
      for (const [k, t] of recent) {
        if (t < cutoff) recent.delete(k)
      }
    }
    return true
  }

  function send(payload) {
    if (!payload || noisy(payload.file, payload.stack)) return
    const msg = (payload.message || '').trim()
    if (msg === 'Script error.' && !payload.stack) return
    const key = `${payload.type}|${msg}|${payload.file || ''}|${payload.line || 0}`
    if (!shouldSend(key)) return
    try {
      window.dispatchEvent(new CustomEvent('faultline:error', { detail: payload }))
    } catch (_) {}
  }

  window.addEventListener(
    'error',
    (ev) => {
      if (!ev) return
      if (ev.target && ev.target !== window) return
      const err = ev.error
      send({
        type: (err && err.name) || 'Error',
        message: ev.message || (err && err.message) || 'Script error',
        file: ev.filename || '',
        line: ev.lineno || 0,
        column: ev.colno || 0,
        stack: (err && err.stack) || '',
        url: location.href,
        severity: 'error',
      })
    },
    true
  )

  window.addEventListener(
    'unhandledrejection',
    (ev) => {
      const reason = ev && ev.reason
      let type = 'UnhandledRejection'
      let message = 'Unhandled rejection'
      let stack = ''
      let file = ''
      let line = 0
      if (reason instanceof Error) {
        type = reason.name || type
        message = reason.message || message
        stack = reason.stack || ''
      } else if (reason != null) {
        message = String(reason)
      }
      send({
        type,
        message,
        file,
        line,
        stack,
        url: location.href,
        severity: 'error',
      })
    },
    true
  )
})()
