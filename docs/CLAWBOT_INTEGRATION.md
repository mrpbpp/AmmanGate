# AmmanGate + Clawbot Integration Guide

This guide explains how to integrate AmmanGate with Clawbot for AI-powered network security management via chat interface.

## Overview

**AmmanGate** provides the network security monitoring and threat detection, while **Clawbot** acts as the AI agent interface that allows you to interact with AmmanGate through natural language commands.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Clawbot AI     │────▶│  Claw Gateway   │────▶│  AmmanGate Core │
│  (Chat Interface)│     │  (Command Router)│     │  (Security API) │
│  Port 8080      │     │  Port 3001      │     │  Port 8787      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Prerequisites

- AmmanGate installed and running
- Node.js 20+ (for claw-gateway)
- Windows, Linux, or macOS

## Installation

### Option 1: Quick Install (Recommended)

Run the installation script for your platform:

**Windows (PowerShell):**
```powershell
cd apps\clawbot
.\install.ps1
```

**Linux/macOS (Bash):**
```bash
cd apps/clawbot
chmod +x install.sh
./install.sh
```

### Option 2: Manual Install

1. **Install Clawbot:**
```bash
curl -fsSL https://openclaw.ai/install.sh | bash
```

2. **Install claw-gateway dependencies:**
```bash
cd apps/claw-gateway
npm install
```

3. **Generate secure keys:**

**PowerShell:**
```powershell
$apiKey = [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])
$webhookSecret = [System.Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])
Write-Host "CLAWBOT_API_KEY=$apiKey"
Write-Host "CLAWBOT_WEBHOOK_SECRET=$webhookSecret"
```

**Bash:**
```bash
CLAWBOT_API_KEY=$(openssl rand -base64 32)
CLAWBOT_WEBHOOK_SECRET=$(openssl rand -base64 32)
echo "CLAWBOT_API_KEY=$CLAWBOT_API_KEY"
echo "CLAWBOT_WEBHOOK_SECRET=$CLAWBOT_WEBHOOK_SECRET"
```

4. **Configure environment:**

Copy `.env.example` to `.env` and update with your values:
```bash
cp .env.example .env
nano .env  # Edit with your values
```

## Starting the Services

### Development Mode

Start each service in separate terminals:

**Terminal 1 - AmmanGate Core:**
```bash
cd apps/bodyguard-core
go run main.go
```

**Terminal 2 - Clawbot:**
```bash
cd apps/clawbot
./start.sh  # Linux/macOS
# or
.\start.bat  # Windows
```

**Terminal 3 - Claw Gateway:**
```bash
cd apps/claw-gateway
npm start
```

**Terminal 4 - Dashboard (Optional):**
```bash
cd apps/bodyguard-ui
npm run dev
```

### Docker Deployment

```bash
# Start all services
docker-compose -f deploy/docker-compose.yml up -d

# View logs
docker-compose -f deploy/docker-compose.yml logs -f

# Stop services
docker-compose -f deploy/docker-compose.yml down
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CLAWBOT_API_KEY` | API key for Clawbot authentication | Required |
| `CLAWBOT_WEBHOOK_SECRET` | Secret for webhook signature verification | Required |
| `CLAWBOT_API_URL` | Clawbot API endpoint | `http://127.0.0.1:8080` |
| `CORE_API_URL` | AmmanGate Core API endpoint | `http://127.0.0.1:8787` |
| `ACTION_PIN` | PIN for approving security actions | `1234` |

### Clawbot Configuration

Edit `apps/clawbot/config/clawbot.yaml`:

```yaml
server:
  host: 0.0.0.0
  port: 8080

api:
  enabled: true
  api_key: ${CLAWBOT_API_KEY}

webhook:
  secret: ${CLAWBOT_WEBHOOK_SECRET}

handlers:
  - name: ammangate
    type: webhook
    url: http://localhost:3001/webhook/clawbot
    secret: ${CLAWBOT_WEBHOOK_SECRET}
    enabled: true
```

## Available Commands

Once everything is running, you can interact with AmmanGate through Clawbot:

### Read Commands (No PIN Required)

```
status          - Show system status
devices [n]     - List recent devices (default: 10)
alerts          - Show active alerts
why <alert_id>  - Explain alert (AI feature coming in v1.0)
help            - Show all commands
```

### Write Commands (PIN Required)

```
quarantine <device>     - Isolate device from network
unquarantine <device>   - Restore device access
block ip <ip>           - Block IP address
block domain <domain>   - Block domain
unblock ip <ip>         - Unblock IP
unblock domain <domain> - Unblock domain
PIN <code>              - Confirm pending action
```

## Usage Examples

### Check System Status
```
You: status

Clawbot: 🛡️ AmmanGate Status

⏱️ Uptime: 2h 15m
💾 Memory: 45 MB
🚨 Active Alerts: 0

_Last event: 2026-03-01T10:30:00Z_
```

### View Devices
```
You: devices 5

Clawbot: 📱 Devices (5)

🟢 *iPhone*
   192.168.1.105 • AA:BB:CC:DD:EE:FF
   Risk: 10/100

🟢 *MacBook Pro*
   192.168.1.100 • 11:22:33:44:55:66
   Risk: 5/100
```

### Block a Suspicious IP
```
You: block ip 192.168.1.200

Clawbot: ⚠️ Action Pending Approval

Action block_ip requested on 192.168.1.200. Reply with: PIN #### (expires 90s)

Reply with: `PIN 1234` to confirm.

[✅ Approve (1234)] [❌ Cancel]

You: PIN 1234

Clawbot: ✅ Action Executed

IP 192.168.1.200 blocked via firewall
Action ID: act_abc12345
```

## Testing

### Test the Gateway Directly

```bash
curl -X POST http://localhost:3001/test \
  -H "Content-Type: application/json" \
  -d '{"message":"status"}'
```

### Test Clawbot Connection

```bash
# Check Clawbot health
curl http://localhost:8080/health

# Check gateway health
curl http://localhost:3001/health

# Register webhook
curl -X POST http://localhost:3001/webhook/register \
  -H "Content-Type: application/json" \
  -d '{"webhook_url":"http://localhost:3001/webhook/clawbot"}'
```

## Troubleshooting

### Clawbot Not Starting

1. Check if port 8080 is available:
```bash
# Windows
netstat -an | findstr 8080

# Linux/macOS
lsof -i :8080
```

2. Verify configuration file exists:
```bash
ls -la apps/clawbot/config/clawbot.yaml
```

### Gateway Cannot Connect to Clawbot

1. Verify Clawbot is running:
```bash
curl http://localhost:8080/health
```

2. Check environment variables are set:
```bash
echo $CLAWBOT_API_KEY
echo $CLAWBOT_WEBHOOK_SECRET
```

3. Check gateway logs for connection errors

### Commands Not Working

1. Verify AmmanGate Core is running:
```bash
curl http://localhost:8787/v1/health
```

2. Check action PIN is correct:
```bash
echo $ACTION_PIN
```

3. Review logs in `apps/claw-gateway/` directory

## Security Best Practices

1. **Always change default credentials** before production use
2. **Use strong, randomly generated keys** for API keys and secrets
3. **Run Clawbot behind a reverse proxy** in production
4. **Enable webhook signature verification** by setting `CLAWBOT_WEBHOOK_SECRET`
5. **Use HTTPS** for all external communications
6. **Limit webhook source IPs** in Clawbot configuration

## Architecture Details

### Message Flow

```
User Message
    │
    ▼
┌─────────────┐
│  Clawbot    │ Receives message via chat interface
│  (Port 8080)│
└─────────────┘
    │
    │ Webhook POST
    ▼
┌─────────────┐
│ Claw Gateway│ Parses command, calls API
│ (Port 3001) │
└─────────────┘
    │
    │ HTTP Request
    ▼
┌─────────────┐
│ AmmanGate   │ Executes command
│ Core (8787) │
└─────────────┘
    │
    │ Response
    ▼
┌─────────────┐
│ Claw Gateway│ Formats response
└─────────────┘
    │
    │ Send Message API
    ▼
┌─────────────┐
│  Clawbot    │ Sends response to user
└─────────────┘
```

### Action Approval Flow

Actions that modify security state require PIN approval:

```
┌──────────┐
│  User    │ Sends "block ip 1.2.3.4"
└──────────┘
     │
     ▼
┌──────────────────┐
│  Claw Gateway    │ Creates approval request
└──────────────────┘
     │
     ▼
┌──────────────────┐
│  AmmanGate Core  │ Stores pending action
└──────────────────┘
     │
     ▼
┌──────────────────┐
│  User            │ Must reply with PIN within 90s
└──────────────────┘
     │
     ▼
┌──────────────────┐
│  Claw Gateway    │ Verifies PIN
└──────────────────┘
     │
     ▼
┌──────────────────┐
│  AmmanGate Core  │ Executes action
└──────────────────┘
```

## Advanced Configuration

### Custom Command Handlers

You can add custom commands by extending the `CommandParser` class in `apps/claw-gateway/index.js`:

```javascript
class CommandParser {
  constructor() {
    this.commands = {
      // ... existing commands
      "custom": this.handleCustom.bind(this),
    };
  }

  async handleCustom(args, chatId, userId) {
    // Your custom logic here
    return { text: "Custom command response" };
  }
}
```

### Multiple Chat Platforms

Clawbot supports multiple platforms. Configure additional handlers in `clawbot.yaml`:

```yaml
handlers:
  - name: ammangate_discord
    type: discord
    token: ${DISCORD_BOT_TOKEN}
    webhook_url: http://localhost:3001/webhook/clawbot
    enabled: true
```

### Rate Limiting

Configure rate limits in Clawbot to prevent abuse:

```yaml
rate_limit:
  messages_per_minute: 60
  commands_per_minute: 30
  burst_size: 10
```

## Support

For issues and questions:
- GitHub Issues: [AmmanGate Repository]
- Documentation: [Full Documentation Link]
- Clawbot Docs: [Clawbot Documentation]

---

**Made with ❤️ for secure homes and businesses**
