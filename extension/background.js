const ENDPOINT = 'http://127.0.0.1:9477/ingest'
const PAGE_ID = 'faultline-extra-page'
const BRIDGE_ID = 'faultline-extra-content'

chrome.runtime.onMessage.addListener((msg) => {
  if (!msg || msg.type !== 'faultline-error' || !msg.payload) {
    return
  }
  fetch(ENDPOINT, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(msg.payload),
    keepalive: true,
  }).catch(() => {})
})

chrome.runtime.onInstalled.addListener(() => {
  syncExtraScripts()
})
chrome.runtime.onStartup.addListener(() => {
  syncExtraScripts()
})
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'sync' && changes.extraHosts) {
    syncExtraScripts()
  }
})

async function syncExtraScripts() {
  const { extraHosts = [] } = await chrome.storage.sync.get({ extraHosts: [] })
  const matches = hostLinesToMatches(extraHosts)
  try {
    await chrome.scripting.unregisterContentScripts({ ids: [PAGE_ID, BRIDGE_ID] })
  } catch (_) {
    /* not registered yet */
  }
  if (!matches.length) return
  await chrome.scripting.registerContentScripts([
    {
      id: PAGE_ID,
      js: ['page.js'],
      matches,
      runAt: 'document_start',
      world: 'MAIN',
      allFrames: true,
      persistAcrossSessions: true,
    },
    {
      id: BRIDGE_ID,
      js: ['content.js'],
      matches,
      runAt: 'document_start',
      allFrames: true,
      persistAcrossSessions: true,
    },
  ])
}

function hostLinesToMatches(lines) {
  const out = []
  const seen = new Set()
  for (const line of lines) {
    const match = toMatchPattern(line)
    if (!match || seen.has(match)) continue
    seen.add(match)
    out.push(match)
  }
  return out
}

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
