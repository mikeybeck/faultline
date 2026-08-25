<script>
  import { onMount } from 'svelte'
  export let events = []
  export let total = 0
  export let sources = []
  export let selected
  export let filter = ''
  export let severity = 'all'
  export let sort = 'recent'
  export let statusMsg = ''
  export let marked = false
  export let onSelect
  export let onOpen
  export let onClear
  export let onClearMark
  export let onCopy
  export let onSettings
  export let onFilter
  export let onSeverity
  export let onSort

  const ROW = 76
  const OVERSCAN = 8
  let listEl
  let scrollTop = 0
  let viewH = 480

  $: start = Math.max(0, Math.floor(scrollTop / ROW) - OVERSCAN)
  $: visibleCount = Math.ceil(viewH / ROW) + OVERSCAN * 2
  $: end = Math.min(events.length, start + visibleCount)
  $: visible = events.slice(start, end)
  $: padTop = start * ROW
  $: padBot = Math.max(0, (events.length - end) * ROW)

  onMount(() => {
    const el = listEl
    if (!el) return
    const apply = () => {
      viewH = el.clientHeight || 480
    }
    apply()
    const ro = new ResizeObserver(apply)
    ro.observe(el)
    return () => ro.disconnect()
  })

  function onScroll(e) {
    scrollTop = e.target.scrollTop
  }

  function rel(iso) {
    if (!iso) return 'unknown'
    const t = new Date(iso).getTime()
    const d = Date.now() - t
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
</script>

<div class="top">
  <div class="brand">
    <h1>Faultline</h1>
    <span>inbox{#if total} · {total}{/if}</span>
  </div>
  <div class="pills">
    {#each sources as s}
      <div class="pill {pillClass(s.state)}" title={s.message || s.path}>
        <span class="dot"></span>
        {s.name}{#if s.state === 'ingesting' && s.message}<span class="pill-msg"> · {s.message}</span>{/if}
      </div>
    {/each}
    {#if sources.length === 0}
      <div class="pill">No sources</div>
    {/if}
  </div>
  {#if statusMsg}<span class="meta">{statusMsg}</span>{/if}
  <button class="btn" on:click={onSettings}>Settings</button>
</div>

<div class="toolbar">
  <input
    class="search"
    placeholder="Filter type, message, file…"
    value={filter}
    on:input={(e) => onFilter(e.target.value)}
  />
  <div class="chips">
    {#each ['all', 'critical', 'error', 'warning'] as s}
      <button class="chip" class:on={severity === s} on:click={() => onSeverity(s)}>{s}</button>
    {/each}
  </div>
  <select class="select" value={sort} on:change={(e) => onSort(e.target.value)}>
    <option value="recent">Recent</option>
    <option value="frequency">Frequency</option>
  </select>
  <button class="btn danger" on:click={onClear}>Clear</button>
  {#if marked}
    <button class="btn" title="Forget the clear point and read the log from the beginning" on:click={onClearMark}>Clear mark</button>
  {/if}
</div>

<div class="body">
  <div class="list" bind:this={listEl} on:scroll={onScroll}>
    {#if events.length === 0}
      <div class="empty">
        <div>
          <h2>No errors yet</h2>
          <p>Waiting for log activity. Append to a watched file to see it here.</p>
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
          <div class="meta"><span class="sev {ev.severity}">{ev.severity}</span> · {rel(ev.lastSeen)}</div>
        </div>
      {/each}
      <div style="height:{padBot}px"></div>
    {/if}
  </div>
  <div class="detail">
    {#if !selected}
      <div class="empty">
        <div>
          <h2>Select an error</h2>
          <p>Full message and stack appear here.</p>
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
        <button class="btn primary" disabled={!selected.file} on:click={() => onOpen(selected.hash)}>Open in editor</button>
        <button class="btn" on:click={() => onCopy(selected)}>Copy</button>
      </div>
      <div class="block">
        <h3>Message</h3>
        <div class="message">{selected.message || '—'}</div>
      </div>
      {#if selected.stack}
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
      <div class="meta">First {rel(selected.firstSeen)} · Last {rel(selected.lastSeen)}</div>
    {/if}
  </div>
</div>
