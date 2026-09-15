<script>
  export let sources = []
  export let editorCommand = 'code'
  export let customEditor = ''
  export let notifications = true
  export let sound = false
  export let fromStart = true
  export let followLatest = false
  export let clearOnCommit = false
  export let extraHostsText = ''
  export let showExtraHosts = false
  export let showFromStart = true
  export let showClearOnCommit = true
  export let onSourceChange
  export let onRemove
  export let onBrowseEditor = () => {}
  export let onExtraHosts = () => {}

  const builtins = ['code', 'cursor', 'phpstorm']
  $: editorSelect = builtins.includes(editorCommand) ? editorCommand : 'custom'

  const types = ['generic', 'laravel', 'apache', 'json', 'browser', 'command']
  const parsers = ['generic', 'json', 'laravel', 'apache']

  function pathPlaceholder(type) {
    if (type === 'command') return 'npm run dev'
    if (type === 'browser') return '127.0.0.1:9477'
    return 'path/to/app.log'
  }
</script>

<div class="field">
  <span class="lbl">Log sources</span>
  {#if sources.length === 0}
    <p class="lede">No sources yet. Open a project folder to scan for <code>*.log</code> files, watch a command, or add a browser source.</p>
  {:else}
    <div class="source-list">
      {#each sources as src, i}
        <div class="source-item" class:command={src.type === 'command'}>
          <input value={src.name} on:input={(e) => onSourceChange(i, { name: e.target.value })} placeholder="name" />
          <select value={src.type} on:change={(e) => onSourceChange(i, { type: e.target.value })}>
            {#each types as t}
              <option value={t}>{t}</option>
            {/each}
          </select>
          {#if src.type === 'command'}
            <select value={src.parser || 'generic'} on:change={(e) => onSourceChange(i, { parser: e.target.value })} title="Parser for command output">
              {#each parsers as t}
                <option value={t}>{t}</option>
              {/each}
            </select>
          {/if}
          <input
            class="path-input"
            value={src.path}
            placeholder={pathPlaceholder(src.type)}
            title={src.path}
            on:input={(e) => onSourceChange(i, { path: e.target.value })}
          />
          <button class="btn ghost danger" on:click={() => onRemove(i)}>Remove</button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<div class="field">
  <label for="editor-select">Editor</label>
  <div class="field-row">
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
    <button type="button" class="btn" on:click={onBrowseEditor}>Browse…</button>
  </div>
</div>
{#if editorSelect === 'custom'}
  <div class="field">
    <label for="custom-editor">Command — full path, or {'{file}'} and {'{line}'}</label>
    <input
      id="custom-editor"
      value={customEditor}
      placeholder={'phpstorm64.exe or nvim +{line} {file}'}
      on:input={(e) => onSourceChange(-1, { editor: e.target.value, custom: true })}
    />
  </div>
{/if}

<div class="toggles">
  <label><input type="checkbox" bind:checked={notifications} /> Desktop notifications</label>
  <label><input type="checkbox" bind:checked={sound} /> Sound</label>
  {#if showFromStart}
    <label><input type="checkbox" bind:checked={fromStart} /> Load existing log lines</label>
  {/if}
  <label><input type="checkbox" bind:checked={followLatest} /> Always select the most recent error</label>
  {#if showClearOnCommit}
    <label><input type="checkbox" bind:checked={clearOnCommit} /> Clear inbox when git HEAD changes</label>
  {/if}
</div>

{#if showExtraHosts}
  <div class="field">
    <label for="extra-hosts">Extra browser hosts</label>
    <textarea
      id="extra-hosts"
      rows="4"
      value={extraHostsText}
      placeholder="localphishingbox.com"
      on:input={(e) => onExtraHosts(e.target.value)}
    ></textarea>
    <p class="hint">One hostname per line. The extension also picks these up from <code>http://127.0.0.1:9477/hosts</code> while Faultline is running.</p>
  </div>
{/if}
