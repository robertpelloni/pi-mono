// ────────────────────────────────────────────────────────────────
// pi-desktop - Wails Frontend
// ────────────────────────────────────────────────────────────────

const {
    InitAgent,
    SendMessage,
    AbortGeneration,
    GetModelInfo,
    GetStats,
    GetVersion,
    GetSessionID,
    CycleModel,
    SetThinkingLevel,
    NewSession,
    ExportSession,
    IsGenerating,
} = window.go.main.App;

// ── DOM refs ──
const messagesEl = document.getElementById('messages');
const chatEl = document.getElementById('chat');
const inputEl = document.getElementById('user-input');
const btnSend = document.getElementById('btn-send');
const btnAbort = document.getElementById('btn-abort');
const statusEl = document.getElementById('status');
const statusText = document.getElementById('status-text');
const modelInfoEl = document.getElementById('model-info');
const sessionIdEl = document.getElementById('session-id');
const versionEl = document.getElementById('version');
const footerStats = document.getElementById('footer-stats');

// ── State ──
let generating = false;
let currentToolEl = null;
let thinkingEl = null;

// ── Safe DOM helpers (no innerHTML) ──
function clearChildren(el) {
    while (el.firstChild) el.removeChild(el.firstChild);
}

function setSafeText(el, text) {
    clearChildren(el);
    el.appendChild(document.createTextNode(text));
}

function makeSpan(text, className) {
    const s = document.createElement('span');
    if (className) s.className = className;
    s.textContent = text;
    return s;
}

function escapeHtml(s) {
    const d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML; // browser-escaped, safe for diff highlighting
}

// ── Init ──
async function init() {
    try {
        const ver = await GetVersion();
        versionEl.textContent = 'v' + ver;

        const provider = new URLSearchParams(window.location.search).get('provider') || '';
        const model = new URLSearchParams(window.location.search).get('model') || '';

        await InitAgent(provider, model);

        const info = await GetModelInfo();
        modelInfoEl.textContent = info.provider + '/' + info.model;

        const sid = await GetSessionID();
        sessionIdEl.textContent = sid ? 'session:' + sid.substring(0, 8) : '';
    } catch (e) {
        addSystemMessage('Init error: ' + e, 'error');
    }

    inputEl.focus();
    updateStats();
    setInterval(updateStats, 2000);
}

// ── Wails Event Handlers ──
window.runtime.EventsOn('agent-event', (event) => {
    handleAgentEvent(event);
});

window.runtime.EventsOn('session-event', (event) => {
    handleSessionEvent(event);
});

window.runtime.EventsOn('agent-ready', (data) => {
    modelInfoEl.textContent = data.provider + '/' + data.model;
    sessionIdEl.textContent = data.session ? 'session:' + data.session.substring(0, 8) : '';
    addSystemMessage('Agent ready.');
});

window.runtime.EventsOn('generation-done', () => {
    setGenerating(false);
});

window.runtime.EventsOn('agent-error', (err) => {
    addSystemMessage('Error: ' + err, 'error');
    setGenerating(false);
});

// ── Agent Event Handler ──
function handleAgentEvent(event) {
    switch (event.type) {
        case 'agent_start':
            setGenerating(true);
            break;

        case 'message_update':
            if (event.eventType === 'text') {
                thinkingEl = null;
                appendAssistantText(event.text || '');
            } else if (event.eventType === 'thinking-start') {
                thinkingEl = addThinkingMessage();
            } else if (event.eventType === 'thinking') {
                appendThinkingText(event.text || '');
            } else if (event.eventType === 'thinking-end') {
                thinkingEl = null;
            }
            break;

        case 'tool_execution_start':
            currentToolEl = addToolMessage(event.tool, event.args);
            setStatus('Running: ' + event.tool);
            break;

        case 'tool_execution_end':
            if (currentToolEl) {
                finishToolMessage(currentToolEl, event.result, event.isError);
                currentToolEl = null;
            }
            setStatus('');
            break;

        case 'agent_end':
            setGenerating(false);
            break;
    }
    scrollToBottom();
}

function handleSessionEvent(event) {
    switch (event.type) {
        case 'compaction_start':
            addSystemMessage('Compacting context...');
            break;
        case 'compaction_end':
            addSystemMessage('Compaction complete');
            break;
        case 'model_select':
            GetModelInfo().then(info => {
                modelInfoEl.textContent = info.provider + '/' + info.model;
            });
            break;
    }
}

// ── Message Rendering ──
function addUserMessage(text) {
    const el = document.createElement('div');
    el.className = 'msg msg-user';
    el.textContent = text;
    messagesEl.appendChild(el);
}

function addSystemMessage(text, type) {
    const el = document.createElement('div');
    el.className = type === 'error' ? 'msg msg-error' : 'msg msg-system';
    el.textContent = text;
    messagesEl.appendChild(el);
}

let lastAssistantEl = null;

function appendAssistantText(text) {
    if (!lastAssistantEl || lastAssistantEl.dataset.type !== 'assistant') {
        lastAssistantEl = document.createElement('div');
        lastAssistantEl.className = 'msg msg-assistant';
        lastAssistantEl.dataset.type = 'assistant';
        const label = document.createElement('div');
        label.className = 'label';
        label.textContent = 'Assistant';
        lastAssistantEl.appendChild(label);
        const body = document.createElement('div');
        body.className = 'msg-body';
        lastAssistantEl.appendChild(body);
        messagesEl.appendChild(lastAssistantEl);
    }
    const body = lastAssistantEl.querySelector('.msg-body');
    body.textContent += text;
}

function addThinkingMessage() {
    const el = document.createElement('div');
    el.className = 'msg msg-thinking';
    el.textContent = 'Thinking... ';
    messagesEl.appendChild(el);
    lastAssistantEl = null;
    return el;
}

function appendThinkingText(text) {
    if (thinkingEl) {
        thinkingEl.textContent += text;
    }
}

const toolIcons = {
    bash: '>',
    read: 'R',
    edit: 'E',
    write: 'W',
    grep: 'G',
    find: 'F',
    ls: 'L',
    delegate_task: 'D',
};

function addToolMessage(name, args) {
    lastAssistantEl = null;
    const el = document.createElement('div');
    el.className = 'msg msg-tool';

    const header = document.createElement('div');
    header.className = 'tool-header';

    const iconSpan = document.createElement('span');
    iconSpan.className = 'tool-icon';
    iconSpan.textContent = toolIcons[name] || '*';

    const nameSpan = document.createElement('span');
    nameSpan.className = 'tool-name';
    nameSpan.textContent = name;

    const argsSpan = document.createElement('span');
    argsSpan.className = 'tool-args';
    argsSpan.textContent = '(' + (args ? formatArgs(args) : '') + ')';

    header.appendChild(iconSpan);
    header.appendChild(nameSpan);
    header.appendChild(argsSpan);

    const resultEl = document.createElement('div');
    resultEl.className = 'tool-result hidden';

    el.appendChild(header);
    el.appendChild(resultEl);
    messagesEl.appendChild(el);
    return el;
}

function finishToolMessage(el, result, isError) {
    const resultEl = el.querySelector('.tool-result');
    if (result) {
        resultEl.classList.remove('hidden');
        if (result.includes('---') && result.includes('+++')) {
            renderDiffSafely(resultEl, result);
        } else {
            resultEl.textContent = result;
        }
    }
    el.classList.add(isError ? 'tool-error' : 'tool-success');
}

function renderDiffSafely(container, text) {
    clearChildren(container);
    const lines = text.split('\n');
    for (const line of lines) {
        const span = document.createElement('span');
        if (line.startsWith('+++') || line.startsWith('---')) {
            span.className = 'diff-header';
        } else if (line.startsWith('@@')) {
            span.className = 'diff-hunk';
        } else if (line.startsWith('+')) {
            span.className = 'diff-add';
        } else if (line.startsWith('-')) {
            span.className = 'diff-del';
        }
        span.textContent = line;
        container.appendChild(span);
        container.appendChild(document.createTextNode('\n'));
    }
}

function formatArgs(args) {
    if (typeof args === 'string') return args;
    try {
        const obj = typeof args === 'object' ? args : JSON.parse(args);
        return Object.entries(obj)
            .map(([k, v]) => k + '=' + String(v).substring(0, 50))
            .join(', ')
            .substring(0, 100);
    } catch {
        return String(args).substring(0, 100);
    }
}

// ── Send / Abort ──
async function send() {
    const text = inputEl.value.trim();
    if (!text || generating) return;

    inputEl.value = '';
    inputEl.style.height = 'auto';
    addUserMessage(text);

    try {
        await SendMessage(text);
    } catch (e) {
        addSystemMessage('Send error: ' + e, 'error');
    }
}

async function abort() {
    try {
        await AbortGeneration();
        addSystemMessage('[Aborted]');
        setGenerating(false);
    } catch (e) {
        console.error('Abort error:', e);
    }
}

function setGenerating(val) {
    generating = val;
    btnSend.classList.toggle('hidden', val);
    btnAbort.classList.toggle('hidden', !val);
    statusEl.classList.toggle('hidden', !val);
}

function setStatus(text) {
    statusText.textContent = text;
    statusEl.classList.toggle('hidden', !text);
}

// ── Stats ──
async function updateStats() {
    try {
        const stats = await GetStats();
        if (!stats) return;

        clearChildren(footerStats);

        if (stats.tokensIn > 0) {
            footerStats.appendChild(makeSpan('Up:' + formatTokens(stats.tokensIn), 'stat-token'));
        }
        if (stats.tokensOut > 0) {
            footerStats.appendChild(makeSpan(' Down:' + formatTokens(stats.tokensOut), 'stat-token'));
        }
        if (stats.cost > 0) {
            footerStats.appendChild(makeSpan(' $' + stats.cost.toFixed(3), 'stat-cost'));
        }
        if (stats.contextPercent > 0) {
            const pct = stats.contextPercent;
            const cls = pct > 90 ? 'critical' : pct > 70 ? 'high' : '';
            footerStats.appendChild(
                makeSpan(' ' + pct.toFixed(1) + '%/' + formatTokens(stats.contextWindow), 'stat-context ' + cls)
            );
        }
    } catch {
        // stats not available yet
    }
}

function formatTokens(count) {
    if (count < 1000) return String(count);
    if (count < 10000) return (count / 1000).toFixed(1) + 'k';
    if (count < 1000000) return Math.floor(count / 1000) + 'k';
    return (count / 1000000).toFixed(1) + 'M';
}

// ── Scroll ──
function scrollToBottom() {
    requestAnimationFrame(() => {
        chatEl.scrollTop = chatEl.scrollHeight;
    });
}

// ── Input Events ──
inputEl.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        send();
    }
    if (e.key === 'Escape' && generating) {
        abort();
    }
});

inputEl.addEventListener('input', () => {
    inputEl.style.height = 'auto';
    inputEl.style.height = Math.min(inputEl.scrollHeight, 120) + 'px';
});

btnSend.addEventListener('click', send);
btnAbort.addEventListener('click', abort);

// ── Keyboard shortcuts ──
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.key === 's') {
        e.preventDefault();
        send();
    }
    if (e.ctrlKey && e.key === 'p') {
        e.preventDefault();
        CycleModel().then(m => {
            if (m) {
                modelInfoEl.textContent = m;
                addSystemMessage('Model: ' + m);
            }
        });
    }
    if (e.ctrlKey && e.key === 'n') {
        e.preventDefault();
        NewSession().then(() => {
            clearChildren(messagesEl);
            addSystemMessage('New session started.');
        });
    }
    if (e.ctrlKey && e.key === 'l') {
        e.preventDefault();
        clearChildren(messagesEl);
    }
});

// ── Start ──
init();
