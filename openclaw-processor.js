// OpenClaw Response Processor untuk AmmanGate
// Memproses respon AI dan mengeksekusi command jika ditemukan

const http = require('http');
const { exec } = require('child_process');
const util = require('util');
const execPromise = util.promisify(exec);

const API_BASE = 'http://127.0.0.1:8787/v1';
const OPENCLAW_GATEWAY = 'ws://127.0.0.1:18789';

// Format respon
function formatResponse(type, data) {
  switch(type) {
    case 'status':
      return `📊 STATUS AMMANGATE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏱️  Uptime: ${data.uptime_sec} detik
💻 CPU: ${data.cpu_load}%
🧠 Memory: ${data.mem_used_mb}MB
🔍 Sensors: ${data.sensors ? Object.keys(data.sensors).filter(k => data.sensors[k]).join(', ') : 'N/A'}`;

    case 'devices':
      const online = data.devices ? data.devices.filter(d => d.ip).length : 0;
      if (online === 0) return '📱 Tidak ada perangkat online';
      return `📱 PERANGKAT (${online} online)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
${data.devices.filter(d => d.ip).map(d => `🖥️  ${d.hostname || 'Unknown'} (${d.ip})
   MAC: ${d.mac}
   Risk: ${d.risk_score || 'N/A'}`).join('\n\n')}`;

    case 'alerts':
      if (!data.alerts || data.alerts.length === 0) return '✅ Tidak ada alert aktif';
      return `⚠️  ALERT AKTIF (${data.alerts.length})
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
${data.alerts.map(a => `🚨 [${a.severity || 'HIGH'}] ${a.category || 'Security'}
   ${a.summary || a.message || 'No details'}`).join('\n\n')}`;

    default:
      return JSON.stringify(data, null, 2);
  }
}

function makeRequest(endpoint, method = 'GET', data = null) {
  return new Promise((resolve, reject) => {
    const url = `${API_BASE}${endpoint}`;
    const options = {
      method: method,
      headers: { 'Content-Type': 'application/json' }
    };

    const req = http.request(url, options, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        try {
          resolve(JSON.parse(body));
        } catch (e) {
          resolve({ raw: body });
        }
      });
    });

    req.on('error', reject);

    if (data) {
      req.write(JSON.stringify(data));
    }

    req.end();
  });
}

// Parse pesan untuk command AmmanGate
function parseCommand(text) {
  const t = text.toLowerCase().trim();

  if (/status|cek status/i.test(t)) return ['status'];
  if (/device|perangkat/i.test(t)) return ['devices'];
  if (/alert|serangan|bahaya/i.test(t)) return ['alerts'];
  if (/blocked|diblokir/i.test(t)) return ['blocked'];
  if (/suricata|ids/i.test(t)) return ['suricata'];
  if (/clamav|antivirus/i.test(t)) return ['clamav'];
  if (/filter/i.test(t)) return ['filters'];

  return null;
}

// Eksekusi command AmmanGate
async function executeCommand(command) {
  const handlers = {
    'status': async () => {
      const data = await makeRequest('/system/status');
      return formatResponse('status', data);
    },
    'devices': async () => {
      const data = await makeRequest('/devices?limit=100');
      return formatResponse('devices', data);
    },
    'alerts': async () => {
      const data = await makeRequest('/alerts/active?limit=20');
      return formatResponse('alerts', data);
    },
    'blocked': async () => {
      const data = await makeRequest('/blocked-devices');
      return formatResponse('blocked', data);
    },
    'suricata': async () => {
      const data = await makeRequest('/suricata/status');
      return `🦅 SURICATA IDS\nStatus: ${data.running ? '🟢 Running' : '🔴 Stopped'}`;
    },
    'clamav': async () => {
      const data = await makeRequest('/clamav/status');
      return `🦠 CLAMAV\nStatus: ${data.running ? '🟢 Running' : '🔴 Stopped'}\nVersion: ${data.version || 'Unknown'}`;
    },
    'filters': async () => {
      const data = await makeRequest('/filters');
      return formatResponse('filters', data);
    }
  };

  const handler = handlers[command];
  if (handler) {
    return await handler();
  }
  return null;
}

// Main processing function
async function processMessage(userMessage) {
  const cmd = parseCommand(userMessage);

  if (cmd) {
    console.log(`🎯 Detected AmmanGate command: ${cmd[0]}`);
    const result = await executeCommand(cmd[0]);
    if (result) {
      console.log('✅ Executed AmmanGate command successfully');
      return result;
    }
  }

  return null; // No AmmanGate command detected
}

// CLI interface
if (require.main === module) {
  const message = process.argv[2];

  if (!message) {
    console.log('Usage: node openclaw-processor.js "<message>"');
    console.log('');
    console.log('Examples:');
    console.log('  node openclaw-processor.js "Status AmmanGate"');
    console.log('  node openclaw-processor.js "Ada berapa perangkat?"');
    process.exit(1);
  }

  processMessage(message)
    .then(result => {
      if (result) {
        console.log('\n=== AmmanGate Response ===');
        console.log(result);
      } else {
        console.log('No AmmanGate command detected');
      }
    })
    .catch(err => {
      console.error('Error:', err.message);
    });
}

module.exports = { processMessage, parseCommand, executeCommand };
