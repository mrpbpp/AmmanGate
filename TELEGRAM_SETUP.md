# Telegram Bot Setup for AmmanGate Alerts

This guide will help you set up a Telegram bot to receive security alerts from AmmanGate with AI-powered explanations.

## Step 1: Create a Telegram Bot

1. Open Telegram and search for **@BotFather**
2. Send the command `/newbot`
3. Follow the prompts to:
   - Choose a name for your bot (e.g., "AmmanGate Security Bot")
   - Choose a username (e.g., `ammangate_security_bot`)
4. **Copy the bot token** - it looks like: `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`

## Step 2: Get Your Chat ID

1. In Telegram, search for **@userinfobot**
2. Send any message (e.g., `/start`)
3. The bot will reply with your **Chat ID** - it looks like: `123456789`

## Step 3: Configure AmmanGate

Edit the `.env` file in `d:\AmmanGate\apps\bodyguard-core\.env`:

```env
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
TELEGRAM_CHAT_ID=123456789
TELEGRAM_ALERTS_ENABLED=true
```

## Step 4: Start the Bot Conversation

1. Find your bot on Telegram (search for the username you created)
2. Click **Start** or send `/start`

## Step 5: Test the Connection

After restarting AmmanGate Core, test the Telegram integration:

```bash
curl -X POST http://127.0.0.1:8787/v1/telegram/test
```

You should receive a test message like:

```
🤖 Clawbot Security Alert Test

✅ AmmanGate Security System is online and connected!

*System Status:*
- Suricata IDS: Active
- ClamAV Antivirus: Active
- Telegram Notifications: Enabled
```

## Step 6: Check Telegram Status

```bash
curl http://127.0.0.1:8787/v1/telegram/status
```

## What You'll Receive

When Suricata IDS detects a threat, you'll receive:

```
🔴 Clawbot Security Alert

🚨 DETECTION
`TEST - SSH Connection Attempt`
_Attempted Administrator Privilege Gain_

📡 CONNECTION
• From: `192.168.1.106:60307`
• To: `192.168.1.117:22`
• Protocol: `TCP`
• Time: `2026-03-02 01:06:45`

⚠️ SEVERITY
███ (1/3)

🤖 AI ANALYSIS
This alert indicates that someone attempted to connect to port 22 (SSH),
which is used for remote system administration...

[AI provides recommendations on what to do]

_Powered by AmmanGate Home Security_
```

## Troubleshooting

### Bot not sending messages

1. Check that you've started the bot (sent `/start`)
2. Verify the bot token and chat ID are correct
3. Check that `TELEGRAM_ALERTS_ENABLED=true`
4. Check AmmanGate Core logs: `tail -f bodyguard-core.log`

### No Chat ID from @userinfobot

Alternative method:
1. Send a message to your bot
2. Visit: `https://api.telegram.org/bot<TOKEN>/getUpdates`
3. Find your `chat id` in the response

### AI explanations not working

1. Verify LM Studio is running: `http://localhost:1234/v1/models`
2. Check `BG_LM_STUDIO_TOKEN` in `.env`
3. Ensure LM Studio has a model loaded

## Security Notes

- Keep your bot token secret - it gives full control of your bot
- Chat IDs are not sensitive, but consider who might see your messages
- Consider using a private channel instead of direct messages for multiple users

## Customization

You can customize alert messages by editing:
- `d:\AmmanGate\apps\bodyguard-core\telegram.go`

The severity levels:
- 🔴 Severity 1: High (Critical threats)
- 🟠 Severity 2: Medium (Suspicious activity)
- 🟡 Severity 3: Low (Informational)
