const ENDPOINT = 'http://127.0.0.1:9477/ingest'
const HEALTH = 'http://127.0.0.1:9477/health'
const HOSTS = 'http://127.0.0.1:9477/hosts'
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
  pingHealth()
})
chrome.runtime.onStartup.addListener(() => {
  syncExtraScripts()
  pingHealth()
})
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'sync' && changes.extraHosts) {
    syncExtraScripts()
  }
})

if (chrome.alarms) {
  chrome.alarms.create('faultline-health', { periodInMinutes: 1 })
  chrome.alarms.onAlarm.addListener((a) => {
    if (!a || a.name !== 'faultline-health') return
    pingHealth()
    pullHosts()
  })
}
pingHealth()
pullHosts()

function pingHealth() {
  fetch(HEALTH, { method: 'GET' }).catch(() => {})
}

async function pullHosts() {
  try {
    const resp = await fetch(HOSTS)
    if (!resp.ok) return
    const data = await resp.json()
    const hosts = Array.isArray(data && data.hosts) ? data.hosts : []
    if (!hosts.length) return
    const { extraHosts = [] } = await chrome.storage.sync.get({ extraHosts: [] })
    const merged = [...extraHosts]
    for (const h of hosts) {
      const v = String(h || '').trim()
      if (v && !merged.includes(v)) merged.push(v)
    }
    if (merged.length !== extraHosts.length) {
      await chrome.storage.sync.set({ extraHosts: merged })
    }
  } catch (_) {
    /* Faultline is not running */
  }
}

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
