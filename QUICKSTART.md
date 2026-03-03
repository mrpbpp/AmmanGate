# AmmanGate Quick Start Guide

## Prerequisites

- Go 1.22+
- Node.js 20+
- Git

## Option 1: Development Mode (Recommended for First Time)

### 1. Start Go Backend

```bash
cd apps/bodyguard-core
go mod download
go run main.go
```

The API will start on `http://127.0.0.1:8787`

### 2. Start Next.js UI (New Terminal)

```bash
cd apps/bodyguard-ui
npm install
npm run dev
```

The dashboard will be available at `http://localhost:3000`

### 3. Start Claw Gateway (New Terminal)

```bash
cd apps/claw-gateway
npm install
npm start
```

The gateway will start on `http://localhost:3001`

### 4. Access Dashboard

Open browser: `http://localhost:3000`

**Default credentials:**
- Username: `admin`
- Password: `admin123`
- Action PIN: `1234`

## Option 2: Docker Deployment

```bash
# Copy and configure environment
cp deploy/.env.example .env
nano .env  # Change ACTION_PIN!

# Start all services
docker-compose -f deploy/docker-compose.yml up -d

# View logs
docker-compose -f deploy/docker-compose.yml logs -f
```

## Testing

### Test API
```bash
curl http://localhost:8787/v1/health
```

### Test Claw Gateway
```bash
curl -X POST http://localhost:3001/test \
  -H "Content-Type: application/json" \
  -d '{"message":"status"}'
```

## WhatsApp Commands (via OpenClaw)

Once OpenClaw is integrated, send these commands:

```
status          - System status
devices         - List devices
alerts          - Active alerts
quarantine <ip> - Isolate device
PIN 1234        - Approve action
help            - All commands
```

## Troubleshooting

### Port already in use
Change port in environment variable:
```bash
BG_ADDR=127.0.0.1:8788 go run main.go
```

### Database errors
Delete and recreate:
```bash
rm -f data/bodyguard.db*
mkdir -p data
```

### Next.js build errors
```bash
rm -rf .next node_modules
npm install
npm run dev
```

## Next Steps

1. Change default credentials
2. Configure network sensors
3. Set up OpenClaw integration
4. Review security settings

See [README.md](README.md) for full documentation.
