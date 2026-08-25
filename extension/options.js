const hostsEl = document.getElementById('hosts')
const statusEl = document.getElementById('status')
const saveEl = document.getElementById('save')

function toMatchPattern(raw) {
  let host = String(raw || '').trim().toLowerCase()
  if (!host || host.startsWith('#')) return ''
  if (host.includes('://')) {
    try {
      host = new URL(host).hostname
    } catch {
      return ''
    }
  } else {
    host = host.replace(/\/.*$/, '')
    host = host.replace(/:\d+$/, '')
  }
  if (!/^(\*\.)?[a-z0-9][a-z0-9.-]*[a-z0-9]$/.test(host) && host !== 'localhost') {
    return ''
  }
  const bare = host.startsWith('*.') ? host.slice(2) : host
  if (!bare.includes('.') && bare !== 'localhost') {
    return ''
  }
  return `*://${host}/*`
}

function linesFromText(text) {
  return text
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter((s) => s && !s.startsWith('#'))
}

function showStatus(kind, msg) {
  statusEl.hidden = false
  statusEl.className = 'status ' + kind
  statusEl.textContent = msg
}

chrome.storage.sync.get({ extraHosts: [] }, (st) => {
  hostsEl.value = (st.extraHosts || []).join('\n')
})

saveEl.addEventListener('click', async () => {
  saveEl.disabled = true
  const extraHosts = linesFromText(hostsEl.value)
  const origins = []
  const seen = new Set()
  for (const line of extraHosts) {
    const match = toMatchPattern(line)
    if (!match) {
      showStatus('err', 'Not a hostname: ' + line)
      saveEl.disabled = false
      return
    }
    if (!seen.has(match)) {
      seen.add(match)
      origins.push(match)
    }
  }
  try {
    if (origins.length) {
      const ok = await chrome.permissions.request({ origins })
      if (!ok) {
        showStatus('err', 'Permission was not granted for those hosts.')
        saveEl.disabled = false
        return
      }
    }
    await chrome.storage.sync.set({ extraHosts })
    showStatus('ok', extraHosts.length ? 'Saved. Reload the site tab if it is already open.' : 'Saved. Extra hosts cleared.')
  } catch (err) {
    showStatus('err', String(err && err.message ? err.message : err))
  }
  saveEl.disabled = false
})
