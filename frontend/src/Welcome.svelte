<script>
  export let projectDir = ''
  export let sources = []
  export let editorCommand = 'code'
  export let customEditor = ''
  export let notifications = true
  export let sound = false
  export let fromStart = false
  export let error = ''
  export let busy = false
  export let onOpenFolder
  export let onAddFile
  export let onRemove
  export let onStart
  export let onSourceChange
  export let actionLabel = 'Start watching'

  const builtins = ['code', 'cursor', 'phpstorm']
  $: editorSelect = builtins.includes(editorCommand) ? editorCommand : 'custom'
</script>

<div class="welcome">
  <div class="card">
    <h1>Faultline</h1>
    <p class="lede">Watch any log file. Group repeats. Jump to the line in your editor.</p>

    {#if error}
      <div class="banner">{error}</div>
    {/if}

    <div class="field">
      <span class="lbl">Project folder</span>
      <div class="path">{projectDir || 'Choose a folder — settings are saved as faultline.yaml'}</div>
    </div>
    <div class="row-actions">
      <button class="btn" on:click={onOpenFolder}>Open folder</button>
      <button class="btn" on:click={onAddFile}>Add log file</button>
    </div>

    <div class="field">
      <span class="lbl">Log sources</span>
      {#if sources.length === 0}
        <p class="lede">No logs yet. Open a project folder to scan for <code>*.log</code> files, or add one yourself.</p>
      {:else}
        <div class="source-list">
          {#each sources as src, i}
            <div class="source-item">
              <input value={src.name} on:input={(e) => onSourceChange(i, { name: e.target.value })} placeholder="name" />
              <select value={src.type} on:change={(e) => onSourceChange(i, { type: e.target.value })}>
                <option value="generic">generic</option>
                <option value="laravel">laravel</option>
                <option value="apache">apache</option>
              </select>
              <div class="path" title={src.path}>{src.path}</div>
              <button class="btn ghost danger" on:click={() => onRemove(i)}>Remove</button>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="field">
      <label for="editor-select">Editor</label>
      <select
        id="editor-select"
        value={editorSelect}
        on:change={(e) => {
          const v = e.target.value
          if (v === 'custom') onSourceChange(-1, { editor: customEditor || 'code' })
          else onSourceChange(-1, { editor: v })
        }}
      >
        <option value="code">VS Code</option>
        <option value="cursor">Cursor</option>
        <option value="phpstorm">PhpStorm</option>
        <option value="custom">Custom command</option>
      </select>
    </div>
    {#if editorSelect === 'custom'}
      <div class="field">
        <label for="custom-editor">Command — use {'{file}'} and {'{line}'}</label>
        <input
          id="custom-editor"
          value={customEditor}
          placeholder="nvim +{line} {file}"
          on:input={(e) => onSourceChange(-1, { editor: e.target.value, custom: true })}
        />
      </div>
    {/if}

    <div class="toggles">
      <label><input type="checkbox" bind:checked={notifications} /> Desktop notifications</label>
      <label><input type="checkbox" bind:checked={sound} /> Sound</label>
      <label><input type="checkbox" bind:checked={fromStart} /> Read existing log content</label>
    </div>

    <button class="btn primary" disabled={busy || sources.length === 0 || !projectDir} on:click={onStart}>
      {busy ? 'Working…' : actionLabel}
    </button>
  </div>
</div>
