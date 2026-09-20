<script>
  import { onMount } from 'svelte'
  import Welcome from './Welcome.svelte'
  import Inbox from './Inbox.svelte'
  import Settings from './Settings.svelte'
  import {
    ApplyAndWatch,
    Bootstrap,
    CandidateFromPath,
    Clear,
    ClearMark,
    DefaultConfig,
    DetectSources,
    Dismiss,
    DismissMatching,
    GetEvent,
    GetState,
    Mute,
    Mutes,
    NewProject,
    OpenInEditor,
    OpenFaultlineLog,
    OpenPath,
    OpenRecent,
    PickEditor,
    PickLogFile,
    PickProjectDir,
    RecentProjects,
    ReportError,
    FaultlineLogPath,
    RevealExtension,
    SaveAndWatch,
    Snooze,
    SaveProjectConfig,
    Unmute,
  } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'

  let screen = 'welcome'
  let projectDir = ''
  let sources = []
  let editorCommand = 'code'
  let customEditor = ''
  let notifications = true
  let sound = false
  let fromStart = true
  let followLatest = false
  let clearOnCommit = false
  let extraHostsText = ''
  let ignoreText = ''
  let views = []
  let error = ''
  let busy = false
  let settingsOpen = false
  let helpOpen = false
  let scannedEmpty = false
  let recent = []
  let dismissMenus = 0

  let events = []
  let sourceStatuses = []
  let sourceCounts = {}
  let ingestAddr = ''
  let extDir = ''
  let extZipURL = ''
  let filter = ''
  let severity = 'all'
  let sourceFilter = ''
  let sort = 'recent'
  let selected = null
  let statusMsg = ''
  let total = 0
  let marked = false
  let mutes = []
  let logPath = ''
  let refreshTimer
  let refreshBusy = false
  let refreshQueued = false
  let toastTimer

  function toast(msg) {
    statusMsg = msg
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => {
      if (statusMsg === msg) statusMsg = ''
    }, 2800)
  }

  function reportClientError(context, err) {
    const detail = err && err.message ? err.message : String(err || 'unknown error')
    const msg = context ? context + ': ' + detail : detail
    toast(msg)
    const stack = (err && err.stack) || ''
    ReportError(msg, stack).catch(() => {})
  }

  function openSettings() {
    settingsOpen = true
    Promise.all([
      loadMutes(),
      loadRecent(),
      FaultlineLogPath().then((p) => { logPath = p || '' }),
    ]).catch((e) => reportClientError('Settings failed to load', e))
  }

  async function openFaultlineLog() {
    try {
      await OpenFaultlineLog()
    } catch (e) {
      reportClientError('Could not open Faultline log', e)
    }
  }

  function hostsFromText(text) {
    return String(text || '')
      .split(/\r?\n/)
      .map((s) => s.trim())
      .filter((s) => s && !s.startsWith('#'))
  }

  function ignoreFromText(text) {
    return String(text || '')
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter((s) => s && !s.startsWith('#'))
      .map((line) => {
        const i = line.indexOf(':')
        if (i < 0) return { message: line }
        const k = line.slice(0, i).trim().toLowerCase()
        const v = line.slice(i + 1).trim()
        if (k === 'type') return { type: v }
        if (k === 'message') return { message: v }
        if (k === 'path') return { path: v }
        if (k === 'source') return { source: v }
        if (k === 'regex') return { regex: v }
        return { message: line }
      })
  }

  function ignoreToText(rules) {
    return (rules || [])
      .map((r) => {
        if (r.type) return 'type: ' + r.type
        if (r.message) return 'message: ' + r.message
        if (r.path) return 'path: ' + r.path
        if (r.source) return 'source: ' + r.source
        if (r.regex) return 'regex: ' + r.regex
        return ''
      })
      .filter(Boolean)
      .join('\n')
  }

  function cfgFromForm() {
    return {
      sources: sources.map((s) => ({ name: s.name, type: s.type, path: s.path, parser: s.parser || '' })),
      notifications: { enabled: notifications, sound },
      editor: { command: editorCommand },
      inbox: {
        followLatest,
        clearOnCommit,
        ignore: ignoreFromText(ignoreText),
        views: (views || []).map((v) => ({ ...v })),
      },
      browser: { extraHosts: hostsFromText(extraHostsText) },
    }
  }

  function applyConfig(cfg) {
    if (!cfg) return
    sources = (cfg.sources || []).map((s) => ({ ...s }))
    notifications = !!(cfg.notifications && cfg.notifications.enabled)
    sound = !!(cfg.notifications && cfg.notifications.sound)
    editorCommand = (cfg.editor && cfg.editor.command) || 'code'
    followLatest = !!(cfg.inbox && cfg.inbox.followLatest)
    clearOnCommit = !!(cfg.inbox && cfg.inbox.clearOnCommit)
    extraHostsText = ((cfg.browser && cfg.browser.extraHosts) || []).join('\n')
    ignoreText = ignoreToText(cfg.inbox && cfg.inbox.ignore)
    views = ((cfg.inbox && cfg.inbox.views) || []).map((v) => ({ ...v }))
    if (!['code', 'cursor', 'phpstorm'].includes(editorCommand)) {
      customEditor = editorCommand
    }
  }

  function newestEvent(list) {
    if (!list || !list.length) return null
    return list.reduce((best, ev) => {
      if (!best) return ev
      return (ev.lastSeen || '') > (best.lastSeen || '') ? ev : best
    }, null)
  }

  function mergeEvent(detail, summary) {
    return {
      ...detail,
      ...summary,
      stack: detail.stack,
      raw: detail.raw,
      message: detail.message || summary.message,
      frames: detail.frames,
      context: detail.context,
      snippet: detail.snippet,
      samples: detail.samples,
    }
  }

  async function refresh() {
    try {
      const st = await GetState(filter, sort, severity, sourceFilter)
      events = st.events || []
      total = st.total || events.length
      marked = !!st.marked
      sourceStatuses = st.sources || []
      sourceCounts = st.sourceCounts || {}
      ingestAddr = st.ingestAddr || ''
      if (followLatest && events.length) {
        const latest = newestEvent(events)
        if (latest && (!selected || selected.hash !== latest.hash)) {
          await selectEvent(latest)
        } else if (latest && selected) {
          selected = mergeEvent(selected, latest)
        }
      } else if (selected) {
        const next = events.find((e) => e.hash === selected.hash)
        if (next) {
          selected = mergeEvent(selected, next)
        } else {
          selected = null
        }
      } else if (events.length) {
        await selectEvent(events[0])
      }
    } catch (e) {
      reportClientError('Refresh failed', e)
    }
  }

  function scheduleRefresh() {
    clearTimeout(refreshTimer)
    refreshTimer = setTimeout(runRefresh, 200)
  }

  async function runRefresh() {
    if (refreshBusy) {
      refreshQueued = true
      return
    }
    refreshBusy = true
    try {
      await refresh()
    } finally {
      refreshBusy = false
      if (refreshQueued) {
        refreshQueued = false
        scheduleRefresh()
      }
    }
  }

  async function selectEvent(ev) {
    selected = ev
    try {
      selected = await GetEvent(ev.hash)
    } catch (e) {
      toast(String(e))
    }
  }

  async function selectFromKeyboard(ev) {
    if (followLatest) {
      followLatest = false
      if (projectDir) {
        try {
          await SaveProjectConfig(cfgFromForm())
        } catch (e) {
          toast(String(e))
        }
      }
    }
    await selectEvent(ev)
  }

  async function loadRecent() {
    try {
      recent = (await RecentProjects()) || []
    } catch {
      recent = []
    }
  }

  async function load() {
    try {
      const b = await Bootstrap()
      projectDir = b.projectDir || ''
      fromStart = b.fromStart !== false
      recent = b.recent || []
      ingestAddr = b.ingestAddr || ''
      extDir = b.extDir || ''
      extZipURL = b.extZipURL || ''
      if (b.config) applyConfig(b.config)
      else {
        const d = await DefaultConfig()
        applyConfig(d)
      }
      screen = b.screen || 'welcome'
      if (screen === 'inbox') await refresh()
    } catch (e) {
      reportClientError('Startup failed', e)
    }
  }

  function typingTarget(el) {
    if (!el) return false
    const tag = el.tagName
    return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable
  }

  function moveSelection(delta) {
    if (!events.length) return
    const i = selected ? events.findIndex((e) => e.hash === selected.hash) : -1
    let next = i + delta
    if (i < 0) next = delta > 0 ? 0 : events.length - 1
    if (next < 0) next = 0
    if (next >= events.length) next = events.length - 1
    const ev = events[next]
    if (ev) selectFromKeyboard(ev)
  }

  onMount(() => {
    load()
    EventsOn('inbox:changed', scheduleRefresh)
    EventsOn('status', scheduleRefresh)
    EventsOn('inbox:focus', async (hash) => {
      if (!hash) return
      try {
        const ev = await GetEvent(hash)
        selected = ev
        await refresh()
      } catch (e) {
        reportClientError('Could not open notified error', e)
      }
    })
    const onWindowError = (e) => {
      ReportError(e.message || 'window error', (e.error && e.error.stack) || '').catch(() => {})
    }
    const onRejection = (e) => {
      const r = e.reason
      ReportError(String(r || 'unhandled rejection'), (r && r.stack) || '').catch(() => {})
    }
    window.addEventListener('error', onWindowError)
    window.addEventListener('unhandledrejection', onRejection)
    const onKey = (e) => {
      if (e.key === 'Escape') {
        if (helpOpen) {
          helpOpen = false
          return
        }
        dismissMenus += 1
        if (settingsOpen) {
          settingsOpen = false
          return
        }
        const active = document.activeElement
        if (active && active.classList && active.classList.contains('search')) {
          if (filter) {
            filter = ''
            scheduleRefresh()
          }
          active.blur()
          return
        }
        if (active && active.classList && active.classList.contains('detail')) {
          active.blur()
          return
        }
        return
      }
      if (e.key === '?' && !typingTarget(e.target)) {
        e.preventDefault()
        if (screen === 'inbox') helpOpen = !helpOpen
        return
      }
      if (settingsOpen || helpOpen || screen !== 'inbox') return
      if (typingTarget(e.target)) {
        if (e.key === '/' && e.target.classList && e.target.classList.contains('search') && e.target.value === '') {
          return
        }
        return
      }
      if (e.key === '/') {
        e.preventDefault()
        const el = document.querySelector('.search')
        if (el) el.focus()
        return
      }
      if (e.key === 'j' || e.key === 'ArrowDown') {
        e.preventDefault()
        moveSelection(1)
        return
      }
      if (e.key === 'k' || e.key === 'ArrowUp') {
        e.preventDefault()
        moveSelection(-1)
        return
      }
      if (e.key === 'Enter') {
        e.preventDefault()
        const el = document.querySelector('.detail')
        if (el) el.focus()
        return
      }
      if (e.key === 'o' && selected) {
        OpenInEditor(selected.hash).then(() => toast('Opened in editor')).catch((err) => toast(String(err)))
        return
      }
      if (e.key === 'd' && selected) {
        doDismiss(selected.hash)
        return
      }
      if (e.key === 'z' && selected) {
        doSnooze(selected.hash, 15)
        return
      }
      if (e.key === 'c') {
        doClear()
        return
      }
      if (e.key === 's') {
        e.preventDefault()
        sort = sort === 'recent' ? 'frequency' : 'recent'
        scheduleRefresh()
      }
    }
    window.addEventListener('keydown', onKey)
    const tick = setInterval(() => {
      if (screen === 'inbox') scheduleRefresh()
    }, 2000)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('error', onWindowError)
      window.removeEventListener('unhandledrejection', onRejection)
      clearInterval(tick)
      clearTimeout(toastTimer)
    }
  })

  async function openFolder() {
    error = ''
    scannedEmpty = false
    const dir = await PickProjectDir()
    if (!dir) return
    projectDir = dir
    busy = true
    try {
      const found = await DetectSources(dir)
      if (found && found.length) {
        sources = found
        scannedEmpty = false
      } else {
        scannedEmpty = true
      }
    } catch (e) {
      error = String(e)
    } finally {
      busy = false
    }
  }

  async function addFile() {
    error = ''
    const path = await PickLogFile()
    if (!path) return
    const c = await CandidateFromPath(path)
    if (!projectDir) {
      projectDir = c.path.replace(/[/\\][^/\\]+$/, '')
    }
    sources = [...sources, c]
    scannedEmpty = false
  }

  function addBrowser() {
    error = ''
    if (sources.some((s) => s.type === 'browser')) {
      toast('Browser source already added')
      return
    }
    sources = [...sources, { name: 'browser', type: 'browser', path: '127.0.0.1:9477' }]
    scannedEmpty = false
  }

  function uniqueName(base) {
    const names = new Set(sources.map((s) => s.name))
    if (!names.has(base)) return base
    let i = 2
    while (names.has(base + '-' + i)) i++
    return base + '-' + i
  }

  function addCommand() {
    error = ''
    sources = [...sources, { name: uniqueName('dev'), type: 'command', path: 'npm run dev' }]
    scannedEmpty = false
  }

  function removeSource(i) {
    sources = sources.filter((_, idx) => idx !== i)
  }

  function onSourceChange(i, patch) {
    if (i < 0) {
      if (patch.custom) customEditor = patch.editor
      if (patch.editor !== undefined) editorCommand = patch.editor
      return
    }
    sources = sources.map((s, idx) => (idx === i ? { ...s, ...patch } : s))
  }

  async function browseEditor() {
    error = ''
    const path = await PickEditor()
    if (!path) return
    customEditor = path
    editorCommand = path
  }

  async function applyView(v) {
    if (!v) return
    filter = v.filter || ''
    severity = v.severity || 'all'
    sourceFilter = v.source || ''
    sort = v.sort || 'recent'
    scheduleRefresh()
  }

  async function saveCurrentView() {
    const name = window.prompt('Name this view')
    if (!name || !name.trim()) return
    const next = {
      name: name.trim(),
      filter,
      severity,
      source: sourceFilter,
      sort,
    }
    views = [...(views || []).filter((v) => v.name !== next.name), next]
    if (!projectDir) return
    try {
      await SaveProjectConfig(cfgFromForm())
      toast('Saved view')
    } catch (e) {
      toast(String(e))
    }
  }

  async function deleteView(name) {
    views = (views || []).filter((v) => v.name !== name)
    if (!projectDir) return
    try {
      await SaveProjectConfig(cfgFromForm())
      toast('Removed view')
    } catch (e) {
      toast(String(e))
    }
  }

  async function setFollowLatest(v) {
    followLatest = !!v
    if (followLatest && events.length) {
      const latest = newestEvent(events)
      if (latest) await selectEvent(latest)
    }
    if (!projectDir) return
    try {
      await SaveProjectConfig(cfgFromForm())
    } catch (e) {
      toast(String(e))
    }
  }

  async function startWatching() {
    error = ''
    busy = true
    try {
      await SaveAndWatch(projectDir, cfgFromForm(), fromStart)
      screen = 'inbox'
      settingsOpen = false
      scannedEmpty = false
      await refresh()
      await loadRecent()
    } catch (e) {
      error = String(e)
      reportClientError('Could not start watching', e)
    } finally {
      busy = false
    }
  }

  async function saveSettings() {
    error = ''
    busy = true
    try {
      await ApplyAndWatch(cfgFromForm(), fromStart)
      settingsOpen = false
      await refresh()
      await loadRecent()
    } catch (e) {
      error = String(e)
      reportClientError('Could not save settings', e)
    } finally {
      busy = false
    }
  }

  async function openRecentProject(configPath) {
    error = ''
    busy = true
    try {
      await OpenRecent(configPath)
      const b = await Bootstrap()
      projectDir = b.projectDir || ''
      if (b.config) applyConfig(b.config)
      screen = 'inbox'
      settingsOpen = false
      await refresh()
      await loadRecent()
    } catch (e) {
      error = String(e)
      screen = 'welcome'
    } finally {
      busy = false
    }
  }

  async function doClear() {
    if (!confirm('Mark inbox clean? Current errors are hidden and these log lines are skipped next time.')) return
    await Clear()
    selected = null
    await refresh()
    toast('Inbox marked clean — skipped on next start')
  }

  async function doClearMark() {
    await ClearMark()
    selected = null
    await refresh()
    toast('Replaying logs from the start')
  }

  async function doDismiss(hash) {
    if (!hash) return
    await Dismiss([hash])
    selected = null
    await refresh()
    toast('Dismissed until it happens again')
  }

  async function doDismissMatching() {
    await DismissMatching(filter, severity, sourceFilter, '')
    selected = null
    await refresh()
    toast('Dismissed matching')
  }

  async function doSnooze(hash, minutes) {
    try {
      await Snooze(hash, minutes)
      selected = null
      await refresh()
      toast('Snoozed ' + minutes + 'm')
    } catch (e) {
      reportClientError('Snooze failed', e)
    }
  }

  async function revealExtension() {
    try {
      await RevealExtension()
      toast('Opened extension folder')
    } catch (e) {
      reportClientError('Could not open extension folder', e)
    }
  }

  async function copyExtensionPath() {
    if (!extDir) {
      toast('Extension path not ready')
      return
    }
    try {
      await navigator.clipboard.writeText(extDir)
      toast('Copied extension path')
    } catch (e) {
      toast(extDir)
    }
  }

  async function doMute(kind, value) {
    try {
      await Mute(kind, value)
      selected = null
      await refresh()
      await loadMutes()
      toast(kind === 'type' ? 'Muted type' : 'Muted')
    } catch (e) {
      toast(String(e))
    }
  }

  async function loadMutes() {
    try {
      mutes = (await Mutes()) || []
    } catch {
      mutes = []
    }
  }

  async function doUnmute(kind, value) {
    try {
      await Unmute(kind, value)
      await refresh()
      await loadMutes()
      toast('Unmuted')
    } catch (e) {
      toast(String(e))
    }
  }

  async function doOpen(hash) {
    try {
      await OpenInEditor(hash)
      toast('Opened in editor')
    } catch (e) {
      toast(String(e))
    }
  }

  async function doOpenPath(file, line) {
    try {
      await OpenPath(file, line || 0)
      toast('Opened in editor')
    } catch (e) {
      toast(String(e))
    }
  }

  async function copyText(text, okMsg) {
    try {
      await navigator.clipboard.writeText(text)
      toast(okMsg)
    } catch {
      toast('Could not copy')
    }
  }

  async function doCopy(ev) {
    const text = [ev.title, ev.location, ev.message, ev.stack].filter(Boolean).join('\n\n')
    await copyText(text, 'Copied')
  }

  async function doCopyRaw(ev) {
    const text = (ev && ev.raw) || ''
    if (!text) {
      toast('No raw log')
      return
    }
    await copyText(text, 'Copied raw')
  }

  async function doCopyMarkdown(ev) {
    const lines = [`**${ev.title || 'Error'}**`]
    if (ev.location) lines.push('`' + ev.location + '`')
    lines.push(`${ev.severity || 'error'} · ${ev.source || ''} · ×${ev.count || 1}`)
    if (ev.message) lines.push('', ev.message)
    if (ev.stack) lines.push('', '```', ev.stack, '```')
    await copyText(lines.join('\n'), 'Copied markdown')
  }

  async function startFresh() {
    await NewProject()
    screen = 'welcome'
    settingsOpen = false
    events = []
    selected = null
    projectDir = ''
    sources = []
    sourceFilter = ''
    mutes = []
    followLatest = false
    extraHostsText = ''
    ignoreText = ''
    views = []
    scannedEmpty = false
    await loadRecent()
  }
</script>

<div class="app">
  <div class="fault-bar"></div>
  {#if screen === 'welcome'}
    <Welcome
      {projectDir}
      {sources}
      {editorCommand}
      {customEditor}
      {recent}
      {scannedEmpty}
      bind:notifications
      bind:sound
      bind:fromStart
      bind:followLatest
      bind:clearOnCommit
      {error}
      {busy}
      onOpenFolder={openFolder}
      onAddFile={addFile}
      onAddBrowser={addBrowser}
      onAddCommand={addCommand}
      onRemove={removeSource}
      onStart={startWatching}
      {onSourceChange}
      onBrowseEditor={browseEditor}
      onOpenRecent={openRecentProject}
      {extDir}
      {extZipURL}
      onRevealExtension={revealExtension}
      onCopyExtensionPath={copyExtensionPath}
    />
  {:else}
    <Inbox
      {events}
      {total}
      sources={sourceStatuses}
      {sourceCounts}
      {selected}
      {filter}
      {severity}
      {sourceFilter}
      {sort}
      {followLatest}
      {views}
      {statusMsg}
      marked={marked}
      {helpOpen}
      {ingestAddr}
      {dismissMenus}
      onSelect={selectEvent}
      onOpen={doOpen}
      onOpenPath={doOpenPath}
      onClear={doClear}
      onClearMark={doClearMark}
      onDismiss={doDismiss}
      onDismissMatching={doDismissMatching}
      onMute={doMute}
      onSnooze={doSnooze}
      onCopy={doCopy}
      onCopyRaw={doCopyRaw}
      onCopyMarkdown={doCopyMarkdown}
      onSettings={openSettings}
      onFilter={(v) => { filter = v; scheduleRefresh() }}
      onSeverity={(v) => { severity = v; scheduleRefresh() }}
      onSourceFilter={(v) => { sourceFilter = v; scheduleRefresh() }}
      onSort={(v) => { sort = v; scheduleRefresh() }}
      onFollowLatest={setFollowLatest}
      onApplyView={applyView}
      onSaveView={saveCurrentView}
      onDeleteView={deleteView}
      onHelp={() => (helpOpen = !helpOpen)}
    />
  {/if}

  {#if settingsOpen}
    <div
      class="modal-bg"
      on:mousedown|self={() => (settingsOpen = false)}
      on:keydown={(e) => e.key === 'Escape' && (settingsOpen = false)}
      role="presentation"
    >
      <div class="modal" tabindex="-1" on:mousedown|stopPropagation role="dialog" aria-labelledby="settings-title">
        <h2 id="settings-title" style="margin-top:0">Settings</h2>
        <Settings
          {projectDir}
          {sources}
          {editorCommand}
          {customEditor}
          {extraHostsText}
          {ignoreText}
          {mutes}
          {recent}
          {logPath}
          bind:notifications
          bind:sound
          bind:fromStart
          bind:followLatest
          bind:clearOnCommit
          {extDir}
          {extZipURL}
          {error}
          {busy}
          onOpenFolder={openFolder}
          onAddFile={addFile}
          onAddBrowser={addBrowser}
          onAddCommand={addCommand}
          onRemove={removeSource}
          onStart={saveSettings}
          {onSourceChange}
          onBrowseEditor={browseEditor}
          onExtraHosts={(v) => (extraHostsText = v)}
          onIgnore={(v) => (ignoreText = v)}
          onUnmute={doUnmute}
          onOpenRecent={openRecentProject}
          onNewProject={startFresh}
          onOpenLog={openFaultlineLog}
          onRevealExtension={revealExtension}
          onCopyExtensionPath={copyExtensionPath}
          onClose={() => (settingsOpen = false)}
        />
      </div>
    </div>
  {/if}
</div>
