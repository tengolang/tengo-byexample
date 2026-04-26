'use strict';

let go = null;
let wasmReady = false;

// Single-file mode: one CodeMirror instance.
let cmEditor = null;

// Multi-file mode: one CodeMirror instance per tab, keyed by tab index.
let cmTabs = {};     // { tabIndex: CodeMirror }
let activeTab = 0;

async function initWasm() {
  const statusEl = document.getElementById('wasm-status');
  const runBtn   = document.getElementById('run-btn');

  try {
    if (typeof Go === 'undefined') {
        throw new Error('wasm_exec.js not loaded');
    }
    go = new Go();
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

function isMultiFile() {
  return document.querySelector('.code-editor-tab') !== null;
}

function getCode() {
  if (isMultiFile()) return null; // unused in multi-file path
  if (cmEditor) return cmEditor.getValue();
  const ta = document.getElementById('code-editor');
  return ta ? ta.value : '';
}

function setCode(val) {
  if (cmEditor) { cmEditor.setValue(val); return; }
  const ta = document.getElementById('code-editor');
  if (ta) ta.value = val;
}

// Returns { name: code, ... } for all tabs in multi-file mode.
function getFiles() {
  const files = {};
  document.querySelectorAll('.code-editor-tab').forEach((ta, i) => {
    const name = ta.dataset.name.replace(/\.tengo$/, '');
    const cm = cmTabs[i];
    files[name] = cm ? cm.getValue() : ta.value;
  });
  return files;
}

function runCode() {
  if (!wasmReady) return;
  const output = document.getElementById('output');
  if (!output) return;

  output.className = '';
  output.textContent = '…';

  setTimeout(() => {
    try {
      let result;
      if (isMultiFile()) {
        result = tengoRun(getFiles());
      } else {
        result = tengoRun(getCode());
      }
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
  if (isMultiFile()) {
    document.querySelectorAll('.code-editor-tab').forEach((ta, i) => {
      const original = ta.dataset.original;
      const cm = cmTabs[i];
      if (cm) cm.setValue(original); else ta.value = original;
    });
  } else {
    const ta = document.getElementById('code-editor');
    setCode(ta ? ta.dataset.original : '');
  }
  const output = document.getElementById('output');
  if (output) {
    const expected = output.dataset.expected;
    if (expected) {
      output.className = 'expected-output';
      output.textContent = expected;
    } else {
      output.className = '';
      output.innerHTML = '<span class="hint">$ (output will appear here)</span>';
    }
  }
}

function switchTab(tabEl) {
  const idx = parseInt(tabEl.dataset.tab, 10);
  if (idx === activeTab) return;

  // Deactivate current tab.
  document.querySelectorAll('.file-tab').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.code-editor-tab').forEach(ta => {
    const cm = cmTabs[parseInt(ta.dataset.tab, 10)];
    if (cm) cm.getWrapperElement().style.display = 'none';
    else ta.style.display = 'none';
  });

  // Activate new tab.
  tabEl.classList.add('active');
  activeTab = idx;
  const newTa = document.querySelector(`.code-editor-tab[data-tab="${idx}"]`);
  if (newTa) {
    const cm = cmTabs[idx];
    if (cm) {
      cm.getWrapperElement().style.display = '';
      cm.refresh();
    } else {
      newTa.style.display = '';
    }
  }
}

function loadExample(select) {
  const opt = select.options[select.selectedIndex];
  if (!opt || !opt.value) return;
  const code = opt.dataset.code;
  setCode(code);
  const ta = document.getElementById('code-editor');
  if (ta) ta.dataset.original = code;
  const output = document.getElementById('output');
  if (output) {
    output.className = '';
    output.innerHTML = '<span class="hint">$ (output will appear here)</span>';
  }
}

document.addEventListener('DOMContentLoaded', () => {
  // --- Copy Buttons ---
  document.querySelectorAll('.copy-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const code = btn.nextElementSibling.querySelector('code').textContent;
      navigator.clipboard.writeText(code).then(() => {
        const originalText = btn.textContent;
        btn.textContent = 'copied!';
        setTimeout(() => { btn.textContent = originalText; }, 2000);
      });
    });
  });

  if (typeof CodeMirror === 'undefined') return;

  const cmOpts = {
    mode:           'go',
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
  };

  if (isMultiFile()) {
    // Initialise one CodeMirror per tab textarea.
    document.querySelectorAll('.code-editor-tab').forEach((ta, i) => {
      const cm = CodeMirror.fromTextArea(ta, cmOpts);
      cm.setSize('100%', 'auto');
      if (i !== 0) cm.getWrapperElement().style.display = 'none';
      cmTabs[i] = cm;
    });
  } else {
    const ta = document.getElementById('code-editor');
    if (!ta) return;
    cmEditor = CodeMirror.fromTextArea(ta, cmOpts);
    cmEditor.setSize('100%', 'auto');
  }
});

// Ctrl+Enter / Cmd+Enter anywhere on the page.
document.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
    e.preventDefault();
    runCode();
  }
});

// Only initialize WASM if we're on an example page with a playground.
if (document.getElementById('code-editor') || document.querySelector('.code-editor-tab')) {
  initWasm();
}
