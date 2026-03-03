# Claw Gateway

AmmanGate WhatsApp/OpenClaw Gateway - handles ChatOps commands for the AI Home Cyber Bodyguard.

## Features

- **Read Commands**: `status`, `devices`, `alerts`, `why`
- **Write Commands**: `quarantine`, `block`, `unblock` (with PIN approval)
- **Secure Approval Flow**: All write actions require PIN confirmation
- **OpenClaw Integration**: Webhook endpoint for WhatsApp bot integration

## Commands

### Read-only (no approval needed)
- `status` - Show system status and active alerts count
- `devices [n]` - List N recent devices (default: 10)
- `alerts` - Show all active alerts
- `why <alert_id>` - Get explanation for an alert (v1.0 with LLM)
- `help` - Show all available commands

### Write (requires PIN approval)
- `quarantine <device>` - Isolate device from internet
- `unquarantine <device>` - Restore device access
- `block ip <ip>` - Block IP address
- `block domain <domain>` - Block domain
- `unblock ip <ip>` - Unblock IP address
- `unblock domain <domain>` - Unblock domain
- `PIN <code>` - Confirm pending action

## Setup

1. Install dependencies:
   ```bash
   npm install
   ```

2. Copy environment file:
   ```bash
   cp .env.example .env
   ```

3. Configure settings:
   - Set `ACTION_PIN` to match `BG_ACTION_PIN` in bodyguard-core
   - Set `CORE_API_URL` if different from default

4. Start the gateway:
   ```bash
   npm start
   ```

## Testing

```bash
curl -X POST http://localhost:3001/test \
  -H "Content-Type: application/json" \
  -d '{"message":"status"}'
```

## OpenClaw Integration

The gateway exposes a webhook endpoint at `/webhook/message` that can be integrated with OpenClaw:

```javascript
// OpenClaw bot configuration
{
  webhookUrl: "http://localhost:3001/webhook/message",
  handleMessage: async (message) => {
    // Forward to claw-gateway
    await fetch(webhookUrl, {
      method: "POST",
      body: JSON.stringify({
        from: message.from,
        message: message.text,
        chat_id: message.chatId,
      }),
    });
  },
}
```

## Security Notes

- Always use HTTPS in production
- Change default PIN immediately
- Use allow-listed phone numbers for WhatsApp access
- Monitor audit logs in bodyguard-core
