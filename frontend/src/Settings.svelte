<script>
  import ProjectForm from './ProjectForm.svelte'

  export let projectDir = ''
  export let sources = []
  export let editorCommand = 'code'
  export let customEditor = ''
  export let notifications = true
  export let sound = false
  export let fromStart = true
  export let followLatest = false
  export let clearOnCommit = false
  export let extraHostsText = ''
  export let mutes = []
  export let recent = []
  export let error = ''
  export let busy = false
  export let onOpenFolder
  export let onAddFile
  export let onAddBrowser = () => {}
  export let onAddCommand = () => {}
  export let onRemove
  export let onStart
  export let onSourceChange
  export let onBrowseEditor = () => {}
  export let onExtraHosts = () => {}
  export let onUnmute = () => {}
  export let onOpenRecent = () => {}
  export let onNewProject = () => {}
  export let onOpenLog = () => {}
  export let onRevealExtension = () => {}
  export let onCopyExtensionPath = () => {}
  export let onClose = () => {}
  export let logPath = ''
  export let extDir = ''
  export let extZipURL = ''

  function projectName(p) {
    if (!p) return ''
    const dir = p.dir || p.Dir || ''
    const parts = dir.replace(/\\/g, '/').split('/').filter(Boolean)
    return parts[parts.length - 1] || dir || (p.configPath || p.ConfigPath)
  }
</script>

<div class="settings-body">
  {#if error}
    <div class="banner">{error}</div>
  {/if}

  <div class="field">
    <span class="lbl">Project folder</span>
    <div class="path">{projectDir || '—'}</div>
  </div>
  <div class="row-actions">
    <button class="btn" on:click={onOpenFolder}>Change folder</button>
    <button class="btn" on:click={onAddFile}>Add log file</button>
    <button class="btn" on:click={onAddCommand}>Add command</button>
    <button class="btn" on:click={onAddBrowser}>Add browser source</button>
  </div>

  <ProjectForm
    {sources}
    {editorCommand}
    {customEditor}
    bind:notifications
    bind:sound
    bind:fromStart
    bind:followLatest
    bind:clearOnCommit
    {extraHostsText}
    showExtraHosts={true}
    {onSourceChange}
    {onRemove}
    {onBrowseEditor}
    {onExtraHosts}
  />

  {#if mutes.length}
    <div class="field">
      <span class="lbl">Muted</span>
      <div class="mute-list">
        {#each mutes as m}
          <div class="mute-row">
            <span class="mute-kind">{m.kind}</span>
            <span class="path" title={m.value}>{m.value}</span>
            <button class="btn ghost" on:click={() => onUnmute(m.kind, m.value)}>Unmute</button>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  {#if recent.length}
    <div class="field">
      <span class="lbl">Recent projects</span>
      <div class="recent-list">
        {#each recent as p}
          <button type="button" class="recent-item" on:click={() => onOpenRecent(p.configPath || p.ConfigPath)}>
            <strong>{projectName(p)}</strong>
            <span class="path">{p.dir || p.Dir}</span>
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <div class="field">
    <span class="lbl">Faultline log</span>
    <div class="path" title={logPath}>{logPath || 'Written next to app config as faultline.log'}</div>
    <p class="hint">App errors also show in the inbox as source “faultline”.</p>
  </div>
  <div class="row-actions">
    <button type="button" class="btn" on:click={onOpenLog}>Open Faultline log</button>
  </div>

  <div class="field">
    <span class="lbl">Browser extension</span>
    <div class="path" title={extDir}>{extDir || 'Bundled with the app'}</div>
    <p class="hint">Load unpacked from that folder, or while watching download {extZipURL || 'http://127.0.0.1:9477/extension.zip'}.</p>
  </div>
  <div class="row-actions">
    <button type="button" class="btn" on:click={onRevealExtension}>Open extension folder</button>
    <button type="button" class="btn" on:click={onCopyExtensionPath}>Copy path</button>
  </div>

  <div class="row-actions">
    <button class="btn primary" disabled={busy || sources.length === 0 || !projectDir} on:click={onStart}>
      {busy ? 'Working…' : 'Save and restart'}
    </button>
    <button class="btn" on:click={onNewProject}>New project</button>
    <button class="btn ghost" on:click={onClose}>Close</button>
  </div>
</div>
