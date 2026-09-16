(() => {
  if (window.__faultlineCapture) return
  window.__faultlineCapture = true

  const recent = new Map()
  const noisyLib =
    /(?:^|\/)(?:vue(?:\.runtime)?(?:\.esm)?(?:\.min)?\.js|react(?:-dom)?(?:\.development|\.production\.min)?\.js)(?:\?|$)/i

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

  function locFromStack(stack) {
    if (!stack) return { file: '', line: 0, column: 0 }
    const re =
      /((?:https?:\/\/|webpack-internal:\/\/\/|file:\/\/)?[^\s)]+\.(?:js|jsx|mjs|cjs|ts|tsx|vue|svelte))(?:\?[^:)\s]*)?:(\d+)(?::(\d+))?/gi
    let fallback = null
    let m
    while ((m = re.exec(stack))) {
      const file = m[1]
      const loc = {
        file,
        line: Number(m[2]) || 0,
        column: Number(m[3]) || 0,
      }
      if (noisy(file, '') || noisyLib.test(file)) {
        if (!fallback) fallback = loc
        continue
      }
      return loc
    }
    return fallback || { file: '', line: 0, column: 0 }
  }

  function fromError(err, severity) {
    const stack = (err && err.stack) || ''
    const loc = locFromStack(stack)
    return {
      type: (err && err.name) || 'Error',
      message: (err && err.message) || String(err),
      file: loc.file,
      line: loc.line,
      column: loc.column,
      stack,
      url: location.href,
      severity: severity || 'error',
    }
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

  function argMessage(args) {
    return args
      .map((a) => {
        if (a == null) return String(a)
        if (typeof a === 'string') return a
        if (a instanceof Error) return a.message || a.name
        try {
          return String(a)
        } catch (_) {
          return ''
        }
      })
      .join(' ')
      .trim()
      .slice(0, 4000)
  }

  // Vue (and similar) catch handler errors and print them instead of rethrowing,
  // so they never become window "error" or unhandledrejection events.
  function hookConsole(method, severity) {
    const orig = console[method]
    if (typeof orig !== 'function') return
    console[method] = function (...args) {
      try {
        const err = args.find((a) => a instanceof Error)
        if (err) {
          send(fromError(err, severity))
        } else {
          const msg = argMessage(args)
          const lower = msg.toLowerCase()
          const vueSwallowed = lower.includes('[vue warn]') && lower.includes('error in')
          const looksThrown = /^(?:uncaught )?[\w$]*(?:error|exception)\b/i.test(msg)
          const looksHTTP = severity === 'error' && /^(?:http\/\d(?:\.\d)?\s+)?[45]\d\d\b/i.test(msg)
          if (vueSwallowed || looksThrown || looksHTTP) {
            send({
              type: vueSwallowed ? 'VueError' : 'ConsoleError',
              message: msg,
              file: '',
              line: 0,
              column: 0,
              stack: '',
              url: location.href,
              severity: vueSwallowed ? 'warning' : severity,
            })
          }
        }
      } catch (_) {}
      return orig.apply(this, args)
    }
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
      let column = 0
      if (reason instanceof Error) {
        const loc = locFromStack(reason.stack || '')
        type = reason.name || type
        message = reason.message || message
        stack = reason.stack || ''
        file = loc.file
        line = loc.line
        column = loc.column
      } else if (reason != null) {
        message = String(reason)
      }
      send({
        type,
        message,
        file,
        line,
        column,
        stack,
        url: location.href,
        severity: 'error',
      })
    },
    true
  )

  hookConsole('error', 'error')
  hookConsole('warn', 'warning')

  const origFetch = window.fetch
  if (typeof origFetch === 'function') {
    window.fetch = function (...args) {
      return origFetch.apply(this, args).then((res) => {
        try {
          const url = String((res && res.url) || '')
          if (
            res &&
            !res.ok &&
            res.status !== 404 &&
            !url.includes('127.0.0.1:9477') &&
            !url.includes('localhost:9477')
          ) {
            const status = (res.status || 0) + ' ' + (res.statusText || '') + ' ' + url
            const sendFail = (extra) => {
              send({
                type: 'FailedFetch',
                message: extra ? status + '\n' + extra : status,
                file: location.href,
                line: 0,
                column: 0,
                stack: '',
                url: location.href,
                severity: res.status >= 500 ? 'error' : 'warning',
              })
            }
            try {
              res
                .clone()
                .text()
                .then((t) => sendFail(String(t || '').slice(0, 500)))
                .catch(() => sendFail(''))
            } catch (_) {
              sendFail('')
            }
          }
        } catch (_) {}
        return res
      })
    }
  }

  const OrigXHR = window.XMLHttpRequest
  if (typeof OrigXHR === 'function') {
    function WrappedXHR() {
      const xhr = new OrigXHR()
      xhr.addEventListener('loadend', () => {
        try {
          const url = String(xhr.responseURL || '')
          if (
            xhr.status >= 400 &&
            xhr.status !== 404 &&
            !url.includes('127.0.0.1:9477') &&
            !url.includes('localhost:9477')
          ) {
            const extra = xhrBody(xhr)
            send({
              type: 'FailedXHR',
              message: xhr.status + ' ' + (xhr.statusText || '') + ' ' + url + (extra ? '\n' + extra : ''),
              file: location.href,
              line: 0,
              column: 0,
              stack: '',
              url: location.href,
              severity: xhr.status >= 500 ? 'error' : 'warning',
            })
          }
        } catch (_) {}
      })
      return xhr
    }
    WrappedXHR.prototype = OrigXHR.prototype
    window.XMLHttpRequest = WrappedXHR
  }

  function xhrBody(xhr) {
    try {
      const rt = xhr.responseType
      if (!rt || rt === '' || rt === 'text') {
        return String(xhr.responseText || '').slice(0, 500)
      }
      if (rt === 'json') {
        const v = xhr.response
        if (v == null) return ''
        if (typeof v === 'string') return v.slice(0, 500)
        try {
          return JSON.stringify(v).slice(0, 500)
        } catch (_) {
          return String(v).slice(0, 500)
        }
      }
      if (typeof xhr.response === 'string') {
        return xhr.response.slice(0, 500)
      }
    } catch (_) {}
    return ''
  }

  function viteMessage(el) {
    if (!el) return ''
    try {
      if (el.shadowRoot && el.shadowRoot.textContent) return el.shadowRoot.textContent.trim()
    } catch (_) {}
    return (el.textContent || '').trim()
  }

  function sendVite(el) {
    const msg = viteMessage(el).slice(0, 4000)
    if (!msg) return
    send({
      type: 'ViteError',
      message: msg,
      file: '',
      line: 0,
      column: 0,
      stack: msg,
      url: location.href,
      severity: 'error',
    })
  }

  function hookViteOverlay() {
    const seen = new WeakSet()
    const scan = () => {
      document.querySelectorAll('vite-error-overlay').forEach((el) => {
        if (seen.has(el)) return
        seen.add(el)
        sendVite(el)
      })
    }
    scan()
    const obs = new MutationObserver(scan)
    obs.observe(document.documentElement, { childList: true, subtree: true })
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', hookViteOverlay)
  } else {
    hookViteOverlay()
  }
})()
