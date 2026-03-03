// AmmanGate Hybrid Bot
// Menggabungkan eksekusi command langsung dengan AI response dari OpenClaw
//
// Flow:
// 1. User mengirim pesan ke Telegram
// 2. Bot cek apakah ini command AmmanGate
// 3. Jika ya: eksekusi langsung via handler.js
// 4. Jika tidak: forward ke OpenClaw untuk AI response

const { Telegraf } = require('telegraf');
const http = require('http');
const { handleAmmanGate, parseCommand } = require('C:/Users/PC/.openclaw/workspace/skills/ammangate/handler.js');

const BOT_TOKEN = '8724885465:AAFT0n7MMBgKfMUUYstNfPwblFqDEhWgAIA';
const ALLOWED_USER = '756112782';
const OPENCLAW_API = 'http://127.0.0.1:18789';

const bot = new Telegraf(BOT_TOKEN);

// Ambil handler command dari AmmanGate
function getCommandFromText(text) {
  const t = text.toLowerCase().trim();

  // Cek command yang dimengerti oleh AmmanGate
  if (/^\/status$|^status|cek status|bagaimana sistem/i.test(t)) return 'status';
  if (/^\/devices$|^device|perangkat|ada berapa/i.test(t)) return 'devices';
  if (/^\/alerts$|^alert|serangan|bahaya/i.test(t)) return 'alerts';
  if (/^\/block|block|blokir|ban/i.test(t)) return 'block';
  if (/^\/unblock|^unblock|unblokir|buka blokir/i.test(t)) return 'unblock';
  if (/^\/blocked|^blocked|diblokir|block list|daftar blokir/i.test(t)) return 'blocked';
  if (/^\/suricata$|^suricata|ids|intrusion/i.test(t)) return 'suricata';
  if (/^\/clamav$|^clamav|antivirus/i.test(t)) return 'clamav';
  if (/^\/scan|^scan|pindai/i.test(t)) return 'scan';
  if (/^\/filters$|^filter|daftar filter/i.test(t)) return 'filters';
  if (/^\/help$|^help|bantuan|perintah/i.test(t)) return 'help';

  return null;
}

// Ekstrak MAC address dari pesan untuk block/unblock
function extractMacAddress(text) {
  const match = text.match(/([0-9A-Fa-f:]{17}|[\d\.]+)/i);
  return match ? match[1] : null;
}

// Ekstrak path dari pesan untuk scan
function extractPath(text) {
  const match = text.match(/scan\s+(.+)/i);
  return match ? match[1].trim() : 'C:/Users/PC/Downloads';
}

// Forward pesan ke OpenClaw untuk AI response
function forwardToOpenClaw(message) {
  return new Promise((resolve, reject) => {
    const postData = JSON.stringify({
      channel: 'telegram',
      text: message
    });

    const options = {
      hostname: '127.0.0.1',
      port: 18789,
      path: '/v1/messages',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
        'Authorization': 'Bearer e79c8288a4438353ec4f38c570aac38427864458394aaa68'
      }
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          const response = JSON.parse(data);
          // Extract AI response
          const aiResponse = response?.choices?.[0]?.message?.content
            || response?.message
            || response?.text
            || 'Maaf, tidak ada response dari AI.';
          resolve(aiResponse);
        } catch (e) {
          resolve('Error parsing AI response.');
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

// Middleware: hanya izinkan user tertentu
bot.use((ctx, next) => {
  if (ctx.from && ctx.from.id.toString() === ALLOWED_USER) {
    return next();
  }
  ctx.reply('❌ Maaf, Anda tidak memiliki izin.');
});

// Handler utama
bot.on('text', async (ctx) => {
  const text = ctx.message.text;
  const command = getCommandFromText(text);

  console.log(`📩 Message: "${text}"`);
  console.log(`🎯 Command detected: ${command || 'none'}`);

  // Jika ini command AmmanGate, eksekusi langsung
  if (command) {
    console.log(`⚡ Executing AmmanGate command: ${command}`);

    try {
      let result;

      switch(command) {
        case 'block':
          const macBlock = extractMacAddress(text);
          result = await handleAmmanGate(`block ${macBlock || ''}`);
          break;
        case 'unblock':
          const macUnblock = extractMacAddress(text);
          result = await handleAmmanGate(`unblock ${macUnblock || ''}`);
          break;
        case 'scan':
          const scanPath = extractPath(text);
          result = await handleAmmanGate(`scan ${scanPath}`);
          break;
        default:
          result = await handleAmmanGate(command);
      }

      ctx.reply(result);
      console.log('✅ Command executed successfully');

    } catch (error) {
      ctx.reply(`❌ Error: ${error.message}`);
      console.error('❌ Command error:', error);
    }

  } else {
    // Jika bukan command, forward ke OpenClaw untuk AI response
    console.log('🤖 Forwarding to OpenClaw AI...');

    try {
      // Tampilkan indikator typing
      await ctx.sendChatAction('typing');

      // Dapatkan response dari AI
      const aiResponse = await forwardToOpenClaw(text);
      ctx.reply(aiResponse);
      console.log('✅ AI response sent');

    } catch (error) {
      ctx.reply(`❌ Error menghubungi OpenClaw: ${error.message}\n\nPastikan OpenClaw Gateway berjalan.`);
      console.error('❌ OpenClaw error:', error);
    }
  }
});

// Command handlers eksplisit untuk slash commands
bot.command(['start', 'help'], async (ctx) => {
  ctx.reply(await handleAmmanGate('help'));
});

bot.command('status', async (ctx) => {
  ctx.reply(await handleAmmanGate('status'));
});

bot.command('devices', async (ctx) => {
  ctx.reply(await handleAmmanGate('devices'));
});

bot.command('alerts', async (ctx) => {
  ctx.reply(await handleAmmanGate('alerts'));
});

bot.command('block', async (ctx) => {
  const mac = ctx.message.text.split(' ')[1];
  ctx.reply(await handleAmmanGate(`block ${mac || ''}`));
});

bot.command('unblock', async (ctx) => {
  const mac = ctx.message.text.split(' ')[1];
  ctx.reply(await handleAmmanGate(`unblock ${mac || ''}`));
});

bot.command('blocked', async (ctx) => {
  ctx.reply(await handleAmmanGate('blocked'));
});

bot.command('suricata', async (ctx) => {
  ctx.reply(await handleAmmanGate('suricata'));
});

bot.command('clamav', async (ctx) => {
  ctx.reply(await handleAmmanGate('clamav'));
});

bot.command('scan', async (ctx) => {
  const path = ctx.message.text.split(' ').slice(1).join(' ');
  ctx.reply(await handleAmmanGate(`scan ${path}`));
});

bot.command('filters', async (ctx) => {
  ctx.reply(await handleAmmanGate('filters'));
});

// Error handler
bot.catch((err, ctx) => {
  console.error('Bot error:', err);
  ctx.reply('❌ Terjadi kesalahan.');
});

// Start bot
console.log('🛡️ AmmanGate Hybrid Bot starting...');
console.log('📱 Command langsung di-handle oleh AmmanGate handler.js');
console.log('🤖 Pesan lain di-forward ke OpenClaw AI');
console.log('');

bot.launch()
  .then(() => {
    console.log('✅ Bot started successfully!');
    console.log(`👤 Allowed user: ${ALLOWED_USER}`);
    console.log('');
    console.log('📌 Commands:');
    console.log('   /status, /devices, /alerts, /block, /unblock, /blocked');
    console.log('   /suricata, /clamav, /scan, /filters, /help');
    console.log('');
    console.log('💬 Untuk pertanyaan lain, bot akan forward ke OpenClaw AI');
  })
  .catch(err => {
    console.error('❌ Failed to start bot:', err);
    if (err.message.includes('409')) {
      console.error('');
      console.error('⚠️ CONFLICT: Bot token sedang digunakan oleh proses lain!');
      console.error('   Matikan OpenClaw Telegram channel atau gunakan token berbeda.');
      console.error('');
      console.error('   Untuk mematikan Telegram di OpenClaw:');
      console.error('   1. Buka: C:\\Users\\PC\\.openclaw\\openclaw.json');
      console.error('   2. Ubah "enabled": true → "enabled": false di channels.telegram');
      console.error('   3. Restart OpenClaw: openclaw restart');
    }
  });

// Enable graceful stop
process.once('SIGINT', () => bot.stop('SIGINT'));
process.once('SIGTERM', () => bot.stop('SIGTERM'));
