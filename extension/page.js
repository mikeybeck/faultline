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
    const key = `${payload.type}|${msg}|${payload.request || ''}|${payload.file || ''}|${payload.line || 0}`
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
      return orig.apply(console, args)
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

  const BODY_CAP = 65536
  const TEXT_CAP = 8192
  // Bytes we are willing to buffer. Larger than the snippet we report, so a
  // normal API payload is read to the end. Past this we skip the body.
  const PEEK_LIMIT = 1048576

  function ignoredURL(url) {
    return url.includes('127.0.0.1:9477') || url.includes('localhost:9477')
  }

  function reportNet(type, status, statusText, url, extra, severity, request) {
    const line = (status || 0) + ' ' + (statusText || '') + ' ' + url
    const req = String(request || '').trim()
    send({
      type,
      message: extra ? line + '\n' + extra : line,
      file: location.href,
      line: 0,
      column: 0,
      stack: '',
      url: location.href,
      severity,
      request: req,
    })
  }

  function capText(text, max, mark) {
    const s = String(text || '')
    if (s.length <= max) return s
    if (!mark) return s.slice(0, max)
    return s.slice(0, max) + '\n…'
  }

  function isBinary(text) {
    return String(text || '').indexOf('\0') >= 0
  }

  function binaryNote(n) {
    const size = Number.isFinite(n) && n >= 0 ? n : 0
    return '(binary, ' + size + ' bytes)'
  }

  function formText(fd) {
    const parts = []
    try {
      for (const [k, v] of fd.entries()) {
        if (typeof v === 'string') {
          parts.push(k + '=' + v)
        } else {
          const name = v && v.name ? v.name : 'file'
          const size = v && typeof v.size === 'number' ? v.size : 0
          parts.push(k + '=(' + name + ', ' + size + ' bytes)')
        }
      }
    } catch (_) {}
    return capText(parts.join('&'), TEXT_CAP, true)
  }

  function bytesNote(view) {
    const n = view ? view.byteLength : 0
    if (n > PEEK_LIMIT) return Promise.resolve(binaryNote(n))
    const sample = view.subarray(0, Math.min(n, TEXT_CAP + 1))
    for (let i = 0; i < sample.length; i++) {
      if (sample[i] === 0) return Promise.resolve(binaryNote(n))
    }
    let text = ''
    try {
      text = new TextDecoder('utf-8', { fatal: false }).decode(sample)
    } catch (_) {
      return Promise.resolve(binaryNote(n))
    }
    if (isBinary(text)) return Promise.resolve(binaryNote(n))
    return Promise.resolve(capText(text, TEXT_CAP, true))
  }

  // Snapshot a request body without consuming the one fetch or XHR will send.
  // A ReadableStream can only be read once, and canceling a tee can crash the
  // renderer, so those bodies are left out.
  function bodyText(body) {
    if (body == null || body === '') return Promise.resolve('')
    if (typeof body === 'string') {
      if (isBinary(body)) return Promise.resolve(binaryNote(body.length))
      return Promise.resolve(capText(body, TEXT_CAP, true))
    }
    if (typeof URLSearchParams !== 'undefined' && body instanceof URLSearchParams) {
      return Promise.resolve(capText(body.toString(), TEXT_CAP, true))
    }
    if (typeof FormData !== 'undefined' && body instanceof FormData) {
      return Promise.resolve(formText(body))
    }
    if (typeof Blob !== 'undefined' && body instanceof Blob) {
      if (body.size > PEEK_LIMIT) return Promise.resolve(binaryNote(body.size))
      return body
        .text()
        .then((t) => (isBinary(t) ? binaryNote(body.size) : capText(t, TEXT_CAP, true)))
        .catch(() => '')
    }
    if (typeof ArrayBuffer !== 'undefined' && body instanceof ArrayBuffer) {
      return bytesNote(new Uint8Array(body))
    }
    if (typeof ArrayBuffer !== 'undefined' && ArrayBuffer.isView(body)) {
      return bytesNote(new Uint8Array(body.buffer, body.byteOffset, body.byteLength))
    }
    return Promise.resolve('')
  }

  function formatRequest(method, body) {
    const text = String(body || '').trim()
    if (!text) return ''
    const m = String(method || '').trim()
    if (!m) return text
    return m + '\n' + text
  }

  function methodOf(input, init) {
    if (init && init.method) return String(init.method).toUpperCase()
    if (typeof Request !== 'undefined' && input instanceof Request && input.method) {
      return String(input.method).toUpperCase()
    }
    return 'GET'
  }

  function urlFromFetchArgs(input) {
    try {
      if (typeof input === 'string') return input
      if (typeof URL !== 'undefined' && input instanceof URL) return String(input)
      if (typeof Request !== 'undefined' && input instanceof Request) return String(input.url || '')
      if (input && input.url) return String(input.url)
    } catch (_) {}
    return ''
  }

  function readFetchBody(input, init) {
    const method = methodOf(input, init)
    if (init && init.body != null) {
      return bodyText(init.body)
        .then((t) => formatRequest(method, t))
        .catch(() => '')
    }
    if (typeof Request !== 'undefined' && input instanceof Request) {
      let len = NaN
      try {
        const raw = input.headers && input.headers.get('content-length')
        if (raw) len = Number(raw)
      } catch (_) {}
      if (Number.isFinite(len) && len > PEEK_LIMIT) {
        return Promise.resolve(formatRequest(method, binaryNote(len)))
      }
      try {
        const clone = input.clone()
        return clone
          .text()
          .then((t) => {
            if (isBinary(t)) return formatRequest(method, binaryNote(t.length))
            return formatRequest(method, capText(t, TEXT_CAP, true))
          })
          .catch(() => '')
      } catch (_) {
        return Promise.resolve('')
      }
    }
    return Promise.resolve('')
  }

  // Skip bodies that cannot be a JSON object. text/* is included so a JSON
  // payload served as text/plain or text/html is still inspected.
  // event-stream stays untouched so a live feed is not buffered forever.
  function shouldPeek(contentType) {
    const ct = String(contentType || '').toLowerCase()
    if (ct.includes('event-stream')) return false
    if (!ct || ct.includes('json') || ct.startsWith('text/')) return true
    if (/^(image|audio|video|font)\//.test(ct)) return false
    if (
      ct.includes('javascript') ||
      ct.includes('ecmascript') ||
      ct.includes('wasm') ||
      ct.includes('octet-stream') ||
      ct.includes('pdf')
    ) {
      return false
    }
    return true
  }

  function errorText(data) {
    const err = data.error
    if (typeof err === 'string' && err.trim()) return err.trim()
    if (
      err &&
      typeof err === 'object' &&
      !Array.isArray(err) &&
      typeof err.message === 'string' &&
      err.message.trim()
    ) {
      return err.message.trim()
    }
    if (typeof data.message === 'string' && data.message.trim()) return data.message.trim()
    return ''
  }

  function appErrorFromValue(data) {
    if (!data || typeof data !== 'object' || Array.isArray(data)) return ''
    if (data.success !== false) return ''
    return errorText(data).slice(0, 500)
  }

  function appFailureMessage(text) {
    const capped = String(text || '').slice(0, BODY_CAP)
    if (!capped.trimStart().startsWith('{')) return ''
    let data
    try {
      data = JSON.parse(capped)
    } catch (_) {
      return ''
    }
    return appErrorFromValue(data)
  }

  function declaredLength(res) {
    try {
      const raw = res.headers && res.headers.get('content-length')
      if (raw == null || raw === '') return NaN
      const n = Number(raw)
      return Number.isFinite(n) && n >= 0 ? n : NaN
    } catch (_) {
      return NaN
    }
  }

  // Downloads and media are not error payloads. text/* and JSON are, and a
  // missing type is allowed so a chunked HTML error page is still read.
  function skipHeavyBody(contentType) {
    const ct = String(contentType || '').toLowerCase()
    if (ct.includes('event-stream')) return true
    if (/^(image|audio|video|font)\//.test(ct)) return true
    return (
      ct.includes('javascript') ||
      ct.includes('ecmascript') ||
      ct.includes('wasm') ||
      ct.includes('octet-stream') ||
      ct.includes('pdf')
    )
  }

  function errorBodyAllowed(contentType) {
    if (skipHeavyBody(contentType)) return false
    const ct = String(contentType || '').toLowerCase()
    return !ct || ct.includes('json') || ct.startsWith('text/')
  }

  // Read a clone to completion and leave the page's Response alone. Do not
  // read with getReader() and cancel: canceling one branch of a fetch tee
  // crashes the renderer (STATUS_ACCESS_VIOLATION) while the page reads the
  // other branch. Skip bodies that are too large to finish. For a failed
  // response, also read chunked JSON or text when the length is unknown.
  function peekText(res, max, failed) {
    const len = declaredLength(res)
    if (Number.isFinite(len) && len > PEEK_LIMIT) return Promise.resolve('')
    let ct = ''
    try {
      ct = (res.headers && res.headers.get('content-type')) || ''
    } catch (_) {}
    if (!Number.isFinite(len)) {
      if (failed) {
        if (!errorBodyAllowed(ct)) return Promise.resolve('')
      } else if (!String(ct).toLowerCase().includes('json')) {
        return Promise.resolve('')
      }
    } else if (failed && skipHeavyBody(ct)) {
      return Promise.resolve('')
    }
    let clone
    try {
      clone = res.clone()
    } catch (_) {
      return Promise.resolve('')
    }
    return clone
      .text()
      .then((t) => capText(t, max, !!failed))
      .catch(() => '')
  }

  const origFetch = window.fetch
  if (typeof origFetch === 'function') {
    window.fetch = function (...args) {
      let bodyPromise = Promise.resolve('')
      let urlHint = ''
      try {
        bodyPromise = readFetchBody(args[0], args[1])
        urlHint = urlFromFetchArgs(args[0])
      } catch (_) {}
      return origFetch.apply(this, args).then(
        (res) => {
          try {
            const url = String((res && res.url) || urlHint || '')
            if (!res || ignoredURL(url)) return res
            const failed = res.status >= 400 && res.status !== 404
            let ct = ''
            try {
              ct = (res.headers && res.headers.get('content-type')) || ''
            } catch (_) {}
            const appFail = res.ok && shouldPeek(ct)
            if (!failed && !appFail) return res
            const max = failed ? TEXT_CAP : BODY_CAP
            const textPromise =
              !res.body || res.bodyUsed ? Promise.resolve('') : peekText(res, max, failed)
            Promise.all([textPromise, bodyPromise])
              .then(([text, request]) => {
                try {
                  if (failed) {
                    reportNet(
                      'FailedFetch',
                      res.status,
                      res.statusText,
                      url,
                      text,
                      res.status >= 500 ? 'error' : 'warning',
                      request
                    )
                  } else {
                    const err = appFailureMessage(text)
                    if (err) {
                      reportNet('FailedFetch', res.status, res.statusText, url, err, 'warning', request)
                    }
                  }
                } catch (_) {}
              })
              .catch(() => {})
          } catch (_) {}
          return res
        },
        (err) => {
          try {
            if (!ignoredURL(urlHint)) {
              bodyPromise
                .then((request) => {
                  try {
                    reportNet(
                      'FailedFetch',
                      0,
                      (err && err.message) || 'Network error',
                      urlHint,
                      '',
                      'error',
                      request
                    )
                  } catch (_) {}
                })
                .catch(() => {})
            }
          } catch (_) {}
          return Promise.reject(err)
        }
      )
    }
  }

  const OrigXHR = window.XMLHttpRequest
  if (typeof OrigXHR === 'function') {
    const origOpen = OrigXHR.prototype.open
    const origSend = OrigXHR.prototype.send
    OrigXHR.prototype.open = function (method, url) {
      try {
        this.__flMethod = String(method || 'GET').toUpperCase()
        this.__flURL = String(url || '')
        this.__flAborted = false
        this.__flSent = false
      } catch (_) {}
      return origOpen.apply(this, arguments)
    }
    OrigXHR.prototype.send = function (body) {
      try {
        this.__flSent = true
        this.__flBodyP = bodyText(body).catch(() => '')
      } catch (_) {
        try {
          this.__flBodyP = Promise.resolve('')
        } catch (_) {}
      }
      return origSend.apply(this, arguments)
    }
    function WrappedXHR() {
      const xhr = new OrigXHR()
      xhr.addEventListener('abort', () => {
        try {
          xhr.__flAborted = true
        } catch (_) {}
      })
      xhr.addEventListener('loadend', () => {
        try {
          const url = String(xhr.responseURL || xhr.__flURL || '')
          if (ignoredURL(url)) return
          const requestP = xhr.__flBodyP || Promise.resolve('')
          const report = (status, statusText, extra, severity) => {
            requestP
              .then((body) => {
                try {
                  reportNet(
                    'FailedXHR',
                    status,
                    statusText,
                    url,
                    extra,
                    severity,
                    formatRequest(xhr.__flMethod || '', body)
                  )
                } catch (_) {}
              })
              .catch(() => {})
          }
          if (xhr.status === 0) {
            if (!xhr.__flSent || xhr.__flAborted) return
            report(0, 'Network error', '', 'error')
            return
          }
          if (xhr.status >= 400 && xhr.status !== 404) {
            xhrErrorBody(xhr)
              .then((extra) => {
                report(xhr.status, xhr.statusText, extra, xhr.status >= 500 ? 'error' : 'warning')
              })
              .catch(() => {})
            return
          }
          if (xhr.status >= 200 && xhr.status < 300) {
            let ct = ''
            try {
              ct = xhr.getResponseHeader('Content-Type') || ''
            } catch (_) {}
            if (!shouldPeek(ct)) return
            const err = appFailureFromXHR(xhr)
            if (!err) return
            report(xhr.status, xhr.statusText, err, 'warning')
          }
        } catch (_) {}
      })
      return xhr
    }
    WrappedXHR.prototype = OrigXHR.prototype
    try {
      Object.setPrototypeOf(WrappedXHR, OrigXHR)
    } catch (_) {}
    for (const key of ['UNSENT', 'OPENED', 'HEADERS_RECEIVED', 'LOADING', 'DONE']) {
      try {
        WrappedXHR[key] = OrigXHR[key]
      } catch (_) {}
    }
    window.XMLHttpRequest = WrappedXHR
  }

  function xhrErrorBody(xhr) {
    try {
      const rt = xhr.responseType
      if (rt === 'json' && xhr.response == null) return Promise.resolve('')
      if (rt === 'arraybuffer' && xhr.response) {
        return bytesNote(new Uint8Array(xhr.response))
      }
      if (rt === 'blob' && xhr.response && typeof xhr.response.text === 'function') {
        const blob = xhr.response
        if (blob.size > PEEK_LIMIT) return Promise.resolve(binaryNote(blob.size))
        return blob
          .text()
          .then((t) => (isBinary(t) ? binaryNote(blob.size) : capText(t, TEXT_CAP, true)))
          .catch(() => '')
      }
    } catch (_) {
      return Promise.resolve('')
    }
    return Promise.resolve(xhrText(xhr, TEXT_CAP))
  }

  function appFailureFromXHR(xhr) {
    try {
      const rt = xhr.responseType
      if (rt === 'json') return appErrorFromValue(xhr.response)
      return appFailureMessage(xhrText(xhr, BODY_CAP))
    } catch (_) {}
    return ''
  }

  function xhrText(xhr, max) {
    try {
      const rt = xhr.responseType
      if (!rt || rt === '' || rt === 'text') {
        return capText(String(xhr.responseText || ''), max, max === TEXT_CAP)
      }
      if (rt === 'json') {
        const v = xhr.response
        if (v == null) return ''
        if (typeof v === 'string') return capText(v, max, max === TEXT_CAP)
        try {
          return capText(JSON.stringify(v), max, max === TEXT_CAP)
        } catch (_) {
          return capText(String(v), max, max === TEXT_CAP)
        }
      }
      if (typeof xhr.response === 'string') {
        return capText(xhr.response, max, max === TEXT_CAP)
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
    if (!document.documentElement) return
    const obs = new MutationObserver(scan)
    try {
      obs.observe(document.documentElement, { childList: true, subtree: true })
    } catch (_) {
      return
    }
    window.addEventListener('pagehide', () => obs.disconnect(), { once: true })
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', hookViteOverlay)
  } else {
    hookViteOverlay()
  }
})()
