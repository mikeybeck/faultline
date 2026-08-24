<script>
  import { onMount } from 'svelte'
  import Welcome from './Welcome.svelte'
  import Inbox from './Inbox.svelte'
  import {
    ApplyAndWatch,
    Bootstrap,
    CandidateFromPath,
    Clear,
    DefaultConfig,
    DetectSources,
    GetState,
    NewProject,
    OpenInEditor,
    PickLogFile,
    PickProjectDir,
    SaveAndWatch,
  } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'

  let screen = 'welcome'
  let projectDir = ''
  let sources = []
  let editorCommand = 'code'
  let customEditor = ''
  let notifications = true
  let sound = false
  let fromStart = false
  let error = ''
  let busy = false
  let settingsOpen = false

  let events = []
  let sourceStatuses = []
  let filter = ''
  let severity = 'all'
  let sort = 'recent'
  let selected = null
  let statusMsg = ''

  function cfgFromForm() {
    return {
      sources: sources.map((s) => ({ name: s.name, type: s.type, path: s.path })),
      notifications: { enabled: notifications, sound },
      editor: { command: editorCommand },
    }
  }

  function applyConfig(cfg) {
    if (!cfg) return
    sources = (cfg.sources || []).map((s) => ({ ...s }))
    notifications = !!(cfg.notifications && cfg.notifications.enabled)
    sound = !!(cfg.notifications && cfg.notifications.sound)
    editorCommand = (cfg.editor && cfg.editor.command) || 'code'
    if (!['code', 'cursor', 'phpstorm'].includes(editorCommand)) {
      customEditor = editorCommand
    }
  }

  async function refresh() {
    const st = await GetState(filter, sort, severity)
    events = st.events || []
    sourceStatuses = st.sources || []
    if (selected) {
      const next = events.find((e) => e.hash === selected.hash)
      selected = next || selected
    } else if (events.length) {
      selected = events[0]
    }
  }

  async function load() {
    const b = await Bootstrap()
    projectDir = b.projectDir || ''
    fromStart = !!b.fromStart
    if (b.config) applyConfig(b.config)
    else {
      const d = await DefaultConfig()
      applyConfig(d)
    }
    screen = b.screen || 'welcome'
    if (screen === 'inbox') await refresh()
  }

  onMount(() => {
    load()
    EventsOn('error:new', (ev) => {
      statusMsg = 'New: ' + (ev.title || ev.type)
      refresh()
    })
    EventsOn('status', () => refresh())
    const onKey = (e) => {
      if (e.key === '/' && screen === 'inbox' && document.activeElement.tagName !== 'INPUT') {
        e.preventDefault()
        const el = document.querySelector('.search')
        if (el) el.focus()
      }
      if (e.key === 'o' && screen === 'inbox' && selected && document.activeElement.tagName !== 'INPUT') {
        OpenInEditor(selected.hash)
      }
      if (e.key === 'c' && screen === 'inbox' && document.activeElement.tagName !== 'INPUT') {
        doClear()
      }
      if (e.key === 'Escape') settingsOpen = false
    }
    window.addEventListener('keydown', onKey)
    const tick = setInterval(() => {
      if (screen === 'inbox') refresh()
    }, 2000)
    return () => {
      window.removeEventListener('keydown', onKey)
      clearInterval(tick)
    }
  })

  async function openFolder() {
    error = ''
    const dir = await PickProjectDir()
    if (!dir) return
    projectDir = dir
    busy = true
    try {
      const found = await DetectSources(dir)
      if (found && found.length) sources = found
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

  async function startWatching() {
    error = ''
    busy = true
    try {
      await SaveAndWatch(projectDir, cfgFromForm(), fromStart)
      screen = 'inbox'
      settingsOpen = false
      await refresh()
    } catch (e) {
      error = String(e)
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
    } catch (e) {
      error = String(e)
    } finally {
      busy = false
    }
  }

  async function doClear() {
    await Clear()
    selected = null
    await refresh()
    statusMsg = 'Cleared'
  }

  async function doOpen(hash) {
    try {
      await OpenInEditor(hash)
      statusMsg = 'Opened in editor'
    } catch (e) {
      statusMsg = String(e)
    }
  }

  async function doCopy(ev) {
    const text = [ev.title, ev.location, ev.message, ev.stack].filter(Boolean).join('\n\n')
    try {
      await navigator.clipboard.writeText(text)
      statusMsg = 'Copied'
    } catch {
      statusMsg = 'Could not copy'
    }
  }

  async function startFresh() {
    await NewProject()
    screen = 'welcome'
    settingsOpen = false
    events = []
    selected = null
    projectDir = ''
    sources = []
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
      bind:notifications
      bind:sound
      bind:fromStart
      {error}
      {busy}
      onOpenFolder={openFolder}
      onAddFile={addFile}
      onRemove={removeSource}
      onStart={startWatching}
      {onSourceChange}
    />
  {:else}
    <Inbox
      {events}
      sources={sourceStatuses}
      {selected}
      {filter}
      {severity}
      {sort}
      {statusMsg}
      onSelect={(ev) => (selected = ev)}
      onOpen={doOpen}
      onClear={doClear}
      onCopy={doCopy}
      onSettings={() => (settingsOpen = true)}
      onFilter={(v) => { filter = v; refresh() }}
      onSeverity={(v) => { severity = v; refresh() }}
      onSort={(v) => { sort = v; refresh() }}
    />
  {/if}

  {#if settingsOpen}
    <div class="modal-bg" on:click={() => (settingsOpen = false)} on:keydown={() => {}} role="presentation">
      <div class="modal" tabindex="-1" on:click|stopPropagation on:keydown={() => {}} role="dialog">
        <h2 style="margin-top:0">Settings</h2>
        {#if error}
          <div class="banner">{error}</div>
        {/if}
        <Welcome
          {projectDir}
          {sources}
          {editorCommand}
          {customEditor}
          bind:notifications
          bind:sound
          bind:fromStart
          error=""
          {busy}
          onOpenFolder={openFolder}
          onAddFile={addFile}
          onRemove={removeSource}
          onStart={saveSettings}
          {onSourceChange}
          actionLabel="Save and restart"
        />
        <div class="row-actions">
          <button class="btn" on:click={startFresh}>New project</button>
          <button class="btn ghost" on:click={() => (settingsOpen = false)}>Close</button>
        </div>
      </div>
    </div>
  {/if}
</div>
