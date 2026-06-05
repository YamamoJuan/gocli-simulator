(function() {
  'use strict';

  // ── Elements ─────────────────────────────────────────────────
  const output       = document.getElementById('terminalOutput');
  const inputArea    = document.getElementById('inputArea');
  const cursor       = document.getElementById('cursor');
  const terminalBody = document.getElementById('terminalBody');
  const footerStatus = document.getElementById('footerStatus');

  // ── State ────────────────────────────────────────────────────
  let commandHistory = [];
  let historyIndex   = -1;
  let isRunning      = false;

  // ============================================================
  // WELCOME
  // ============================================================
  function showWelcome() {
    const lines = [
      { text: 'Welcome to Go CLI Simulator', cls: 'cmd-welcome welcome-title' },
      { text: 'Write Go code directly and press Enter to execute.', cls: 'cmd-welcome' },
      { text: 'Type <span class="fg-yellow">clear</span> to clear the terminal.', cls: 'cmd-welcome' },
      { text: 'Type <span class="fg-yellow">help</span> for available commands.', cls: 'cmd-welcome' },
      { text: 'Type <span class="fg-yellow">gonano</span> for the built-in text editor.', cls: 'cmd-welcome' },
      { text: '', cls: 'cmd-welcome' },
    ];
    lines.forEach(l => appendLine('', l.text, l.cls));
  }

  // ── Append a line ────────────────────────────────────────────
  function appendLine(promptText, text, cls) {
    const block = document.createElement('div');
    block.className = 'cmd-block';

    if (promptText) {
      const p = document.createElement('span');
      p.className = 'cmd-prompt';
      p.textContent = promptText + ' ';
      block.appendChild(p);
    }

    if (text) {
      const c = document.createElement('span');
      c.className = cls || '';
      c.innerHTML = text;
      block.appendChild(c);
    }

    output.appendChild(block);
    scrollToBottom();
  }

  // ── Append output (stdout + stderr) ─────────────────────────
  function appendOutput(stdout, stderr) {
    const block = document.createElement('div');
    block.className = 'cmd-block';

    if (stdout) {
      const o = document.createElement('span');
      o.className = 'cmd-output';
      o.innerHTML = ansiToHtml(escapeHtml(stdout));
      block.appendChild(o);
    }

    if (stderr) {
      stderr.split('\n').forEach((line, i, arr) => {
        if (!line.trim()) return;
        const e = document.createElement('span');
        e.className = 'cmd-error';
        e.innerHTML = ansiToHtml(escapeHtml(line));
        block.appendChild(e);
        if (i < arr.length - 1) block.appendChild(document.createElement('br'));
      });
    }

    output.appendChild(block);
    scrollToBottom();
  }

  // ── ANSI strip ──────────────────────────────────────────────
  function ansiToHtml(text) {
    return text
      .replace(/\x1b\[[0-9;]*[A-Za-z]/g, '')
      .replace(/\x1b\][^\x07]*\x07/g, '')
      .replace(/\x1b[^A-Za-z]*/g, '');
  }

  // ── HTML escape ─────────────────────────────────────────────
  function escapeHtml(text) {
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  // ── Show typed command ──────────────────────────────────────
  function showCommand(text) {
    const block = document.createElement('div');
    block.className = 'cmd-block';

    const prompt = document.createElement('span');
    prompt.className = 'cmd-prompt';
    prompt.textContent = 'user@kasming:~$ ';

    const cmd = document.createElement('span');
    cmd.className = 'cmd-text';
    cmd.textContent = text;

    block.appendChild(prompt);
    block.appendChild(cmd);
    output.appendChild(block);
    scrollToBottom();
  }

  // ── Show error ──────────────────────────────────────────────
  function showError(msg) {
    const block = document.createElement('div');
    block.className = 'cmd-block';
    const e = document.createElement('span');
    e.className = 'cmd-error';
    e.textContent = msg;
    block.appendChild(e);
    output.appendChild(block);
    scrollToBottom();
  }

  // ── Show info ───────────────────────────────────────────────
  function showInfo(msg) {
    const block = document.createElement('div');
    block.className = 'cmd-block';
    const el = document.createElement('span');
    el.className = 'cmd-info';
    el.textContent = msg;
    block.appendChild(el);
    output.appendChild(block);
    scrollToBottom();
  }

  // ── Scroll to bottom ────────────────────────────────────────
  function scrollToBottom() {
    terminalBody.scrollTop = terminalBody.scrollHeight;
  }

  // ── Status bar ──────────────────────────────────────────────
  function setStatus(text, cls) {
    footerStatus.innerHTML = cls ? `<span class="${cls}">${text}</span>` : text;
  }

  // ============================================================
  // EXECUTE VIA API
  // ============================================================
  async function runCode(code) {
    if (isRunning) return;
    isRunning = true;
    setStatus('⏳ Compiling & running...', 'status-running');
    showCommand(code);

    try {
      const resp = await fetch('/run', {
        method: 'POST',
        headers: { 'Content-Type': 'text/plain' },
        body: code,
      });
      const data = await resp.json();

      if (data.stdout || data.stderr) {
        appendOutput(data.stdout || '', data.stderr || '');
      } else {
        showInfo('(no output)');
      }
    } catch (err) {
      showError('Connection error: ' + err.message);
    } finally {
      isRunning = false;
      setStatus('Ready');
      focusInput();
    }
  }

  // ============================================================
  // BUILT-IN COMMANDS
  // ============================================================
  const builtins = {

    clear: () => {
      output.innerHTML = '';
      return true;
    },

    help: () => {
      const lines = [
        '# Go CLI Simulator — Available Commands',
        '',
        '  Direct Go code:',
        '    fmt.Println("Hello!")',
        '    for i := 0; i &lt; 5; i++ { fmt.Println(i) }',
        '    fmt.Printf("Value: %d\\n", 42)',
        '',
        '  Built-in commands:',
        '    clear          Clear the terminal',
        '    help           Show this help message',
        '    about          About this simulator',
        '    date           Show current date & time',
        '    gonano         Open the GNU nano text editor',
        '    echo &lt;text&gt;   Echo text to terminal',
        '    go version     Show Go version info',
        '',
        '  Keyboard shortcuts:',
        '    Enter          Execute code / run command',
        '    Ctrl+L         Clear terminal',
        '    Ctrl+C         Cancel current input',
        '    ↑ / ↓          Navigate command history',
        '',
        '  Note: Package "fmt" is auto-imported.',
      ];

      lines.forEach(l => {
        const block = document.createElement('div');
        block.className = 'cmd-block';
        const el = document.createElement('span');
        el.className = l.startsWith('#') ? 'cmd-info' : 'cmd-output';
        el.innerHTML = l;
        block.appendChild(el);
        output.appendChild(block);
      });
      scrollToBottom();
      return true;
    },

    about: () => {
      const block = document.createElement('div');
      block.className = 'cmd-block';
      block.innerHTML = [
        '<span class="ansi-fg-green">Go CLI Web Simulator</span>',
        '  Version 1.0.0',
        '  A web-based terminal for running Go code.',
        '  Built with Go (net/http) + Vanilla JS.',
        '',
        '  Features:',
        '  • Execute Go code in real-time',
        '  • Git Bash-like terminal styling',
        '  • macOS-style window controls',
        '  • Command history (↑/↓)',
        '  • Built-in gonano text editor',
        '  • Auto-import fmt package',
        '  • 10-second execution timeout',
      ].join('\n');
      block.style.whiteSpace = 'pre-wrap';
      block.style.lineHeight = '1.6';
      output.appendChild(block);
      scrollToBottom();
      return true;
    },

    date: () => {
      const now = new Date();
      const block = document.createElement('div');
      block.className = 'cmd-block';
      const el = document.createElement('span');
      el.className = 'cmd-output';
      el.textContent = now.toString();
      block.appendChild(el);
      output.appendChild(block);
      scrollToBottom();
      return true;
    },

    gonano: () => {
      showCommand('gonano');
      const block = document.createElement('div');
      block.className = 'cmd-block';
      block.innerHTML = [
        '<span style="color:#3fb950">Opening GNU nano text editor...</span>',
        '<span style="color:#586e75">  ^X  Exit  |  ^R  Run Go  |  Esc  Cancel</span>',
      ].join('<br>');
      output.appendChild(block);
      scrollToBottom();
      setTimeout(() => openGonano(), 100);
      return true;
    },
  };

  // ============================================================
  // AUTO-IMPORT FMT
  // ============================================================
  function autoImportFmt(code) {
    if (/^package\s+\w+/m.test(code) || /"fmt"/.test(code)) return code;

    const trimmed = code.trim();

    if (/^(fmt\.|print|println)\(/.test(trimmed)) {
      return `package main\nimport "fmt"\nfunc main() {\n  ${code}\n}`;
    }

    if (/^(for|if|switch|var|const|type)\b/.test(trimmed)) {
      return `package main\nimport "fmt"\nfunc main() {\n  ${code}\n}`;
    }

    if (trimmed === '}' || /^func\b/.test(trimmed)) {
      if (!/"fmt"/.test(code)) {
        return code.replace(/(package\s+\w+)?/, (m) =>
          m ? `${m}\nimport "fmt"` : 'package main\nimport "fmt"\n'
        );
      }
      return code;
    }

    if (code.includes('package main')) {
      if (!/"fmt"/.test(code)) {
        code = code.replace(/(package main\s*)/, '$1\nimport "fmt"');
      }
      return code;
    }

    return `package main\nimport "fmt"\n\n${code}`;
  }

  // ============================================================
  // HANDLE INPUT
  // ============================================================
  function handleInput() {
    if (isRunning) return;

    const raw = inputArea.textContent.trim();
    if (!raw) return;

    commandHistory.unshift(raw);
    if (commandHistory.length > 100) commandHistory.pop();
    historyIndex = -1;
    inputArea.textContent = '';

    if (builtins[raw]) {
      showCommand(raw);
      builtins[raw]();
      return;
    }

    if (raw.startsWith('echo ')) {
      showCommand(raw);
      const block = document.createElement('div');
      block.className = 'cmd-block';
      const el = document.createElement('span');
      el.className = 'cmd-output';
      el.textContent = raw.slice(5);
      block.appendChild(el);
      output.appendChild(block);
      scrollToBottom();
      return;
    }

    if (raw === 'go version') {
      showCommand(raw);
      const block = document.createElement('div');
      block.className = 'cmd-block';
      const el = document.createElement('span');
      el.className = 'cmd-output';
      el.textContent = 'go version go1.22+ (simulated)';
      block.appendChild(el);
      output.appendChild(block);
      scrollToBottom();
      return;
    }

    runCode(autoImportFmt(raw));
  }

  // ============================================================
  // FOCUS / CURSOR
  // ============================================================
  function focusInput() {
    inputArea.focus();
    moveCursorToEnd();
  }

  function moveCursorToEnd() {
    const range = document.createRange();
    const sel   = window.getSelection();
    range.selectNodeContents(inputArea);
    range.collapse(false);
    sel.removeAllRanges();
    sel.addRange(range);
  }

  // ============================================================
  // EVENT LISTENERS — TERMINAL INPUT
  // ============================================================
  inputArea.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleInput();
    } else if (e.ctrlKey && e.key === 'l') {
      e.preventDefault();
      builtins.clear();
    } else if (e.ctrlKey && e.key === 'c') {
      e.preventDefault();
      showCommand(inputArea.textContent + '^C');
      inputArea.textContent = '';
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (historyIndex < commandHistory.length - 1) {
        historyIndex++;
        inputArea.textContent = commandHistory[historyIndex];
        moveCursorToEnd();
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIndex > 0) {
        historyIndex--;
        inputArea.textContent = commandHistory[historyIndex];
        moveCursorToEnd();
      } else if (historyIndex === 0) {
        historyIndex = -1;
        inputArea.textContent = '';
      }
    }
  });

  inputArea.addEventListener('input', moveCursorToEnd);

  terminalBody.addEventListener('click', (e) => {
    if (e.target === terminalBody || e.target.classList.contains('cmd-block')) {
      focusInput();
    }
  });

  // Window control buttons
  window.windowCtrl = function(action) {
    const win = document.querySelector('.window');
    if (action === 'close') {
      if (confirm('Close Go CLI Simulator?')) {
        document.body.innerHTML = [
          '<div style="display:flex;align-items:center;justify-content:center;',
          'height:100vh;color:#6e7681;font-family:monospace;text-align:center;padding:40px;">',
          'Go CLI Simulator closed.<br><br>',
          '<button onclick="location.reload()" style="background:#073642;',
          'color:#93a1a1;border:1px solid #0a4454;padding:8px 20px;',
          'border-radius:4px;cursor:pointer;font-family:monospace;">Refresh to restart</button>',
          '</div>'
        ].join('');
      }
    } else if (action === 'minimize') {
      win.style.transition = 'transform 0.3s, opacity 0.3s';
      win.style.transform = 'scale(0.05)';
      win.style.transformOrigin = 'bottom left';
      win.style.opacity = '0';
      setTimeout(() => {
        win.style.transition = '';
        win.style.transform = '';
        win.style.transformOrigin = '';
        win.style.opacity = '';
      }, 1200);
    } else if (action === 'maximize') {
      if (win.dataset.maxed === '1') {
        win.style.transition = 'all 0.25s ease';
        win.style.width = 'min(960px, 95vw)';
        win.style.height = 'min(600px, 90vh)';
        win.style.borderRadius = '8px';
        win.style.left = '';
        win.style.top = '';
        win.dataset.maxed = '0';
      } else {
        win.style.transition = 'all 0.25s ease';
        win.style.width = '100vw';
        win.style.height = '100vh';
        win.style.borderRadius = '0';
        win.style.left = '0';
        win.style.top = '0';
        win.dataset.maxed = '1';
      }
    }
  };

  // ============================================================
  // GONANO EDITOR
  // ============================================================

  // Shortcut state
  let pendingCtrlR = false;
  let pendingCtrlX = false;

  function gonanoResetShortcuts() {
    pendingCtrlR = false;
    pendingCtrlX = false;
  }

  function flashShortcut(index, color) {
    const shortcuts = document.querySelectorAll('.gonano-shortcut');
    if (shortcuts[index]) {
      shortcuts[index].style.color = color;
      shortcuts[index].style.fontWeight = 'bold';
      setTimeout(() => {
        shortcuts[index].style.color = '';
        shortcuts[index].style.fontWeight = '';
      }, 400);
    }
  }

  // ── Update gutter (line numbers) — dipanggil langsung, bukan di DOMContentLoaded ──
  // Ganti fungsi updateGonanoGutter di terminal.js dengan ini:

  function updateGonanoGutter(textarea) {
    const gutter   = document.querySelector('.gonano-gutter');
    if (!gutter) return;

    const lines      = textarea.value.split('\n');
    const totalLines = Math.max(lines.length, 30);
    const textUp     = textarea.value.substring(0, textarea.selectionStart);
    const curLine    = textUp.split('\n').length;

    let html = '';
    for (let i = 1; i <= totalLines; i++) {
      const active = i === curLine ? ' gutter-active' : '';
      html += `<div class="gonano-gutter-line${active}" id="gonano-gutter-${i}">${i}</div>`;
    }
    gutter.innerHTML = html;
    gutter.scrollTop = textarea.scrollTop;
  }

  // ── Update status bar ────────────────────────────────────────
  function updateGonanoStatus(textarea) {
    const statusL = document.getElementById('gonano-status-left');
    const statusR = document.getElementById('gonano-status-right');
    if (!statusL || !statusR) return;

    const text      = textarea.value;
    const cursorPos = textarea.selectionStart;
    const textUp    = text.substring(0, cursorPos);
    const curLine   = textUp.split('\n').length;
    const lineStart = textUp.lastIndexOf('\n') + 1;
    const curCol    = cursorPos - lineStart + 1;

    statusL.textContent = ` Ln ${curLine}, Col ${curCol}`;
    statusR.textContent = `UTF-8  |  Go  |  main.go  |  ${text.length} chars`;
  }

  // ── Open gonano ─────────────────────────────────────────────
  function openGonano() {
    const editor   = document.getElementById('gonano-editor');
    const textarea = document.getElementById('gonano-textarea');
    const menu     = document.getElementById('gonano-menu');

    if (!editor || !textarea) return;

    editor.style.display = 'flex';
    if (menu) menu.style.display = 'none';
    gonanoResetShortcuts();

    // Default Go template
    textarea.value = [
      'package main',
      '',
      'import "fmt"',
      '',
      'func main() {',
      '\tfmt.Println("Hello from gonano!")',
      '}',
      '',
    ].join('\n');

    // Focus and place cursor at end
    textarea.focus();

    // Update gutter + status SEBELUM DOMContentLoaded menjalankan listener
    updateGonanoGutter(textarea);
    updateGonanoStatus(textarea);

    // Place cursor at end of last line
    textarea.selectionStart = textarea.value.length;
    textarea.selectionEnd   = textarea.value.length;
  }

  // ── Close gonano ─────────────────────────────────────────────
  window.gonanoClose = function() {
    const editor   = document.getElementById('gonano-editor');
    const textarea = document.getElementById('gonano-textarea');
    const menu     = document.getElementById('gonano-menu');

    if (editor)   editor.style.display = 'none';
    if (textarea) textarea.value = '';
    if (menu)     menu.style.display = 'none';

    gonanoResetShortcuts();
    setTimeout(() => inputArea && inputArea.focus(), 50);
  };

  // ── Close menu only ──────────────────────────────────────────
  window.gonanoCloseMenu = function() {
    const menu = document.getElementById('gonano-menu');
    if (menu) menu.style.display = 'none';
    gonanoResetShortcuts();
    const textarea = document.getElementById('gonano-textarea');
    if (textarea) textarea.focus();
  };

  // ── Execute from gonano ──────────────────────────────────────
  window.gonanoExecute = async function() {
    const textarea = document.getElementById('gonano-textarea');
    const editor   = document.getElementById('gonano-editor');
    const menu     = document.getElementById('gonano-menu');

    const code = textarea ? textarea.value.trim() : '';

    if (menu)   menu.style.display = 'none';
    if (editor) editor.style.display = 'none';
    if (textarea) textarea.value = '';

    gonanoResetShortcuts();
    setTimeout(() => inputArea && inputArea.focus(), 50);

    if (!code) {
      showError('No code to execute.');
      return;
    }

    const block = document.createElement('div');
    block.className = 'cmd-block';
    block.innerHTML = [
      '<span class="cmd-prompt">user@kasming:~$ </span>',
      '<span class="cmd-text" style="color:#3fb950">[gonano] go run main.go</span>',
    ].join('');
    output.appendChild(block);

    setStatus('⏳ Running gonano code...', 'status-running');

    try {
      const resp = await fetch('/run', {
        method: 'POST',
        headers: { 'Content-Type': 'text/plain' },
        body: code,
      });
      const data = await resp.json();

      const resultBlock = document.createElement('div');
      resultBlock.className = 'cmd-block';

      if (data.stdout) {
        const outEl = document.createElement('span');
        outEl.className = 'cmd-output';
        outEl.textContent = data.stdout;
        resultBlock.appendChild(outEl);
      }

      if (data.stderr) {
        data.stderr.split('\n').forEach((line, i, arr) => {
          if (!line.trim()) return;
          const errEl = document.createElement('span');
          errEl.className = 'cmd-error';
          errEl.textContent = line;
          resultBlock.appendChild(errEl);
          if (i < arr.length - 1) resultBlock.appendChild(document.createElement('br'));
        });
      }

      output.appendChild(resultBlock);
      scrollToBottom();

    } catch (err) {
      const errBlock = document.createElement('div');
      errBlock.className = 'cmd-block';
      const el = document.createElement('span');
      el.className = 'cmd-error';
      el.textContent = 'Connection error: ' + err.message;
      errBlock.appendChild(el);
      output.appendChild(errBlock);
      scrollToBottom();
    }

    setStatus('Ready');
  };

  // ============================================================
  // GONANO EVENT LISTENERS — bind langsung (tidak pakai DOMContentLoaded)
  // sehingga gutter update langsung tersedia saat openGonano() dipanggil
  // ============================================================
  const textarea = document.getElementById('gonano-textarea');
  const gutter   = document.querySelector('.gonano-gutter');
  const menu     = document.getElementById('gonano-menu');
  const editor   = document.getElementById('gonano-editor');

  if (textarea) {

    // Sync gutter scroll dengan textarea scroll
    textarea.addEventListener('scroll', () => {
      if (gutter) gutter.scrollTop = textarea.scrollTop;
    });

    // Input → update gutter + status setiap kali user mengetik
    textarea.addEventListener('input', () => {
      updateGonanoGutter(textarea);
      updateGonanoStatus(textarea);
    });

    // Cursor position changes
    textarea.addEventListener('keyup',  () => updateGonanoStatus(textarea));
    textarea.addEventListener('click',  () => updateGonanoStatus(textarea));

    // ── Keyboard shortcuts ────────────────────────────────────
    textarea.addEventListener('keydown', (e) => {

      // Tab → insert 2 spaces (smart indent)
      if (e.key === 'Tab') {
        e.preventDefault();
        const start  = textarea.selectionStart;
        const end    = textarea.selectionEnd;
        const before = textarea.value.substring(0, start);
        const after  = textarea.value.substring(end);

        const prevLine = before.split('\n').pop();
        const insert   = prevLine.trim().endsWith('{') ? '\t\t' : '\t';

        textarea.value = before + insert + after;
        textarea.selectionStart = textarea.selectionEnd = start + insert.length;
        updateGonanoGutter(textarea);
        updateGonanoStatus(textarea);
        return;
      }

      // Ctrl+R → Execute (double-press: first = menu, second = run)
      if (e.ctrlKey && e.key === 'r') {
        e.preventDefault();

        if (!pendingCtrlR) {
          pendingCtrlR = true;
          flashShortcut(1, '#859900');

          if (menu) {
            menu.style.display = 'block';
            menu.querySelector('.gonano-menu-header').textContent = '^R — Execute';
            menu.querySelector('.gonano-menu-body').innerHTML = [
              '<div class="gonano-menu-item gonano-menu-active" onclick="gonanoExecute()">',
              '  Execute code with <span class="gonano-key">Enter</span> / <span class="gonano-key">R</span>',
              '</div>',
              '<div class="gonano-menu-item" onclick="gonanoCloseMenu()">',
              '  Cancel with <span class="gonano-key">Esc</span>',
              '</div>',
            ].join('');
          }
          return;
        }

        // Second Ctrl+R → run
        pendingCtrlR = false;
        if (menu) menu.style.display = 'none';
        window.gonanoExecute();
        return;
      }

      // Ctrl+X → Exit (double-press: first = menu, second = close)
      if (e.ctrlKey && e.key === 'x') {
        e.preventDefault();

        if (!pendingCtrlX) {
          pendingCtrlX = true;
          flashShortcut(0, '#f85149');

          if (menu) {
            menu.style.display = 'block';
            menu.querySelector('.gonano-menu-header').textContent = '^X — Exit nano';
            menu.querySelector('.gonano-menu-body').innerHTML = [
              '<div class="gonano-menu-item" onclick="gonanoExecute()">',
              '  <span class="gonano-key">Ctrl+X</span> Exit &amp; Run (save before exit)',
              '</div>',
              '<div class="gonano-menu-item gonano-menu-active" onclick="gonanoClose()">',
              '  <span class="gonano-key">X</span> Exit without running',
              '</div>',
              '<div class="gonano-menu-item" onclick="gonanoCloseMenu()">',
              '  Cancel with <span class="gonano-key">Esc</span>',
              '</div>',
            ].join('');
          }
          return;
        }

        // Second Ctrl+X → close
        pendingCtrlX = false;
        if (menu) menu.style.display = 'none';
        window.gonanoClose();
        return;
      }

      // Escape → close menu
      if (e.key === 'Escape') {
        if (menu && menu.style.display !== 'none') {
          gonanoResetShortcuts();
          window.gonanoCloseMenu();
        }
      }

      // Enter while menu is open → execute
      if (e.key === 'Enter' && menu && menu.style.display !== 'none') {
        e.preventDefault();
        window.gonanoExecute();
      }
    });

    // Click on editor background → refocus textarea
    if (editor) {
      editor.addEventListener('click', (e) => {
        if (e.target === editor || e.target.classList.contains('gonano-window')) {
          textarea.focus();
        }
      });
    }
  }

  // ============================================================
  // INIT
  // ============================================================
  showWelcome();
  focusInput();
  setStatus('Ready');
})();