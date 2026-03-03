// Test Telegram Bot - Simple version
const { Telegraf } = require('telegraf');

const BOT_TOKEN = '8724885465:AAFT0n7MMBgKfMUUYstNfPwblFqDEhWgAIA';
const ALLOWED_USER = '756112782';

const bot = new Telegraf(BOT_TOKEN);

// Middleware: hanya izinkan user tertentu
bot.use((ctx, next) => {
  if (ctx.from && ctx.from.id.toString() === ALLOWED_USER) {
    return next();
  }
  ctx.reply('❌ Maaf, Anda tidak memiliki izin.');
});

// Command handlers
bot.command(['start', 'help'], (ctx) => {
  ctx.reply('🛡️ AmmanGate Bot Online!\n\nCommands:\n/status\n/devices\n/alerts\n/block <mac>\n/unblock <mac>\n/blocked\n/suricata\n/clamav\n/scan [path]\n/filters');
});

bot.command('status', async (ctx) => {
  const http = require('http');
  const data = await new Promise((resolve, reject) => {
    http.get('http://127.0.0.1:8787/v1/system/status', (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => resolve(JSON.parse(body)));
    }).on('error', reject);
  });
  ctx.reply(`📊 STATUS\nCPU: ${data.cpu_load}%\nMemory: ${data.mem_used_mb}MB`);
});

bot.on('text', (ctx) => {
  ctx.reply(`Pesan diterima: "${ctx.message.text}"\n\nGunakan /help untuk bantuan.`);
});

console.log('🛡️ Starting simple bot...');

bot.launch()
  .then(() => {
    console.log('✅ Bot started!');
    console.log('👤 Allowed user:', ALLOWED_USER);
  })
  .catch(err => {
    console.error('❌ Error:', err.message);
    if (err.message.includes('409')) {
      console.error('');
      console.error('⚠️ CONFLICT: Bot token sedang digunakan!');
      console.error('   Matikan OpenClaw Telegram channel atau stop proses lain.');
    }
  });

process.once('SIGINT', () => bot.stop('SIGINT'));
process.once('SIGTERM', () => bot.stop('SIGTERM'));
