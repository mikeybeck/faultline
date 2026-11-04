<script>
  import { onMount } from 'svelte'
  export let events = []
  export let total = 0
  export let sources = []
  export let sourceCounts = {}
  export let selected
  export let filter = ''
  export let severity = 'all'
  export let sourceFilter = ''
  export let sort = 'recent'
  export let followLatest = false
  export let views = []
  export let statusMsg = ''
  export let toastActionLabel = ''
  export let marked = false
  export let helpOpen = false
  export let ingestAddr = ''
  export let dismissMenus = 0
  export let version = ''
  export let onSelect
  export let onOpen
  export let onOpenPath = () => {}
  export let onClear
  export let onClearMark
  export let onDismiss
  export let onDismissMatching
  export let onMute
  export let onSnooze = () => {}
  export let onCopy
  export let onCopyRaw = () => {}
  export let onCopyMarkdown = () => {}
  export let onSettings
  export let onFilter
  export let onSeverity
  export let onSourceFilter = () => {}
  export let onSort
  export let onFollowLatest = () => {}
  export let onApplyView = () => {}
  export let onSaveView = () => {}
  export let onDeleteView = () => {}
  export let onToastAction = () => {}
  export let onHelp = () => {}

  const ROW = 76
  const OVERSCAN = 8
  let listEl
  let scrollTop = 0
  let viewH = 480
  let muteOpen = false
  let copyOpen = false
  let snoozeOpen = false
  let viewsOpen = false
  let savingView = false
  let managingViews = false
  let viewNameDraft = ''
  let now = Date.now()

  $: start = Math.max(0, Math.floor(scrollTop / ROW) - OVERSCAN)
  $: visibleCount = Math.ceil(viewH / ROW) + OVERSCAN * 2
  $: end = Math.min(events.length, start + visibleCount)
  $: visible = events.slice(start, end)
  $: padTop = start * ROW
  $: padBot = Math.max(0, (events.length - end) * ROW)
  $: narrowed = !!(filter && filter.trim()) || (severity && severity !== 'all') || !!sourceFilter
  $: browserOnly = sources.length > 0 && sources.every((s) => s.type === 'browser')
  $: commandOnly = sources.length > 0 && sources.every((s) => s.type === 'command')
  $: if (!selected) { muteOpen = false; copyOpen = false; snoozeOpen = false }
  $: if (dismissMenus) {
    muteOpen = false
    copyOpen = false
    snoozeOpen = false
    viewsOpen = false
    savingView = false
    managingViews = false
  }

  function resetViewDraft() {
    savingView = false
    managingViews = false
    viewNameDraft = ''
  }

  function commitSaveView() {
    const name = (viewNameDraft || '').trim()
    if (!name) return
    onSaveView(name)
    resetViewDraft()
    viewsOpen = false
  }
  $: samples = ((selected && selected.samples) || []).filter((s) => s && s !== (selected && selected.message))
  $: contextLines = (selected && selected.context) || []
  $: frames = (selected && selected.frames) || []
  $: if (selected && listEl) scrollSelected()

  function scrollSelected() {
    if (!selected || !listEl) return
    const i = events.findIndex((e) => e.hash === selected.hash)
    if (i < 0) return
    const top = i * ROW
    if (top < listEl.scrollTop) listEl.scrollTop = top
    else if (top + ROW > listEl.scrollTop + viewH) listEl.scrollTop = top + ROW - viewH
  }

  onMount(() => {
    const el = listEl
    if (!el) return
    const apply = () => {
      viewH = el.clientHeight || 480
    }
    apply()
    const ro = new ResizeObserver(apply)
    ro.observe(el)
    const close = (e) => {
      if (!e.target.closest('.menu')) {
        muteOpen = false
        copyOpen = false
        snoozeOpen = false
        viewsOpen = false
        resetViewDraft()
      }
    }
    document.addEventListener('mousedown', close)
    const tick = setInterval(() => {
      now = Date.now()
    }, 15000)
    return () => {
      ro.disconnect()
      document.removeEventListener('mousedown', close)
      clearInterval(tick)
    }
  })

  function onScroll(e) {
    scrollTop = e.target.scrollTop
  }

  function rel(iso, clock) {
    if (!iso) return 'unknown'
    const t = new Date(iso).getTime()
    const d = (clock || now) - t
    if (d < 1000) return 'just now'
    if (d < 60000) return Math.floor(d / 1000) + 's ago'
    if (d < 3600000) return Math.floor(d / 60000) + 'm ago'
    if (d < 86400000) return Math.floor(d / 3600000) + 'h ago'
    return new Date(iso).toLocaleString()
  }

  function pillClass(state) {
    if (state === 'ok') return 'ok'
    if (state === 'waiting' || state === 'ingesting') return 'wait'
    if (state === 'error') return 'error'
    return ''
  }

  function toggleSource(name) {
    onSourceFilter(sourceFilter === name ? '' : name)
  }

  function emptyTitle() {
    if (narrowed) return 'Nothing matches'
    return 'No errors yet'
  }

  function emptyBody() {
    if (narrowed) return 'Try clearing the search, severity, or source filter.'
    if (browserOnly) {
      const addr = ingestAddr || '127.0.0.1:9477'
      return 'Waiting for the extension. Load it unpacked from the extension/ folder, then open your app on localhost. Listening on ' + addr + '.'
    }
    if (commandOnly) return 'Waiting for the command to print an error. Info and debug lines are skipped.'
    return 'Waiting for log activity. If your app prints to the terminal, add a command source in Settings (for example npm run dev).'
  }

  function sourceCount(name) {
    const n = sourceCounts && sourceCounts[name]
    return n ? n : 0
  }

  function frameLabel(fr) {
    if (fr.file) return fr.file + (fr.line ? ':' + fr.line : '')
    return fr.text
  }
</script>

<div class="top">
  <div class="brand">
    <h1>Faultline</h1>
    {#if version}<span class="ver">{version}</span>{/if}
    <span>inbox{#if total} · {total}{/if}</span>
  </div>
  <div class="pills">
    {#each sources as s}
      <button
        type="button"
        class="pill {pillClass(s.state)}"
        class:on={sourceFilter === s.name}
        aria-pressed={sourceFilter === s.name}
        title={(s.message || s.path || s.name) + (s.state ? ` · ${s.state}` : '')}
        on:click={() => toggleSource(s.name)}
      >
        <span class="dot" aria-hidden="true"></span>
        {s.name}{#if sourceCount(s.name)} · {sourceCount(s.name)}{/if}{#if s.state === 'ingesting' && s.message}<span class="pill-msg"> · {s.message}</span>{/if}
      </button>
    {/each}
    {#if sources.length === 0}
      <div class="pill">No sources</div>
    {/if}
  </div>
  <button type="button" class="btn ghost" title="Keyboard shortcuts" on:click={onHelp}>?</button>
  <button type="button" class="btn" on:click|stopPropagation={onSettings}>Settings</button>
</div>

<div class="toolbar">
  <div class="toolbar-group toolbar-filters">
    <input
      class="search"
      placeholder="Filter type, message, file…"
      value={filter}
      on:input={(e) => onFilter(e.target.value)}
      aria-label="Filter errors"
    />
    <div class="chips" role="group" aria-label="Severity">
      {#each ['all', 'critical', 'error', 'warning'] as s}
        <button
          type="button"
          class="chip"
          class:on={severity === s}
          aria-pressed={severity === s}
          on:click={() => onSeverity(s)}
        >{s}</button>
      {/each}
    </div>
  </div>
  <div class="toolbar-group">
    <select class="select" value={sort} aria-label="Sort" on:change={(e) => onSort(e.target.value)}>
      <option value="recent">Recent</option>
      <option value="frequency">Frequency</option>
    </select>
    <button
      type="button"
      class="chip"
      class:on={followLatest}
      aria-pressed={followLatest}
      title="Keep the newest error selected as it arrives (session preference also in Settings)"
      on:click={() => onFollowLatest(!followLatest)}
    >Follow latest</button>
  </div>
  <div class="toolbar-group toolbar-end">
    <div class="menu">
      <button
        type="button"
        class="btn"
        aria-expanded={viewsOpen}
        aria-haspopup="menu"
        on:click={() => {
          viewsOpen = !viewsOpen
          if (!viewsOpen) resetViewDraft()
          muteOpen = false
          copyOpen = false
          snoozeOpen = false
        }}
      >Views</button>
      {#if viewsOpen}
        <div class="menu-pop" role="menu">
          {#if !managingViews}
            {#each views as v}
              <button type="button" class="menu-item" role="menuitem" on:click={() => { viewsOpen = false; resetViewDraft(); onApplyView(v) }}>{v.name}</button>
            {/each}
            {#if !views.length && !savingView}
              <div class="menu-empty">No saved views</div>
            {/if}
            {#if savingView}
              <div class="menu-save">
                <input
                  class="path-input"
                  placeholder="View name"
                  bind:value={viewNameDraft}
                  on:keydown={(e) => e.key === 'Enter' && commitSaveView()}
                />
                <button type="button" class="btn primary sm" on:click={commitSaveView}>Save</button>
              </div>
            {:else}
              <button type="button" class="menu-item" role="menuitem" on:click={() => { savingView = true; managingViews = false; viewNameDraft = '' }}>Save current filters…</button>
            {/if}
            {#if views.length}
              <button type="button" class="menu-item" role="menuitem" on:click={() => { managingViews = true; savingView = false }}>Delete a view…</button>
            {/if}
          {:else}
            <div class="menu-empty">Choose a view to delete</div>
            {#each views as v}
              <button
                type="button"
                class="menu-item danger-item"
                role="menuitem"
                on:click={() => {
                  if (!confirm(`Delete saved view “${v.name}”?`)) return
                  onDeleteView(v.name)
                }}
              >Delete “{v.name}”</button>
            {/each}
            <button type="button" class="menu-item" role="menuitem" on:click={() => (managingViews = false)}>Back</button>
          {/if}
        </div>
      {/if}
    </div>
    {#if narrowed}
      <button type="button" class="btn" title="Hide the errors currently listed until they happen again" on:click={onDismissMatching}>
        Dismiss matching{#if total} · {total}{/if}
      </button>
    {/if}
    <button type="button" class="btn danger" title="Mark inbox clean — hide current errors and skip these log lines next time" on:click={onClear}>Mark clean</button>
    {#if marked}
      <button type="button" class="btn" title="Replay log files from the beginning. Browser errors are kept." on:click={onClearMark}>Replay logs</button>
    {/if}
  </div>
</div>

<div class="body">
  <div class="list" bind:this={listEl} on:scroll={onScroll}>
    {#if events.length === 0}
      <div class="empty">
        <div>
          <h2>{emptyTitle()}</h2>
          <p>{emptyBody()}</p>
        </div>
      </div>
    {:else}
      <div style="height:{padTop}px"></div>
      {#each visible as ev (ev.hash)}
        <div class="row" class:sel={selected && selected.hash === ev.hash} on:click={() => onSelect(ev)} on:keydown={(e) => e.key === 'Enter' && onSelect(ev)} role="button" tabindex="0">
          <div class="row-top">
            <div class="row-title">{ev.title}</div>
            {#if ev.count > 1}<span class="count">×{ev.count}</span>{/if}
          </div>
          <div class="meta">{ev.location || 'no location'} · {ev.source}</div>
          <div class="meta"><span class="sev {ev.severity}">{ev.severity}</span> · {rel(ev.lastSeen, now)}</div>
        </div>
      {/each}
      <div style="height:{padBot}px"></div>
    {/if}
  </div>
  <div class="detail" tabindex="-1">
    {#if !selected}
      <div class="empty">
        <div>
          <h2>Select an error</h2>
          <p>Full message and stack appear here. Press <kbd>?</kbd> for shortcuts.</p>
        </div>
      </div>
    {:else}
      <h2>{selected.title}</h2>
      <div class="meta">
        <span class="sev {selected.severity}">{selected.severity}</span>
        · {selected.source}
        · occurred ×{selected.count}
      </div>
      {#if selected.location}
        <div class="meta" style="margin-top:8px">
          <button class="loc" on:click={() => onOpen(selected.hash)}>{selected.location}</button>
        </div>
      {/if}
      <div class="detail-actions">
        <button
          type="button"
          class="btn primary"
          disabled={!selected.file}
          title={selected.file ? 'Open at file:line' : 'No file path on this event — try the location link if shown'}
          on:click={() => onOpen(selected.hash)}
        >Open in editor</button>
        <div class="menu">
          <button
            type="button"
            class="btn"
            aria-expanded={copyOpen}
            aria-haspopup="menu"
            on:click={() => { copyOpen = !copyOpen; muteOpen = false; snoozeOpen = false; viewsOpen = false }}
          >Copy</button>
          {#if copyOpen}
            <div class="menu-pop" role="menu">
              <button type="button" class="menu-item" role="menuitem" on:click={() => { copyOpen = false; onCopy(selected) }}>Copy message</button>
              <button type="button" class="menu-item" role="menuitem" on:click={() => { copyOpen = false; onCopyMarkdown(selected) }}>Copy markdown</button>
              <button type="button" class="menu-item" role="menuitem" disabled={!selected.raw} title={selected.raw ? 'May include request bodies or secrets' : ''} on:click={() => { copyOpen = false; onCopyRaw(selected) }}>Copy raw</button>
            </div>
          {/if}
        </div>
        <button type="button" class="btn" title="Hide until this error happens again" on:click={() => onDismiss(selected.hash)}>Dismiss</button>
        <div class="menu">
          <button
            type="button"
            class="btn"
            aria-expanded={snoozeOpen}
            aria-haspopup="menu"
            title="Hide for a while even if it repeats"
            on:click={() => { snoozeOpen = !snoozeOpen; muteOpen = false; copyOpen = false; viewsOpen = false }}
          >Snooze</button>
          {#if snoozeOpen}
            <div class="menu-pop" role="menu">
              <button type="button" class="menu-item" role="menuitem" on:click={() => { snoozeOpen = false; onSnooze(selected.hash, 15) }}>15 minutes</button>
              <button type="button" class="menu-item" role="menuitem" on:click={() => { snoozeOpen = false; onSnooze(selected.hash, 60) }}>1 hour</button>
            </div>
          {/if}
        </div>
        <div class="menu">
          <button
            type="button"
            class="btn"
            aria-expanded={muteOpen}
            aria-haspopup="menu"
            on:click={() => { muteOpen = !muteOpen; copyOpen = false; snoozeOpen = false; viewsOpen = false }}
          >Mute</button>
          {#if muteOpen}
            <div class="menu-pop" role="menu">
              <button type="button" class="menu-item" role="menuitem" on:click={() => { muteOpen = false; onMute('hash', selected.hash) }}>Never show this error</button>
              {#if selected.type}
                <button type="button" class="menu-item" role="menuitem" on:click={() => { muteOpen = false; onMute('type', selected.type) }}>Never show {selected.type}</button>
              {/if}
            </div>
          {/if}
        </div>
      </div>
      <div class="block">
        <h3>Message</h3>
        <div class="message">{selected.message || '—'}</div>
      </div>
      {#if samples.length}
        <div class="block">
          <h3>Also seen</h3>
          <div class="stack">
            {#each samples as s}
              <div>{s}</div>
            {/each}
          </div>
        </div>
      {/if}
      {#if selected.snippet}
        <div class="block">
          <h3>Source</h3>
          <div class="stack">{selected.snippet}</div>
        </div>
      {/if}
      {#if contextLines.length}
        <div class="block">
          <h3>Nearby log</h3>
          <div class="stack">{contextLines.join('\n')}</div>
        </div>
      {/if}
      {#if frames.length}
        <div class="block">
          <h3>Stack</h3>
          <div class="stack stack-frames">
            {#each frames as fr}
              {#if fr.file}
                <div class="frame">
                  <button class="loc" type="button" on:click={() => onOpenPath(fr.file, fr.line)}>{frameLabel(fr)}</button>
                  {#if fr.text && fr.text !== frameLabel(fr)}
                    <span class="frame-text">{fr.text}</span>
                  {/if}
                </div>
              {:else}
                <div class="frame-text">{fr.text}</div>
              {/if}
            {/each}
          </div>
        </div>
      {:else if selected.stack}
        <div class="block">
          <h3>Stack</h3>
          <div class="stack">{selected.stack}</div>
        </div>
      {/if}
      {#if selected.raw}
        <div class="block">
          <h3>Raw</h3>
          <div class="stack">{selected.raw}</div>
        </div>
      {/if}
      <div class="meta">First {rel(selected.firstSeen, now)} · Last {rel(selected.lastSeen, now)}</div>
    {/if}
  </div>
</div>

{#if helpOpen}
  <div class="modal-bg" on:mousedown|self={onHelp} on:keydown={(e) => e.key === 'Escape' && onHelp()} role="presentation">
    <div class="modal help-modal" role="dialog" aria-labelledby="help-title" tabindex="-1" on:mousedown|stopPropagation>
      <h2 id="help-title">Shortcuts</h2>
      <table class="keys">
        <tbody>
          <tr><td><kbd>/</kbd></td><td>Search</td></tr>
          <tr><td><kbd>j</kbd> <kbd>k</kbd> or arrows</td><td>Move selection</td></tr>
          <tr><td><kbd>Enter</kbd></td><td>Focus detail</td></tr>
          <tr><td><kbd>o</kbd></td><td>Open in editor</td></tr>
          <tr><td><kbd>d</kbd></td><td>Dismiss until it happens again</td></tr>
          <tr><td><kbd>z</kbd></td><td>Snooze 15 minutes</td></tr>
          <tr><td><kbd>c</kbd></td><td>Mark inbox clean</td></tr>
          <tr><td><kbd>s</kbd></td><td>Toggle sort</td></tr>
          <tr><td><kbd>?</kbd></td><td>This help</td></tr>
          <tr><td><kbd>Esc</kbd></td><td>Close menus, then clear search</td></tr>
        </tbody>
      </table>
      <p class="hint">Mute never shows an error (or a type) until you unmute it in Settings.</p>
      <button class="btn" on:click={onHelp}>Close</button>
    </div>
  </div>
{/if}

{#if statusMsg}
  <div class="toast" role="status">
    <span>{statusMsg}</span>
    {#if toastActionLabel}
      <button type="button" class="toast-action" on:click={onToastAction}>{toastActionLabel}</button>
    {/if}
  </div>
{/if}
