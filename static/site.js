'use strict';

const go = new Go();
let wasmReady = false;
let cmEditor   = null;

async function initWasm() {
  const statusEl = document.getElementById('wasm-status');
  const runBtn   = document.getElementById('run-btn');

  try {
    const response = await fetch('tengo.wasm.gz');
    if (!response.ok) throw new Error(`HTTP ${response.status}`);

    const decompressed = response.body.pipeThrough(new DecompressionStream('gzip'));
    const wasmResponse = new Response(decompressed, {
      headers: { 'Content-Type': 'application/wasm' },
    });

    const result = await WebAssembly.instantiateStreaming(wasmResponse, go.importObject);
    go.run(result.instance);
    wasmReady = true;
    if (runBtn)   runBtn.disabled = false;
    if (statusEl) { statusEl.textContent = 'ctrl+enter'; statusEl.className = 'ready'; }
  } catch (err) {
    if (statusEl) { statusEl.textContent = 'runtime load failed: ' + err.message; statusEl.className = 'error'; }
    console.error('WASM init error:', err);
  }
}

function getCode() {
  if (cmEditor) return cmEditor.getValue();
  const ta = document.getElementById('code-editor');
  return ta ? ta.value : '';
}

function setCode(val) {
  if (cmEditor) { cmEditor.setValue(val); return; }
  const ta = document.getElementById('code-editor');
  if (ta) ta.value = val;
}

function runCode() {
  if (!wasmReady) return;
  const output = document.getElementById('output');
  if (!output) return;

  output.className = '';
  output.textContent = '…';

  // Yield to the browser before the synchronous WASM call so it can repaint.
  setTimeout(() => {
    try {
      const result = tengoRun(getCode());
      if (result.error) {
        output.className = 'error';
        output.textContent = result.error;
      } else {
        output.textContent = result.output !== '' ? result.output : '(no output)';
      }
    } catch (err) {
      output.className = 'error';
      output.textContent = 'runtime error: ' + err.message;
    }
  }, 10);
}

function resetCode() {
  const ta = document.getElementById('code-editor');
  setCode(ta ? ta.dataset.original : '');
  const output = document.getElementById('output');
  if (output) {
    output.className = '';
    output.innerHTML = '<span class="hint">$ (output will appear here)</span>';
  }
}

document.addEventListener('DOMContentLoaded', () => {
  const ta = document.getElementById('code-editor');
  if (!ta || typeof CodeMirror === 'undefined') return;

  cmEditor = CodeMirror.fromTextArea(ta, {
    mode:           'go',   // close enough for Tengo syntax
    lineNumbers:    true,
    indentUnit:     4,
    tabSize:        4,
    indentWithTabs: false,
    lineWrapping:   false,
    extraKeys: {
      'Ctrl-Enter': runCode,
      'Cmd-Enter':  runCode,
      Tab: (cm) => cm.somethingSelected()
        ? cm.indentSelection('add')
        : cm.replaceSelection('    '),
    },
  });

  // Let CM grow to fit content rather than scroll inside a fixed box.
  cmEditor.setSize('100%', 'auto');
});

// Ctrl+Enter / Cmd+Enter anywhere on the page.
document.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
    e.preventDefault();
    runCode();
  }
});

initWasm();
