// Global State
let socket = null;
let authToken = null;
let queryHistory = [];

// DOM Elements
const authOverlay = document.getElementById('auth-overlay');
const loginForm = document.getElementById('login-form');
const btnSkipAuth = document.getElementById('btn-skip-auth');
const navBtns = document.querySelectorAll('.nav-btn');
const viewSections = document.querySelectorAll('.view-section');
const wsStatusDot = document.getElementById('ws-status-dot');
const wsStatusText = document.getElementById('ws-status-text');
const chatForm = document.getElementById('chat-form');
const chatInput = document.getElementById('chat-input');
const chatMessages = document.getElementById('chat-messages');
const dbSelect = document.getElementById('db-select');
const queryHistoryBody = document.getElementById('query-history-body');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    // Event Listeners
    loginForm.addEventListener('submit', handleLogin);
    btnSkipAuth.addEventListener('click', skipAuth);
    chatForm.addEventListener('submit', handleChatSubmit);
    
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

// Auth Handlers
async function handleLogin(e) {
    e.preventDefault();
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    
    // Note: This relies on the backend existing /api/v1/user/login endpoint
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
            connectWebSocket();
        } else {
            alert('Login failed: ' + (data.message || 'Unknown error'));
        }
    } catch (err) {
        console.error('Login error:', err);
        // Fallback for demo
        skipAuth();
    }
}

function skipAuth() {
    authOverlay.classList.remove('active');
    connectWebSocket();
}

// WebSocket Connection
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    // For local dev where server is running on 5000, fallback to hardcoded if not served from go
    let wsUrl = `${protocol}//${host}/api/v1/chat`;
    if(host === '' || host.includes('5500')){
       wsUrl = `ws://localhost:5000/api/v1/chat`; 
    }

    if (authToken) {
        wsUrl += `?token=${authToken}`;
    }

    try {
        socket = new WebSocket(wsUrl);

        socket.onopen = () => {
            wsStatusDot.className = 'status-indicator online';
            wsStatusText.textContent = 'Connected';
            
            // Send init payload based on selected DB
            const initPayload = {
                type: "start",
                payload: {
                    db_type: dbSelect.value,
                    db_name: "example_db",
                    db_url: "example_url"
                },
                timestamp: new Date().toISOString()
            };
            socket.send(JSON.stringify(initPayload));
        };

        socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            appendSystemMessage(data);
            updateDashboard(data);
        };

        socket.onerror = (error) => {
            console.error('WebSocket Error:', error);
            wsStatusDot.className = 'status-indicator offline';
            wsStatusText.textContent = 'Error';
        };

        socket.onclose = () => {
            wsStatusDot.className = 'status-indicator offline';
            wsStatusText.textContent = 'Disconnected';
            setTimeout(connectWebSocket, 5000); // Reconnect attempt
        };
    } catch (e) {
        console.error("Failed to connect WS", e);
    }
}

// Chat Handlers
function handleChatSubmit(e) {
    e.preventDefault();
    const text = chatInput.value.trim();
    if (!text || !socket || socket.readyState !== WebSocket.OPEN) return;

    // Show User Message
    appendUserMessage(text);
    
    // Add to History
    queryHistory.push({
        timestamp: new Date(),
        question: text,
        sql: 'Processing...',
        status: 'pending'
    });
    renderHistory();

    // Send via WS
    const payload = {
        type: "chat",
        payload: { question: text },
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
    // Find last pending query
    const lastPending = [...queryHistory].reverse().find(q => q.status === 'pending');
    if (lastPending) {
        lastPending.status = 'success';
        if (data.payload?.sql) lastPending.sql = data.payload.sql;
        renderHistory();
    }
    
    document.getElementById('stat-total-queries').textContent = queryHistory.length;
}

function renderHistory() {
    if (queryHistory.length === 0) return;
    
    queryHistoryBody.innerHTML = '';
    
    [...queryHistory].reverse().slice(0, 10).forEach(q => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${q.timestamp.toLocaleTimeString()}</td>
            <td>${escapeHtml(q.question)}</td>
            <td><code style="background: rgba(0,0,0,0.3); padding: 2px 4px; border-radius: 4px;">${escapeHtml(q.sql)}</code></td>
            <td><span class="badge ${q.status === 'success' ? 'success' : ''}">${q.status}</span></td>
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
    
    dataArray.slice(0, 5).forEach(row => { // Limit to 5 rows for chat
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
