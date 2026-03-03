// AmmanGate Telegram Proxy Bot
// Menangani pesan Telegram langsung tanpa perlu AI function calling

const { Telegraf } = require('telegraf');
const http = require('http');

const API_BASE = 'http://127.0.0.1:8787/v1';
const BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN || 'YOUR_TELEGRAM_BOT_TOKEN_HERE';
const ALLOWED_USER = process.env.ALLOWED_USER_ID || 'YOUR_TELEGRAM_USER_ID_HERE';

const bot = new Telegraf(BOT_TOKEN);

// Format respon
function formatResponse(type, data) {
  switch(type) {
    case 'status':
      return `📊 STATUS AMMANGATE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏱️  Uptime: ${data.uptime_sec} detik
💻 CPU: ${data.cpu_load}%
🧠 Memory: ${data.mem_used_mb}MB
🔍 Sensors: ${data.sensors ? Object.keys(data.sensors).filter(k => data.sensors[k]).join(', ') : 'N/A'}
⏰ Last Event: ${data.last_event_ts || 'N/A'}`;

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
   ${a.summary || a.message || 'No details'}
   ${a.timestamp || ''}`).join('\n\n')}`;

    case 'blocked':
      if (!data.blocked_devices || data.blocked_devices.length === 0) return '🔓 Tidak ada perangkat diblokir';
      return `🚫 PERANGKAT DIBLOKIR (${data.blocked_devices.length})
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
${data.blocked_devices.map(d => `🔒 ${d.mac}
   Reason: ${d.reason || 'Unknown'}
   Since: ${d.blocked_at || ''}`).join('\n\n')}`;

    case 'suricata':
      return `🦅 SURICATA IDS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: ${data.running ? '🟢 Running' : '🔴 Stopped'}
${data.eve_log_path ? `Log: ${data.eve_log_path}` : ''}`;

    case 'clamav':
      return `🦠 CLAMAV ANTIVIRUS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: ${data.running ? '🟢 Running' : '🔴 Stopped'}
Version: ${data.version || 'Unknown'}
DB Version: ${data.db_version || 'Unknown'}`;

    case 'scan':
      return `🔍 CLAMAV SCAN RESULT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Path: ${data.path || 'Unknown'}
Infected: ${data.infected_count || 0} files
Scanned: ${data.scanned_count || 0} files`;

    case 'filters':
      if (!data.filters || data.filters.length === 0) return '🔽 Tidak ada filter aktif';
      return `🔽 FILTER AKTIF (${data.filters.length})
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
${data.filters.map(f => `${f.enabled ? '✅' : '❌'} ${f.name}
   Type: ${f.type}
   Value: ${f.value}`).join('\n\n')}`;

    default:
      return JSON.stringify(data, null, 2);
  }
}

// HTTP Request helper
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

// Command handlers
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

  'block': async (mac) => {
    if (!mac) return '❌ Error: MAC address diperlukan\nContoh: /block AA:BB:CC:DD:EE:FF';
    const data = await makeRequest('/block-device', 'POST', {
      mac,
      reason: 'Blocked via Telegram',
      duration: 'permanent'
    });
    return `🔒 Perangkat ${mac} berhasil diblokir`;
  },

  'unblock': async (mac) => {
    if (!mac) return '❌ Error: MAC address diperlukan\nContoh: /unblock AA:BB:CC:DD:EE:FF';
    await makeRequest(`/block-device/${mac}`, 'DELETE');
    return `🔓 Perangkat ${mac} berhasil di-unblock`;
  },

  'blocked': async () => {
    const data = await makeRequest('/blocked-devices');
    return formatResponse('blocked', data);
  },

  'suricata': async () => {
    const data = await makeRequest('/suricata/status');
    return formatResponse('suricata', data);
  },

  'clamav': async () => {
    const data = await makeRequest('/clamav/status');
    return formatResponse('clamav', data);
  },

  'scan': async (path = 'C:/Users/PC/Downloads') => {
    const data = await makeRequest('/clamav/scan', 'POST', { path, recursive: true });
    return formatResponse('scan', data);
  },

  'filters': async () => {
    const data = await makeRequest('/filters');
    return formatResponse('filters', data);
  },

  'help': async () => {
    return `🛡️ AMMANGATE COMMANDS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Commands:
/status - Cek status sistem
/devices - Lihat daftar perangkat
/alerts - Lihat alert aktif
/block <mac> - Blokir perangkat
/unblock <mac> - Unblock perangkat
/blocked - Lihat perangkat diblokir
/suricata - Status Suricata IDS
/clamav - Status ClamAV
/scan - Scan dengan ClamAV
/filters - Lihat semua filter

Contoh penggunaan:
/status - Cek status
/block AA:BB:CC:DD:EE:FF - Blokir device
/scan C:/Users/PC/Downloads - Scan folder`;
  }
};

// Parse pesan user
function parseCommand(text) {
  const t = text.toLowerCase().trim();

  // Commands with /
  if (text.startsWith('/')) {
    const parts = text.substring(1).split(' ');
    return [parts[0], ...parts.slice(1)];
  }

  // Status commands
  if (/status|cek status|bagaimana sistem/.test(t)) return ['status'];

  // Device commands
  if (/device|perangkat|ada berapa/.test(t)) return ['devices'];

  // Alert commands
  if (/alert|serangan|bahaya/.test(t)) return ['alerts'];

  // Block commands
  const blockMatch = text.match(/block\s+([0-9A-Fa-f:]{17}|[\d\.]+)/i);
  if (blockMatch || (/blokir/i.test(t) && !/unblokir|unblock/i.test(t))) return ['block', blockMatch ? blockMatch[1] : null];

  // Unblock commands
  const unblockMatch = text.match(/unblock\s+([0-9A-Fa-f:]{17}|[\d\.]+)/i);
  if (unblockMatch || /unblokir|buka blokir/i.test(t)) return ['unblock', unblockMatch ? unblockMatch[1] : null];

  // Blocked list
  if (/blocked|diblokir|block list|daftar blokir/i.test(t)) return ['blocked'];

  // Suricata
  if (/suricata|ids|intrusion/i.test(t)) return ['suricata'];

  // ClamAV
  if (/clamav|antivirus/i.test(t)) return ['clamav'];

  // Scan
  if (/scan|pindai/i.test(t)) return ['scan'];

  // Filters
  if (/filter|daftar filter/i.test(t)) return ['filters'];

  // Help
  if (/help|bantuan|perintah/i.test(t)) return ['help'];

  return null;
}

// Middleware: hanya izinkan user tertentu
bot.use((ctx, next) => {
  if (ctx.from && ctx.from.id.toString() === ALLOWED_USER) {
    return next();
  }
  ctx.reply('❌ Maaf, Anda tidak memiliki izin.');
});

// Command handlers
bot.command(['start', 'help'], async (ctx) => {
  ctx.reply(await handlers.help());
});

bot.command('status', async (ctx) => {
  ctx.reply(await handlers.status());
});

bot.command('devices', async (ctx) => {
  ctx.reply(await handlers.devices());
});

bot.command('alerts', async (ctx) => {
  ctx.reply(await handlers.alerts());
});

bot.command('block', async (ctx) => {
  const mac = ctx.message.text.split(' ')[1];
  ctx.reply(await handlers.block(mac));
});

bot.command('unblock', async (ctx) => {
  const mac = ctx.message.text.split(' ')[1];
  ctx.reply(await handlers.unblock(mac));
});

bot.command('blocked', async (ctx) => {
  ctx.reply(await handlers.blocked());
});

bot.command('suricata', async (ctx) => {
  ctx.reply(await handlers.suricata());
});

bot.command('clamav', async (ctx) => {
  ctx.reply(await handlers.clamav());
});

bot.command('scan', async (ctx) => {
  const path = ctx.message.text.split(' ')[1];
  ctx.reply(await handlers.scan(path));
});

bot.command('filters', async (ctx) => {
  ctx.reply(await handlers.filters());
});

// Handle all text messages
bot.on('text', async (ctx) => {
  const cmd = parseCommand(ctx.message.text);
  if (!cmd) {
    ctx.reply('❌ Perintah tidak dikenali.\n\n' + await handlers.help());
    return;
  }

  const [command, ...args] = cmd;
  const handler = handlers[command];

  if (!handler) {
    ctx.reply('❌ Perintah tidak dikenali.\n\n' + await handlers.help());
    return;
  }

  try {
    const result = await handler(...args);
    ctx.reply(result);
  } catch (error) {
    ctx.reply(`❌ Error: ${error.message}`);
  }
});

// Error handler
bot.catch((err, ctx) => {
  console.error('Telegram bot error:', err);
  ctx.reply('❌ Terjadi kesalahan.');
});

// Start bot
console.log('🤖 AmmanGate Telegram Proxy Bot starting...');
bot.launch()
  .then(() => {
    console.log('✅ Bot started successfully!');
    console.log(`📱 Allowed user: ${ALLOWED_USER}`);
  })
  .catch(err => {
    console.error('❌ Failed to start bot:', err);
  });

// Enable graceful stop
process.once('SIGINT', () => bot.stop('SIGINT'));
process.once('SIGTERM', () => bot.stop('SIGTERM'));
