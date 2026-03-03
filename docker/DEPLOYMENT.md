# AmmanGate Docker Deployment Guide

## Complete Stack

The Docker setup now includes:
- **Backend**: Go API Server with ClamAV and Suricata (port 8787)
- **Frontend**: Next.js UI (port 3000)

## Quick Start

### 1. Backend Only (Minimal)

```bash
# Build and start backend only
docker compose up -d bodyguard-core

# Access backend
curl http://localhost:8787/v1/health
```

### 2. Full Stack (Backend + Frontend)

```bash
# Build and start everything
docker compose up -d

# Wait for services to be healthy
sleep 10

# Access:
# Backend API: http://localhost:8787/v1
# Frontend UI: http://localhost:3000
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                        Docker Network                        │
│                      (ammangate-net)                         │
│                                                               │
│  ┌────────────────────┐         ┌─────────────────────┐    │
│  │   Frontend         │         │   Backend           │    │
│  │   (Next.js)        │────────>│   (Go API)          │    │
│  │   Port: 3000       │         │   Port: 8787        │    │
│  │   Container        │         │   Container        │    │
│  │                    │         │                     │    │
│  │ - Environment:    │         │ - Environment:     │    │
│  │   - BACKEND_API   │         │   - BG_ADDR         │    │
│  │   - http://        │         │   - ClamAV          │    │
│  │     bodyguard-     │         │   - Suricata        │    │
│  │     core:8787/v1  │         │                     │    │
│  └────────────────────┘         └─────────────────────┘    │
│                                       │                     │        │
│  ┌────────────────────┐         ┌─────────────────────┐    │
│  │   Volumes          │         │   Volumes          │    │
│  │  - ammangate-data  │         │  - clamav-data     │    │
│  │                     │         │  - suricata-logs   │    │
│  └────────────────────┘         └─────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

## Environment Configuration

### Frontend Environment

Create `apps/bodyguard-ui/.env.local` for local development:

```bash
# For local development (localhost)
BACKEND_API=http://127.0.0.1:8787/v1

# For Docker development (container names)
# BACKEND_API=http://bodyguard-core:8787/v1

# For production (actual hostnames)
# BACKEND_API=http://your-server:8787/v1
```

### Backend Environment

Root `.env` file:

```bash
# Server Configuration
BG_ADDR=0.0.0.0:8787

# Credentials
BG_ADMIN_USER=admin
BG_ADMIN_PASS=change-this-in-production
BG_ACTION_PIN=change-this-in-production
BG_SESSION_SECRET=change-this-in-production

# ClamAV
CLAMAV_ADDRESS=/var/run/clamav/clamd.ctl

# Suricata
SURICATA_EVE_LOG=/var/log/suricata/eve.json

# Telegram
TELEGRAM_ALERTS_ENABLED=true
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id
```

## Docker Commands

### Build and Start

```bash
# Build all services
docker compose build

# Start all services
docker compose up -d

# Start specific service
docker compose up -d bodyguard-core
docker compose up -d bodyguard-ui
```

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f bodyguard-core
docker compose logs -f bodyguard-ui

# Last 100 lines
docker compose logs --tail=100 bodyguard-core
```

### Stop and Remove

```bash
# Stop all services
docker compose down

# Stop and remove volumes (WARNING: deletes data!)
docker compose down -v
```

### Management

```bash
# Restart services
docker compose restart

# Rebuild and restart
docker compose up -d --build

# Scale services (if needed)
docker compose up -d --scale bodyguard-core=2
```

### Shell Access

```bash
# Backend shell
docker compose exec bodyguard-core /bin/bash

# Frontend shell
docker compose exec bodyguard-ui sh
```

## Troubleshooting

### Frontend can't connect to backend

1. **Check if backend is running**:
   ```bash
   docker compose ps
   curl http://localhost:8787/v1/health
   ```

2. **Check frontend logs**:
   ```bash
   docker compose logs bodyguard-ui
   ```

3. **Verify network connectivity**:
   ```bash
   docker compose exec bodyguard-ui ping bodyguard-core
   ```

### Backend healthcheck failing

1. **Check backend logs**:
   ```bash
   docker compose logs bodyguard-core
   ```

2. **Check if ClamAV is running**:
   ```bash
   docker compose exec bodyguard-core clamdscan --version
   ```

### Volume issues

```bash
# List volumes
docker volume ls

# Inspect volume
docker volume inspect ammangate-data

# Remove and recreate (WARNING: data loss!)
docker compose down -v
docker compose up -d
```

### Update ClamAV Definitions

```bash
# Using management script
./docker-manage.sh update-av

# Or directly
docker compose exec bodyguard-core freshclam --datadir=/var/lib/clamav
```

## Production Deployment

### 1. Use Environment Files

```bash
# Copy example env files
cp .env.example .env
cp apps/bodyguard-ui/.env.docker.example apps/bodyguard-ui/.env.local

# Update with production values
nano .env
nano apps/bodyguard-ui/.env.local
```

### 2. Change Default Credentials

```bash
# Update .env
BG_ADMIN_PASS=strong-password-here
BG_ACTION_PIN=123456
BG_SESSION_SECRET=$(openssl rand -base64 32)
```

### 3. Configure Telegram

```bash
# Add your Telegram credentials
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_CHAT_ID=your_chat_id
```

### 4. Enable HTTPS (Recommended)

Use a reverse proxy like nginx or Traefik:

```yaml
# Example with nginx
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - bodyguard-ui
```

### 5. Resource Limits

Add to docker-compose.yml:

```yaml
services:
  bodyguard-core:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

## Backup and Restore

### Backup

```bash
# Backup all data
docker run --rm -v ammangate-data:/data -v $(pwd)/backup:/backup \
  alpine tar czf /backup/ammangate-$(date +%Y%m%d).tar.gz /data

# Backup ClamAV definitions
docker run --rm -v clamav-data:/data -v $(pwd)/backup:/backup \
  alpine tar czf /backup/clamav-$(date +%Y%m%d).tar.gz /data
```

### Restore

```bash
# Stop containers
docker compose down

# Restore data
docker run --rm -v ammangate-data:/data -v $(pwd)/backup:/backup \
  alpine tar xzf /backup/ammangate-20240303.tar.gz -C /

# Start containers
docker compose up -d
```

## Monitoring

### Health Checks

```bash
# Check service health
curl http://localhost:8787/v1/health    # Backend
curl http://localhost:3000/api/health    # Frontend

# Docker health status
docker compose ps
```

### Resource Usage

```bash
# Container stats
docker stats

# Disk usage
docker system df

# Volume usage
docker du -v $(docker volume ls -q)
```

## Update Strategy

### Update Application

```bash
# Pull latest code
git pull

# Rebuild and restart
docker compose up -d --build

# Or just restart (no rebuild)
docker compose restart
```

### Update ClamAV

```bash
# Update definitions
./docker-manage.sh update-av

# Or schedule automatic updates (cron job)
# 0 2 * * * cd /path/to/AmmanGate && ./docker-manage.sh update-av
```

## Support

For issues:
1. Check logs: `docker compose logs -f`
2. Check status: `docker compose ps`
3. Review this guide: `docker/README.md`
4. Quick start: `DOCKER-START.md`
