// Explicit Client Session State Machine
const ClientState = Object.freeze({
    AWAITING_AUTH: 'AWAITING_AUTH',
    AWAITING_DB_CONFIG: 'AWAITING_DB_CONFIG',
    DISCONNECTED: 'DISCONNECTED',
    CONNECTING: 'CONNECTING',
    INITIALIZING_DB: 'INITIALIZING_DB',
    READY: 'READY',
    EXECUTING_QUERY: 'EXECUTING_QUERY',
    ERROR: 'ERROR'
});

// Global State
let socket = null;
let authToken = null;
let sessionState = ClientState.AWAITING_AUTH;
let queryHistory = [];
let reconnectTimer = null;

// Explicit DB Connection Configuration State (must be provided before WebSocket connects)
let dbConnectionConfig = {
    configured: false,
    useServerDB: false,
    dbType: 'postgres',
    dbName: '',
    dbUrl: '',
    llmType: 'muse'
};

// DOM Elements
const authOverlay = document.getElementById('auth-overlay');
const dbConfigOverlay = document.getElementById('db-config-overlay');
const loginForm = document.getElementById('login-form');
const dbConfigForm = document.getElementById('db-config-form');
const useServerDbCheckbox = document.getElementById('use-server-db');
const customDbFields = document.getElementById('custom-db-fields');
const cfgDbType = document.getElementById('cfg-db-type');
const cfgDbName = document.getElementById('cfg-db-name');
const cfgDbUrl = document.getElementById('cfg-db-url');
const cfgLlmType = document.getElementById('cfg-llm-type');
const btnReconfigureDb = document.getElementById('btn-reconfigure-db');

const btnSkipAuth = document.getElementById('btn-skip-auth');
const navBtns = document.querySelectorAll('.nav-btn');
const viewSections = document.querySelectorAll('.view-section');
const wsStatusDot = document.getElementById('ws-status-dot');
const wsStatusText = document.getElementById('ws-status-text');
const chatForm = document.getElementById('chat-form');
const chatInput = document.getElementById('chat-input');
const chatMessages = document.getElementById('chat-messages');
const dbSelect = document.getElementById('db-select');
const llmSelect = document.getElementById('llm-select');
const queryHistoryBody = document.getElementById('query-history-body');
const statLatency = document.getElementById('stat-latency');
const refreshDashboardBtn = document.getElementById('refresh-dashboard');

function transitionSessionState(nextState, statusLabel) {
    sessionState = nextState;
    if (!wsStatusDot || !wsStatusText) return;

    switch (nextState) {
        case ClientState.READY:
        case ClientState.EXECUTING_QUERY:
            wsStatusDot.className = 'status-indicator online';
            wsStatusText.textContent = statusLabel || (nextState === ClientState.EXECUTING_QUERY ? 'Running Query...' : 'Connected');
            break;
        case ClientState.CONNECTING:
        case ClientState.INITIALIZING_DB:
            wsStatusDot.className = 'status-indicator online';
            wsStatusText.textContent = statusLabel || 'Initializing DB...';
            break;
        case ClientState.AWAITING_DB_CONFIG:
            wsStatusDot.className = 'status-indicator offline';
            wsStatusText.textContent = statusLabel || 'Awaiting DB Config';
            break;
        case ClientState.ERROR:
            wsStatusDot.className = 'status-indicator offline';
            wsStatusText.textContent = statusLabel || 'Error';
            break;
        default:
            wsStatusDot.className = 'status-indicator offline';
            wsStatusText.textContent = statusLabel || 'Disconnected';
            break;
    }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    // Event Listeners
    loginForm.addEventListener('submit', handleLogin);
    btnSkipAuth.addEventListener('click', skipAuth);
    chatForm.addEventListener('submit', handleChatSubmit);

    if (dbConfigForm) {
        dbConfigForm.addEventListener('submit', handleDbConfigSubmit);
    }

    if (useServerDbCheckbox && customDbFields) {
        useServerDbCheckbox.addEventListener('change', () => {
            const useServer = useServerDbCheckbox.checked;
            customDbFields.style.opacity = useServer ? '0.45' : '1';
            customDbFields.style.pointerEvents = useServer ? 'none' : 'auto';
        });
    }

    if (btnReconfigureDb) {
        btnReconfigureDb.addEventListener('click', openDbConfigModal);
    }

    if (dbSelect) {
        dbSelect.addEventListener('change', () => {
            dbConnectionConfig.dbType = dbSelect.value;
            if (cfgDbType) cfgDbType.value = dbSelect.value;
            if (socket && socket.readyState === WebSocket.OPEN && dbConnectionConfig.configured) {
                sendStartHandshake();
            }
        });
    }

    if (llmSelect) {
        llmSelect.addEventListener('change', () => {
            dbConnectionConfig.llmType = llmSelect.value;
            if (cfgLlmType) cfgLlmType.value = llmSelect.value;
        });
    }

    if (refreshDashboardBtn) {
        refreshDashboardBtn.addEventListener('click', () => {
            renderHistory();
            updateMetrics();
        });
    }
    
    // Auto-resize textarea
    chatInput.addEventListener('input', function() {
        this.style.height = 'auto';
        this.style.height = (this.scrollHeight) + 'px';
    });

    // Navigation
    navBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const target = btn.dataset.target;
            
            navBtns.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            
            viewSections.forEach(v => v.classList.remove('active'));
            document.getElementById(target).classList.add('active');
        });
    });

    // Enter key to submit
    chatInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleChatSubmit(e);
        }
    });
});

// Step 1: Auth Handlers -> Transitions to Step 2 (DB Configuration Modal) before connecting to WebSocket
async function handleLogin(e) {
    e.preventDefault();
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    
    try {
        const response = await fetch('/api/v1/user/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });
        
        const data = await response.json();
        if (response.ok) {
            authToken = data.token || data.data?.access_token;
            authOverlay.classList.remove('active');
            openDbConfigModal();
        } else {
            alert('Login failed: ' + (data.message || 'Unknown error'));
        }
    } catch (err) {
        console.error('Login error:', err);
        skipAuth();
    }
}

function skipAuth() {
    authOverlay.classList.remove('active');
    openDbConfigModal();
}

function openDbConfigModal() {
    transitionSessionState(ClientState.AWAITING_DB_CONFIG, 'Awaiting DB Config');
    if (dbConfigOverlay) {
        dbConfigOverlay.classList.add('active');
    }
}

// Step 2: Validate user DB configuration (or server DB leg) before opening WebSocket connection
function handleDbConfigSubmit(e) {
    e.preventDefault();
    const useServerDB = useServerDbCheckbox ? useServerDbCheckbox.checked : false;
    const selectedDbType = cfgDbType ? cfgDbType.value : 'postgres';
    const enteredDbName = cfgDbName ? cfgDbName.value.trim() : '';
    const enteredDbUrl = cfgDbUrl ? cfgDbUrl.value.trim() : '';
    const selectedLlmType = cfgLlmType ? cfgLlmType.value : 'muse';

    if (!useServerDB && (!enteredDbName || !enteredDbUrl)) {
        alert('Please provide both the Database Name (db_name) and Connection URL (db_url), or enable "Query Server Host Database".');
        return;
    }

    dbConnectionConfig = {
        configured: true,
        useServerDB: useServerDB,
        dbType: selectedDbType,
        dbName: enteredDbName,
        dbUrl: enteredDbUrl,
        llmType: selectedLlmType
    };

    if (dbSelect) dbSelect.value = selectedDbType;
    if (llmSelect) llmSelect.value = selectedLlmType;
    if (dbConfigOverlay) dbConfigOverlay.classList.remove('active');

    // If a socket is already open, send the new "start" handshake immediately; otherwise establish the WebSocket connection now
    if (socket && socket.readyState === WebSocket.OPEN) {
        sendStartHandshake();
    } else {
        connectWebSocket();
    }
}

function sendStartHandshake() {
    if (!socket || socket.readyState !== WebSocket.OPEN || !dbConnectionConfig.configured) return;
    transitionSessionState(ClientState.INITIALIZING_DB, 'Initializing DB...');
    const initPayload = {
        type: "start",
        payload: {
            db_type: dbConnectionConfig.dbType,
            db_name: dbConnectionConfig.dbName,
            db_url: dbConnectionConfig.dbUrl,
            llm_type: llmSelect ? llmSelect.value : dbConnectionConfig.llmType,
            use_server_db: dbConnectionConfig.useServerDB
        },
        timestamp: new Date().toISOString()
    };
    socket.send(JSON.stringify(initPayload));
}

// Step 3: WebSocket Connection (only initiated after DB config is provided)
function connectWebSocket() {
    if (!dbConnectionConfig.configured) {
        openDbConfigModal();
        return;
    }

    if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    let wsUrl = `${protocol}//${host}/api/v1/chat`;
    if (host === '' || host.includes('5500')) {
        wsUrl = `ws://localhost:5000/api/v1/chat`;
    }

    if (authToken) {
        wsUrl += `?token=${encodeURIComponent(authToken)}`;
    }

    transitionSessionState(ClientState.CONNECTING, 'Connecting...');

    try {
        socket = new WebSocket(wsUrl);

        socket.onopen = () => {
            sendStartHandshake();
        };

        socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            handleServerMessage(data);
        };

        socket.onerror = (error) => {
            console.error('WebSocket Error:', error);
            transitionSessionState(ClientState.ERROR, 'Error');
        };

        socket.onclose = () => {
            transitionSessionState(ClientState.DISCONNECTED, 'Disconnected');
            if (dbConnectionConfig.configured) {
                reconnectTimer = setTimeout(connectWebSocket, 5000);
            }
        };
    } catch (e) {
        console.error("Failed to connect WS", e);
        transitionSessionState(ClientState.ERROR, 'Error');
    }
}

function handleServerMessage(data) {
    if (data.type === 'start_response') {
        if (data.status === 'success') {
            transitionSessionState(ClientState.READY, 'Connected');
        } else {
            transitionSessionState(ClientState.ERROR, 'DB Init Failed');
        }
        appendSystemMessage(data);
        return;
    }

    transitionSessionState(ClientState.READY, 'Connected');
    appendSystemMessage(data);
    updateDashboard(data);
}

// Chat Handlers
function handleChatSubmit(e) {
    e.preventDefault();
    if (!dbConnectionConfig.configured) {
        openDbConfigModal();
        return;
    }
    const text = chatInput.value.trim();
    if (!text || !socket || socket.readyState !== WebSocket.OPEN) return;

    // Show User Message
    appendUserMessage(text);
    
    // Add to History with explicit pending state and start timestamp
    queryHistory.push({
        timestamp: new Date(),
        startedAtMs: performance.now(),
        latencyMs: null,
        question: text,
        sql: 'Generating SQL...',
        status: 'pending'
    });
    renderHistory();

    transitionSessionState(ClientState.EXECUTING_QUERY, 'Running Query...');

    // Send via WS
    const payload = {
        type: "chat",
        payload: {
            question: text,
            llm_type: llmSelect ? llmSelect.value : dbConnectionConfig.llmType
        },
        timestamp: new Date().toISOString()
    };
    socket.send(JSON.stringify(payload));

    chatInput.value = '';
    chatInput.style.height = 'auto';
}

function appendUserMessage(text) {
    const div = document.createElement('div');
    div.className = 'message user';
    div.innerHTML = `
        <div class="avatar"><i data-lucide="user"></i></div>
        <div class="bubble"><p>${escapeHtml(text)}</p></div>
    `;
    chatMessages.appendChild(div);
    lucide.createIcons({root: div});
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

function appendSystemMessage(data) {
    const div = document.createElement('div');
    div.className = 'message system';
    
    let content = `<p>${escapeHtml(data.message || data.payload?.response || JSON.stringify(data))}</p>`;
    
    if (data.payload?.sql) {
        content += `<div class="sql-block">${escapeHtml(data.payload.sql)}</div>`;
    }

    if (data.payload?.remediation_hint && data.status === 'error') {
        content += `<p style="font-size: 0.85rem; opacity: 0.85; margin-top: 0.4rem;"><strong>Hint:</strong> ${escapeHtml(data.payload.remediation_hint)}</p>`;
    }

    if (data.payload?.data && Array.isArray(data.payload.data) && data.payload.data.length > 0) {
        content += createTableFromData(data.payload.data);
    }

    div.innerHTML = `
        <div class="avatar"><i data-lucide="bot"></i></div>
        <div class="bubble">${content}</div>
    `;
    chatMessages.appendChild(div);
    lucide.createIcons({root: div});
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Dashboard Handlers
function updateDashboard(data) {
    const lastPending = [...queryHistory].reverse().find(q => q.status === 'pending');
    if (lastPending) {
        const isError = data.status === 'error';
        lastPending.status = isError ? 'error' : 'success';
        lastPending.latencyMs = data.payload?.latency_ms || Math.round(performance.now() - lastPending.startedAtMs);
        if (data.payload?.sql) {
            lastPending.sql = data.payload.sql;
        } else if (isError) {
            lastPending.sql = 'Rejected / Failed';
        } else {
            lastPending.sql = 'SELECT (Executed)';
        }
        renderHistory();
    }
    
    updateMetrics();
}

function updateMetrics() {
    document.getElementById('stat-total-queries').textContent = queryHistory.length;
    const completedWithLatency = queryHistory.filter(q => typeof q.latencyMs === 'number' && q.latencyMs >= 0);
    if (statLatency && completedWithLatency.length > 0) {
        const avg = Math.round(completedWithLatency.reduce((acc, q) => acc + q.latencyMs, 0) / completedWithLatency.length);
        statLatency.textContent = `${avg}ms`;
    }
}

function renderHistory() {
    if (queryHistory.length === 0) return;
    
    queryHistoryBody.innerHTML = '';
    
    [...queryHistory].reverse().slice(0, 10).forEach(q => {
        const tr = document.createElement('tr');
        const badgeClass = q.status === 'success' ? 'success' : (q.status === 'error' ? 'error' : '');
        tr.innerHTML = `
            <td>${q.timestamp.toLocaleTimeString()}</td>
            <td>${escapeHtml(q.question)}</td>
            <td><code style="background: rgba(0,0,0,0.3); padding: 2px 4px; border-radius: 4px;">${escapeHtml(q.sql)}</code></td>
            <td><span class="badge ${badgeClass}">${escapeHtml(q.status)}</span></td>
        `;
        queryHistoryBody.appendChild(tr);
    });
}

// Utilities
function escapeHtml(unsafe) {
    if (!unsafe) return '';
    return unsafe
         .toString()
         .replace(/&/g, "&amp;")
         .replace(/</g, "&lt;")
         .replace(/>/g, "&gt;")
         .replace(/"/g, "&quot;")
         .replace(/'/g, "&#039;");
}

function createTableFromData(dataArray) {
    if (!dataArray || dataArray.length === 0) return '';
    
    const headers = Object.keys(dataArray[0]);
    let tableHtml = '<table class="data-table"><thead><tr>';
    
    headers.forEach(h => tableHtml += `<th>${escapeHtml(h)}</th>`);
    tableHtml += '</tr></thead><tbody>';
    
    dataArray.slice(0, 5).forEach(row => {
        tableHtml += '<tr>';
        headers.forEach(h => {
            let val = row[h];
            if (typeof val === 'object') val = JSON.stringify(val);
            tableHtml += `<td>${escapeHtml(val)}</td>`;
        });
        tableHtml += '</tr>';
    });
    
    tableHtml += '</tbody></table>';
    if (dataArray.length > 5) {
        tableHtml += `<p style="font-size: 0.8rem; color: var(--text-secondary); margin-top: 0.5rem;">Showing 5 of ${dataArray.length} rows</p>`;
    }
    return tableHtml;
}
