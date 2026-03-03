// AmmanGate Telegram Bot - Direct LM Studio Integration
// Command langsung dieksekusi via handler.js
// Pertanyaan lain di-forward ke LM Studio untuk AI response

const { Telegraf } = require('telegraf');
const http = require('http');
const { handleAmmanGate } = require('C:/Users/PC/.openclaw/workspace/skills/ammangate/handler.js');

const BOT_TOKEN = '8724885465:AAFT0n7MMBgKfMUUYstNfPwblFqDEhWgAIA';
const ALLOWED_USER = '756112782';
const LM_STUDIO_API = 'http://127.0.0.1:1234/v1';
const LM_API_KEY = 'sk-lm-pMFYpWRK:HE18W8492Xl459S3WFbX';

const bot = new Telegraf(BOT_TOKEN);

// Logging
function log(msg, data = '') {
  const timestamp = new Date().toLocaleTimeString('id-ID');
  console.log(`[${timestamp}] ${msg}`, data);
}

// Cek apakah pesan adalah command AmmanGate
function isAmmanGateCommand(text) {
  const t = text.toLowerCase().trim();
  const commands = [
    // Status - lebih banyak pattern
    /^status$/, /^\/status$/, /cek status|status sistem/,
    /baca data|lihat data|data terakhir|info sistem|kondisi sistem/,
    /bagaimana sistem|apa kabar sistem|keadaan ammangate/,
    /show status|get status|system info/,

    // Devices
    /^devices$/, /^\/devices$/, /device|perangkat/,
    /ada berapa|berapa perangkat|daftar perangkat|list device/,
    /lihat device|tampilkan device/,

    // Alerts
    /^alerts$/, /^\/alerts$/, /alert|serangan|bahaya/,
    /ada serangan|ada bahaya|ada alert|lihat alert|show alert/,

    // Block
    /^block/, /^\/block/, /blokir|ban/,

    // Unblock
    /^unblock/, /^\/unblock/, /unblokir|buka blokir/,

    // Blocked list
    /^blocked$/, /^\/blocked$/, /diblokir|block list|daftar blokir/,
    /lihat terblokir|device terblokir/,

    // Suricata
    /^suricata$/, /^\/suricata$/, /suricata|ids|intrusion/,

    // ClamAV
    /^clamav$/, /^\/clamav$/, /clamav|antivirus/,

    // Scan
    /^scan/, /^\/scan/, /scan|pindai/,

    // Filters
    /^filters$/, /^\/filters$/, /filter|daftar filter/,

    // Help
    /^help$/, /^\/help$/, /help|bantuan|perintah/
  ];
  return commands.some(r => r.test(t));
}

// Normalisasi command
function normalizeCommand(text) {
  const t = text.toLowerCase().trim().replace(/^\//, '');

  // Status patterns
  if (/^status|cek status|status sistem|baca data|lihat data|data terakhir|info sistem|kondisi sistem|bagaimana sistem|apa kabar sistem|keadaan ammangate/.test(t)) return 'status';

  // Devices patterns
  if (/^devices|device|perangkat|ada berapa|berapa perangkat|daftar perangkat|list device|lihat device|tampilkan device/.test(t)) return 'devices';

  // Alerts patterns
  if (/^alerts|alert|serangan|bahaya|ada serangan|ada bahaya|ada alert|lihat alert|show alert/.test(t)) return 'alerts';

  // Block patterns
  if (/^block|blokir|ban/.test(t) && !/unblokir|unblock/.test(t)) return 'block';

  // Unblock patterns
  if (/^unblock|unblokir|buka blokir/.test(t)) return 'unblock';

  // Blocked patterns
  if (/^blocked|diblokir|block list|daftar blokir|lihat terblokir|device terblokir/.test(t)) return 'blocked';

  // Suricata patterns
  if (/^suricata|suricata|ids|intrusion/.test(t)) return 'suricata';

  // ClamAV patterns
  if (/^clamav|clamav|antivirus/.test(t)) return 'clamav';

  // Scan patterns
  if (/^scan|scan|pindai/.test(t)) return 'scan';

  // Filters patterns
  if (/^filters|filter|daftar filter/.test(t)) return 'filters';

  // Help patterns
  if (/^help|help|bantuan|perintah/.test(t)) return 'help';

  return t;
}

// Ekstrak argumen dari pesan
function extractArgs(text, command) {
  switch(command) {
    case 'block':
    case 'unblock':
      const macMatch = text.match(/([0-9A-Fa-f:]{17}|[\d\.]+)/i);
      return macMatch ? macMatch[1] : '';
    case 'scan':
      const pathMatch = text.match(/scan\s+(.+)/i);
      return pathMatch ? pathMatch[1].trim() : 'C:/Users/PC/Downloads';
    default:
      return '';
  }
}

// Forward ke LM Studio untuk AI response
async function askAI(message, context = '') {
  return new Promise((resolve, reject) => {
    const messages = [
      {
        role: 'system',
        content: 'You are AmmanGate AI assistant. AmmanGate is a home network security system. Answer helpfully and concisely in Indonesian or English based on user preference.'
      }
    ];

    if (context) {
      messages.push({ role: 'assistant', content: context });
    }

    messages.push({
      role: 'user',
      content: message
    });

    const postData = JSON.stringify({
      model: 'huihui-ai_-_qwen2.5-coder-7b-instruct-abliterated',
      messages: messages,
      temperature: 0.7,
      max_tokens: 1000
    });

    const options = {
      hostname: '127.0.0.1',
      port: 1234,
      path: '/v1/chat/completions',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${LM_API_KEY}`,
        'Content-Length': Buffer.byteLength(postData)
      }
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          const response = JSON.parse(data);
          const aiResponse = response?.choices?.[0]?.message?.content
            || response?.message
            || 'Maaf, tidak ada response dari AI.';
          resolve(aiResponse);
        } catch (e) {
          reject(new Error('Parse error: ' + e.message));
        }
      });
    });

    req.on('error', (err) => {
      reject(err);
    });

    req.write(postData);
    req.end();
  });
}

// Middleware: auth
bot.use((ctx, next) => {
  const userId = ctx.from?.id?.toString();
  if (userId === ALLOWED_USER) return next();
  ctx.reply('❌ Maaf, Anda tidak memiliki izin.');
  log(`❌ Unauthorized access attempt: ${userId}`);
});

// Commands - Slash handlers
bot.command('start', async (ctx) => {
  ctx.reply('🛡️ *AmmanGate Bot Online!*\n\n' + await handleAmmanGate('help'));
  log('✅ /start command');
});

bot.command('help', async (ctx) => {
  ctx.reply(await handleAmmanGate('help'));
  log('✅ /help command');
});

bot.command('status', async (ctx) => {
  const result = await handleAmmanGate('status');
  ctx.reply(result);
  log('✅ /status executed');
});

bot.command('devices', async (ctx) => {
  const result = await handleAmmanGate('devices');
  ctx.reply(result);
  log('✅ /devices executed');
});

bot.command('alerts', async (ctx) => {
  const result = await handleAmmanGate('alerts');
  ctx.reply(result);
  log('✅ /alerts executed');
});

bot.command('block', async (ctx) => {
  const mac = ctx.message.text.split(' ').slice(1).join(' ').trim();
  const result = await handleAmmanGate(`block ${mac}`);
  ctx.reply(result);
  log(`✅ /block ${mac}`);
});

bot.command('unblock', async (ctx) => {
  const mac = ctx.message.text.split(' ').slice(1).join(' ').trim();
  const result = await handleAmmanGate(`unblock ${mac}`);
  ctx.reply(result);
  log(`✅ /unblock ${mac}`);
});

bot.command('blocked', async (ctx) => {
  const result = await handleAmmanGate('blocked');
  ctx.reply(result);
  log('✅ /blocked executed');
});

bot.command('suricata', async (ctx) => {
  const result = await handleAmmanGate('suricata');
  ctx.reply(result);
  log('✅ /suricata executed');
});

bot.command('clamav', async (ctx) => {
  const result = await handleAmmanGate('clamav');
  ctx.reply(result);
  log('✅ /clamav executed');
});

bot.command('scan', async (ctx) => {
  const path = ctx.message.text.split(' ').slice(1).join(' ').trim();
  const result = await handleAmmanGate(`scan ${path}`);
  ctx.reply(result);
  log(`✅ /scan ${path}`);
});

bot.command('filters', async (ctx) => {
  const result = await handleAmmanGate('filters');
  ctx.reply(result);
  log('✅ /filters executed');
});

// Main text handler
bot.on('text', async (ctx) => {
  const text = ctx.message.text;
  log(`📩 Message: "${text}"`);

  // Cek apakah ini command AmmanGate
  if (isAmmanGateCommand(text)) {
    const cmd = normalizeCommand(text);
    const args = extractArgs(text, cmd);
    const fullCmd = args ? `${cmd} ${args}` : cmd;

    log(`⚡ Executing AmmanGate: ${fullCmd}`);

    try {
      await ctx.sendChatAction('typing');
      const result = await handleAmmanGate(fullCmd);
      ctx.reply(result);
      log(`✅ Executed: ${cmd}`);
    } catch (err) {
      ctx.reply(`❌ Error: ${err.message}`);
      log(`❌ Error: ${err.message}`);
    }

  } else {
    // Forward ke LM Studio AI
    log('🤖 Forwarding to LM Studio AI...');

    try {
      await ctx.sendChatAction('typing');
      const aiResponse = await askAI(text);
      ctx.reply(aiResponse);
      log('✅ AI response sent');
    } catch (err) {
      ctx.reply(`❌ Error LM Studio: ${err.message}\n\nPastikan LM Studio berjalan di port 1234.`);
      log(`❌ LM Studio error: ${err.message}`);
    }
  }
});

// Error handler
bot.catch((err, ctx) => {
  log(`❌ Bot error: ${err.message}`);
  ctx.reply('❌ Terjadi kesalahan.').catch(() => {});
});

// Start bot
log('🛡️ Starting AmmanGate Bot (Direct LM Studio)...');
log('📱 Commands: /status /devices /alerts /block /unblock /blocked /suricata /clamav /scan /filters');
log('🤖 Other messages → LM Studio AI');

bot.launch();
log('✅ Bot started! Listening for messages...');

// Graceful shutdown
process.once('SIGINT', () => {
  log('🛑 SIGINT received');
  bot.stop('SIGINT');
});
process.once('SIGTERM', () => {
  log('🛑 SIGTERM received');
  bot.stop('SIGTERM');
});
