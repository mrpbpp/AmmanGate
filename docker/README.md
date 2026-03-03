# AmmanGate Docker Setup

Complete Docker setup for AmmanGate with all dependencies including ClamAV Antivirus and Suricata IDS.

## Prerequisites

- Docker 20.10 or higher
- Docker Compose 2.0 or higher
- At least 2GB RAM available
- 10GB free disk space

## Quick Start

### 1. Clone and Navigate

```bash
cd /path/to/AmmanGate
```

### 2. Build and Start

```bash
# Build Docker images
./docker-manage.sh build

# Start containers
./docker-manage.sh start
```

### 3. Verify Installation

```bash
# Check container status
./docker-manage.sh status

# View logs
./docker-manage.sh logs
```

### 4. Access AmmanGate

- **API Server**: http://localhost:8787/v1
- **Health Check**: http://localhost:8787/v1/health

## Management Commands

```bash
# Build images
./docker-manage.sh build

# Start containers
./docker-manage.sh start

# Stop containers
./docker-manage.sh stop

# Restart containers
./docker-manage.sh restart

# View logs
./docker-manage.sh logs

# Open shell in container
./docker-manage.sh shell

# Update ClamAV definitions
./docker-manage.sh update-av

# Remove everything (containers + volumes)
./docker-manage.sh clean
```

## Environment Variables

Edit `docker-compose.yml` to configure:

```yaml
environment:
  # API Configuration
  - BG_ADDR=0.0.0.0:8787              # API bind address
  - BG_ADMIN_USER=admin              # Admin username
  - BG_ADMIN_PASS=admin123           # Admin password (CHANGE IN PRODUCTION!)
  - BG_ACTION_PIN=1234               # Action PIN (CHANGE IN PRODUCTION!)
  - BG_SESSION_SECRET=...           # Session secret (CHANGE IN PRODUCTION!)

  # Telegram Integration
  - TELEGRAM_ALERTS_ENABLED=true     # Enable Telegram alerts
  - TELEGRAM_BOT_TOKEN=your_token    # Your bot token
  - TELEGRAM_CHAT_ID=your_chat_id    # Your chat ID

  # ClamAV Configuration
  - CLAMAV_ADDRESS=/var/run/clamav/clamd.ctl

  # Suricata Configuration
  - SURICATA_EVE_LOG=/var/log/suricata/eve.json
```

## Volumes

| Volume | Description |
|--------|-------------|
| `ammangate-data` | Application data and database |
| `clamav-data` | ClamAV virus definitions |
| `suricata-logs` | Suricata IDS logs |
| `clamav-logs` | ClamAV scan logs |

## Included Components

### 1. Ubuntu 22.04 LTS (Minimal)
- Minimal base image with only essential packages
- Optimized for security and size

### 2. Go 1.21.5
- Multi-stage build for minimal final image
- Full CGO support for SQLite

### 3. ClamAV Antivirus
- Real-time virus scanning
- Automatic definition updates
- Integration with AmmanGate API

### 4. Suricata IDS
- Intrusion Detection System
- EVE JSON logging
- Compatible with AmmanGate alerts

## Container Capabilities

The container runs with elevated privileges for network monitoring:

- `NET_ADMIN` - Network administration (required for Suricata)
- `NET_RAW` - Raw socket access
- `SYS_ADMIN` - System administration
- `--privileged` - Full privileges for IDS functionality
- `network_mode: host` - Direct host network access

## Troubleshooting

### View Logs

```bash
# All logs
./docker-manage.sh logs

# Specific service
docker logs ammangate-core

# Follow logs
docker logs -f ammangate-core
```

### Open Shell

```bash
./docker-manage.sh shell
```

Inside the container, you can:
- Test ClamAV: `clamdscan --version`
- Check Suricata: `suricata -V`
- View logs: `tail -f /var/log/suricata/eve.json`

### Update ClamAV Definitions

```bash
./docker-manage.sh update-av
```

### Check ClamAV Status

```bash
docker exec ammangate-core clamdscan --version
docker exec ammangate-core freshclam --datadir=/var/lib/clamav
```

### Restart Services

```bash
# Restart everything
./docker-manage.sh restart

# Restart individual service
docker restart ammangate-core
```

## Building Without Docker Compose

If you prefer to use plain Docker:

```bash
# Build image
docker build -f docker/Dockerfile.bodyguard-core -t ammangate-core:latest .

# Run container
docker run -d \
  --name ammangate-core \
  --hostname ammangate \
  --restart unless-stopped \
  --network host \
  --privileged \
  -v $(pwd)/data:/ammangate/data \
  -v ammangate-clamav:/var/lib/clamav \
  -v ammangate-suricata:/var/log/suricata \
  -p 8787:8787 \
  ammangate-core:latest
```

## Security Considerations

1. **Change Default Credentials**
   - Update `BG_ADMIN_PASS` in docker-compose.yml
   - Update `BG_ACTION_PIN`
   - Set a strong `BG_SESSION_SECRET`

2. **Telegram Security**
   - Use environment variables for bot token
   - Don't commit tokens to version control

3. **Network Exposure**
   - Container runs with `network_mode: host` for IDS functionality
   - Consider using a reverse proxy for production
   - Enable TLS/HTTPS in production

## Production Deployment

For production deployment:

1. **Use environment file**:
   ```bash
   cp .env.example .env
   # Edit .env with your values
   ```

2. **Enable HTTPS**:
   ```yaml
   environment:
     - BG_SECURE=true
     - BG_TLS_CERT=/path/to/cert.pem
     - BG_TLS_KEY=/path/to/key.pem
   ```

3. **Configure backups**:
   ```bash
   # Backup volumes
   docker run --rm -v ammangate-data:/data -v $(pwd)/backup:/backup ubuntu \
     tar czf /backup/ammangate-$(date +%Y%m%d).tar.gz /data
   ```

## Development

### Rebuild During Development

```bash
# Stop container
./docker-manage.sh stop

# Rebuild image
./docker-manage.sh build

# Start container
./docker-manage.sh start
```

### Access Development Tools

```bash
# Open shell
./docker-manage.sh shell

# Install additional tools
apt-get update && apt-get install -y [package]

# Run Go commands
go version
go mod [command]
```

## Support

For issues or questions:
- Check logs: `./docker-manage.sh logs`
- Check container status: `./docker-manage.sh status`
- Review configuration files in `docker/` directory

## License

MIT License - See LICENSE file for details
