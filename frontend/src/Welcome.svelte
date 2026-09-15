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
  export let error = ''
  export let busy = false
  export let recent = []
  export let scannedEmpty = false
  export let onOpenFolder
  export let onAddFile
  export let onAddBrowser = () => {}
  export let onAddCommand = () => {}
  export let onRemove
  export let onStart
  export let onSourceChange
  export let onBrowseEditor = () => {}
  export let onOpenRecent = () => {}
  export let extDir = ''
  export let extZipURL = ''
  export let onRevealExtension = () => {}
  export let onCopyExtensionPath = () => {}

  function projectName(p) {
    if (!p) return ''
    const dir = p.dir || p.Dir || ''
    const parts = dir.replace(/\\/g, '/').split('/').filter(Boolean)
    return parts[parts.length - 1] || dir || (p.configPath || p.ConfigPath)
  }
</script>

<div class="welcome">
  <div class="card">
    <h1>Faultline</h1>
    <p class="lede">Watch any log file or command. Group repeats. Jump to the line in your editor. Browser errors arrive through the extension — nothing to add to your app.</p>

    {#if error}
      <div class="banner">{error}</div>
    {/if}

    {#if recent.length && !projectDir}
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
      <span class="lbl">Project folder</span>
      <div class="path">{projectDir || 'Choose a folder — settings are saved as faultline.yaml'}</div>
    </div>
    <div class="row-actions">
      <button class="btn" on:click={onOpenFolder}>Open folder</button>
      <button class="btn" on:click={onAddFile}>Add log file</button>
      <button class="btn" on:click={onAddCommand}>Add command</button>
      <button class="btn" on:click={onAddBrowser}>Add browser source</button>
    </div>

    {#if scannedEmpty}
      <div class="callout">
        <strong>No log files found.</strong>
        Most local apps print to the terminal. Add a command source such as <code>npm run dev</code> or <code>docker compose logs -f</code>, or capture page errors with the browser extension.
      </div>
    {/if}

    <ProjectForm
      {sources}
      {editorCommand}
      {customEditor}
      bind:notifications
      bind:sound
      bind:fromStart
      bind:followLatest
      bind:clearOnCommit
      {onSourceChange}
      {onRemove}
      {onBrowseEditor}
    />

    <details class="help-ext">
      <summary>Browser extension</summary>
      <ol>
        <li>Click <strong>Open extension folder</strong>, then Chrome / Edge: <code>chrome://extensions</code> → Developer mode → Load unpacked → select that folder.</li>
        <li>Firefox 128+: <code>about:debugging#/runtime/this-firefox</code> → Load Temporary Add-on → <code>manifest.json</code> in the same folder.</li>
        <li>While Faultline is watching, download <code>faultline-extension.zip</code> from {extZipURL || 'http://127.0.0.1:9477/extension.zip'}.</li>
        <li>Open your app on localhost. The inbox pill turns green when the extension connects.</li>
      </ol>
      <div class="row-actions">
        <button type="button" class="btn" on:click={onRevealExtension}>Open extension folder</button>
        <button type="button" class="btn" on:click={onCopyExtensionPath}>Copy path</button>
      </div>
      {#if extDir}
        <p class="path" title={extDir}>{extDir}</p>
      {/if}
      <p class="hint">Custom hosts (not localhost) go in Settings after you start, or in the extension’s Options page.</p>
    </details>

    <button class="btn primary" disabled={busy || sources.length === 0 || !projectDir} on:click={onStart}>
      {busy ? 'Working…' : 'Start watching'}
    </button>
  </div>
</div>
