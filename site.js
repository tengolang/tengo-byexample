'use strict';

let go = null;
let wasmReady = false;
let cmEditor = null;

// Multi-file state: tab contents keyed by index, active tab index.
let tabContents = {};
let activeTabIdx = 0;

function isMultiFile() {
  return document.getElementById('tab-data') !== null;
}

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

function getCode() {
  if (cmEditor) return cmEditor.getValue();
  const ta = document.getElementById('code-editor');
  return ta ? ta.value : '';
}

function setCode(val) {
  if (cmEditor && !isMultiFile()) { cmEditor.setValue(val); return; }
  const ta = document.getElementById('code-editor');
  if (ta) ta.value = val;
}

// Collect {moduleName: source} for all tabs (saves active tab first).
function getFiles() {
  if (cmEditor) tabContents[activeTabIdx].code = cmEditor.getValue();
  const files = {};
  Object.values(tabContents).forEach(tab => {
    files[tab.name.replace(/\.tengo$/, '')] = tab.code;
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
      const result = isMultiFile() ? tengoRun(getFiles()) : tengoRun(getCode());
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
    Object.keys(tabContents).forEach(i => {
      tabContents[i].code = tabContents[i].original;
    });
    if (cmEditor) cmEditor.setValue(tabContents[activeTabIdx].code);
  } else {
    const ta = document.getElementById('code-editor');
    const orig = ta ? ta.dataset.original : '';
    if (cmEditor) cmEditor.setValue(orig); else if (ta) ta.value = orig;
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
  const newIdx = parseInt(tabEl.dataset.tab, 10);
  if (newIdx === activeTabIdx) return;
  // Save current tab content before switching.
  if (cmEditor) tabContents[activeTabIdx].code = cmEditor.getValue();
  // Update tab UI.
  document.querySelectorAll('.file-tab').forEach(t => t.classList.remove('active'));
  tabEl.classList.add('active');
  // Load new tab into the shared editor.
  activeTabIdx = newIdx;
  if (cmEditor) {
    cmEditor.setValue(tabContents[newIdx].code);
    cmEditor.refresh();
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
    // Load tab content from JSON script element (avoids attribute newline normalization).
    const tabDataEl = document.getElementById('tab-data');
    const tabs = JSON.parse(tabDataEl.textContent);
    tabs.forEach((tab, i) => {
      tabContents[i] = { name: tab.name, code: tab.code, original: tab.original };
    });
    // One shared CodeMirror instance mounted on #multi-editor.
    const editorEl = document.getElementById('multi-editor');
    if (editorEl && Object.keys(tabContents).length > 0) {
      cmEditor = CodeMirror(editorEl, { ...cmOpts, value: tabContents[0].code });
      cmEditor.setSize('100%', 'auto');
    }
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

// Only initialize WASM if there is an editor on the page.
if (document.getElementById('code-editor') || document.getElementById('multi-editor')) {
  initWasm();
}
