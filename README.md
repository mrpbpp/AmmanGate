# AmmanGate - AI Home Cyber Bodyguard

> Sistem keamanan rumah berbasis AI dengan proteksi real-time, antivirus, IDS, dan kontrol orang tua.

> AI-powered home security system with real-time protection, antivirus, IDS, and parental control.

[\![Go Version](https://img.shields.io/badge/Go-1.24.0-blue)](https://golang.org)
[\![Node.js](https://img.shields.io/badge/Node.js-20.0-green)](https://nodejs.org)
[\![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## Table of Contents / Daftar Isi

- [About / Tentang](#about--tentang)
- [Architecture / Arsitektur](#architecture--arsitektur)
- [Features / Fitur](#features--fitur)
- [Requirements / Kebutuhan](#requirements--kebutuhan)
- [Installation / Instalasi](#installation--instalasi)
- [Configuration / Konfigurasi](#configuration--konfigurasi)
- [Usage / Penggunaan](#usage--penggunaan)
- [Docker Deployment](#docker-deployment)
- [Contributing](#contributing)
- [License](#license)

---

## About / Tentang

**AmmanGate** adalah sistem keamanan jaringan rumah yang komprehensif dengan fitur AI-powered security analysis. Sistem ini memantau semua perangkat di jaringan Anda, mendeteksi ancaman keamanan, dan memberikan notifikasi real-time melalui Telegram.

**AmmanGate** is a comprehensive home network security system with AI-powered security analysis. It monitors all devices on your network, detects security threats, and provides real-time notifications via Telegram.

### Key Capabilities / Kemampuan Utama

- Real-time network device discovery and monitoring
- ClamAV antivirus integration for file scanning
- Suricata IDS for intrusion detection
- Parental control with DNS filtering
- Device blocking by MAC address
- AI-powered security analysis (via LM Studio)
- Telegram bot integration for alerts
- Web dashboard for management

---

## Architecture / Arsitektur

The AmmanGate system consists of several integrated components:

### Components / Komponen

| Component / Komponen | Technology / Teknologi | Description / Deskripsi |
|---------------------|------------------------|------------------------|
| **Backend API** | Go 1.24 | REST API & WebSocket server |
| **Frontend UI** | Next.js 14 | Web dashboard for management |
| **Antivirus** | ClamAV | Real-time virus scanning |
| **IDS** | Suricata | Network intrusion detection |
| **DNS Server** | Go DNS | Content filtering & parental control |
| **AI Agent** | OpenClaw | Natural language interface |
| **Database** | SQLite/PostgreSQL | Data persistence |

---

## Features / Fitur

### Network Security / Keamanan Jaringan

- **Device Discovery** - Automatic detection of all network devices
- **Risk Scoring** - AI-powered risk assessment for each device
- **Device Fingerprinting** - OS and service detection
- **Network Monitoring** - Real-time traffic analysis

### Antivirus Protection / Proteksi Antivirus

- **ClamAV Integration** - Industry-standard antivirus engine
- **Real-time Scanning** - On-demand file scanning
- **Virus Definition Updates** - Automatic signature updates
- **Scan Results** - Detailed threat reports

### Intrusion Detection / Deteksi Intrusi

- **Suricata IDS** - Enterprise-grade intrusion detection
- **EVE JSON Logging** - Structured event logs
- **Alert Deduplication** - Reduces notification spam
- **GeoIP Lookup** - Geographic source tracking

### Parental Control / Kontrol Orang Tua

- **DNS Filtering** - Block malicious/inappropriate domains
- **Device Profiles** - Per-device filtering levels (off, light, moderate, strict)
- **MAC Address Blocking** - Block/unblock specific devices
- **DNS Query Logging** - Monitor all DNS requests

### AI-Powered Analysis / Analisis Berbasis AI

- **Security Analysis** - Natural language security queries
- **Anomaly Detection** - AI-powered pattern recognition
- **Smart Recommendations** - Actionable security insights
- **Explainable Alerts** - Human-readable threat descriptions

### Communication / Komunikasi

- **Telegram Bot** - Real-time alerts and notifications
- **AI Chat Interface** - Natural language interaction via OpenClaw
- **WebSocket Updates** - Live dashboard updates
- **Multi-user Support** - Multiple user accounts with roles

---

## Requirements / Kebutuhan

### System Requirements / Persyaratan Sistem

- **OS**: Windows 10/11, Linux, or macOS
- **RAM**: 4GB minimum, 8GB recommended
- **Disk**: 500MB for application, 2GB+ for logs and database
- **Network**: Administrative access for network monitoring

### Software Dependencies / Ketergantungan Perangkat Lunak

#### For Backend / Untuk Backend
- **Go** 1.24.0 or later
- **Git** (for cloning)
- **ClamAV** (optional, for antivirus)
- **Suricata** (optional, for IDS)

#### For Frontend / Untuk Frontend
- **Node.js** 20.x or later
- **npm** or **yarn** or **pnpm**

#### Optional Components / Komponen Opsional
- **LM Studio** - For local AI analysis
- **Docker** - For containerized deployment
- **PostgreSQL** - For production database (alternative to SQLite)

---

## Installation / Instalasi

### Quick Install (Windows) / Instalasi Cepat

1. **Clone the repository / Clone repositori**

   ```bash
   git clone https://github.com/yourusername/AmmanGate.git
   cd AmmanGate
   ```

2. **Configure environment / Konfigurasi environment**

   ```bash
   copy .env.example .env
   notepad .env
   ```

3. **Start services / Jalankan layanan**

   ```bash
   start-services.bat
   ```

4. **Access the dashboard / Akses dashboard**

   - Frontend UI: http://localhost:3000
   - Backend API: http://localhost:8787/v1
   - Default login: `admin` / `admin123`

### Docker Installation / Instalasi Docker

```bash
# Build and start all services
docker compose up -d --build

# Check status
docker compose ps

# View logs
docker compose logs -f
```

See [DOCKER-START.md](DOCKER-START.md) for detailed Docker instructions.

### Manual Installation / Instalasi Manual

#### Backend / Backend

```bash
cd apps/bodyguard-core
go mod download
go run main.go
```

#### Frontend / Frontend

```bash
cd apps/bodyguard-ui
npm install
npm run dev
```

---

## Configuration / Konfigurasi

### Environment Variables / Variabel Lingkungan

Create a `.env` file in the root directory:

```bash
# Admin credentials
BG_ADMIN_USER=admin
BG_ADMIN_PASS=your_secure_password

# Action PIN
BG_ACTION_PIN=123456

# API Server
BG_ADDR=127.0.0.1:8787

# Database
DB_TYPE=sqlite
BG_DB=./data/bodyguard.db

# Telegram Bot
TELEGRAM_BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_CHAT_ID=your_telegram_chat_id
TELEGRAM_ALERTS_ENABLED=true

# ClamAV
CLAMAV_ADDRESS=127.0.0.1:3310

# Suricata
SURICATA_EVE_LOG=./suricata-logs/eve.json

# LM Studio (optional)
BG_LM_STUDIO_URL=http://localhost:1234/v1
BG_LM_STUDIO_MODEL=your_model_name
```

### Telegram Bot Setup / Setup Bot Telegram

1. Create a bot via [@BotFather](https://t.me/BotFather) on Telegram
2. Copy the bot token
3. Get your chat ID from [@userinfobot](https://t.me/userinfobot)
4. Add credentials to `.env` file

### ClamAV Setup / Setup ClamAV

#### Windows
```bash
# Run setup script
setup-clamav.bat
```

#### Linux
```bash
sudo apt-get install clamav clamav-daemon
sudo freshclam
sudo systemctl start clamav-daemon
```

### Suricata Setup / Setup Suricata

#### Windows
```bash
# Run permission fix
fix-suricata-permissions.bat

# Start in user mode
start-suricata-user.bat
```

#### Linux
```bash
sudo apt-get install suricata
sudo suricata -T -c /etc/suricata/suricata.yaml
```

---

## Usage / Penggunaan

### Web Dashboard / Dashboard Web

1. **Login** - Access http://localhost:3000
2. **Dashboard** - View system status and alerts
3. **Devices** - Monitor all network devices
4. **Events** - View security events
5. **Settings** - Configure system settings

### Telegram Bot / Bot Telegram

Send commands to your Telegram bot:

- `/status` - System status
- `/devices` - List all devices
- `/alerts` - Active alerts
- `/scan` - Quick security scan

Or use natural language:
- "How is my network doing?"
- "Show me high-risk devices"
- "What happened today?"

### API Endpoints / Endpoint API

#### Authentication
```bash
POST /v1/auth/login     # Login
POST /v1/auth/logout    # Logout
```

#### System
```bash
GET  /v1/health           # Health check
GET  /v1/system/status    # System status
GET  /v1/system/network   # Network info
```

#### Devices
```bash
GET  /v1/devices              # List devices
GET  /v1/devices/{id}         # Device details
POST /v1/devices/{id}/fingerprint  # Fingerprint device
```

#### Security
```bash
GET  /v1/alerts/active       # Active alerts
GET  /v1/events              # Security events
POST /v1/ai/analyze          # AI analysis
```

#### Antivirus
```bash
GET  /v1/clamav/status       # ClamAV status
POST /v1/clamav/scan         # Scan data
```

#### Intrusion Detection
```bash
GET  /v1/suricata/status     # Suricata status
GET  /v1/suricata/alerts     # IDS alerts
```

---

## Docker Deployment

AmmanGate provides complete Docker containerization with all dependencies included.

### Quick Start

```bash
# Build and start
docker compose up -d --build

# Access services
# Frontend: http://localhost:3000
# Backend:  http://localhost:8787/v1
```

### Management Scripts

**Linux/Mac:**
```bash
./docker-manage.sh build    # Build images
./docker-manage.sh start    # Start containers
./docker-manage.sh stop     # Stop containers
./docker-manage.sh logs     # View logs
./docker-manage.sh shell    # Shell access
```

**Windows:**
```cmd
docker-manage.bat build
docker-manage.bat start
docker-manage.bat stop
docker-manage.bat logs
```

For detailed Docker documentation, see:
- [DOCKER-START.md](DOCKER-START.md) - Quick start guide
- [docker/README.md](docker/README.md) - Complete Docker documentation
- [docker/DEPLOYMENT.md](docker/DEPLOYMENT.md) - Production deployment guide

---

## Project Structure / Struktur Proyek

```
AmmanGate/
|-- apps/
|   |-- bodyguard-core/       # Backend API (Go)
|   |   |-- main.go           # Application entry point
|   |   |-- api.go            # API handlers
|   |   |-- models.go         # Data models
|   |   |-- telegram.go       # Telegram integration
|   |   |-- suricata.go       # Suricata IDS manager
|   |   `-- migrations/       # Database migrations
|   |
|   |-- bodyguard-ui/         # Frontend UI (Next.js)
|   |   |-- app/              # Next.js 14 app directory
|   |   |   |-- dashboard/    # Dashboard pages
|   |   |   |-- devices/      # Device management
|   |   |   |-- settings/     # Settings pages
|   |   |   `-- api/          # API routes
|   |   |-- components/       # React components
|   |   `-- lib/              # Utility functions
|   |
|   `-- claw-gateway/         # OpenClaw AI Agent
|
|-- docker/                   # Docker configuration
|   |-- Dockerfile.bodyguard-core
|   |-- Dockerfile.bodyguard-ui
|   |-- clamd.conf           # ClamAV config
|   `-- suricata.yaml        # Suricata config
|
|-- migrations/               # Database migrations
|-- scripts/                  # Utility scripts
|-- suricata-logs/           # Suricata log files
|-- .env.example             # Environment template
|-- docker-compose.yml       # Docker orchestration
|-- README.md                # This file
`-- DOCKER-START.md          # Docker quick start
```

---

## Troubleshooting / Pemecahan Masalah

### Port Already in Use / Port Sudah Digunakan

**Windows:**
```cmd
netstat -ano | findstr :8787
taskkill /PID [PID] /F
```

**Linux/Mac:**
```bash
lsof -ti:8787 | xargs kill -9
```

### ClamAV Not Responding / ClamAV Tidak Merespons

```bash
# Check ClamAV status
./docker-manage.sh shell
clamdscan --version

# Update virus definitions
freshclam
```

### Telegram Bot Not Working / Bot Telegram Tidak Berfungsi

1. Verify bot token is correct
2. Check chat ID matches your Telegram user ID
3. Ensure `TELEGRAM_ALERTS_ENABLED=true`
4. Test connection: `curl http://localhost:8787/v1/telegram/test`

### Database Errors / Error Database

```bash
# Reset database (WARNING: Deletes all data)
rm ./data/bodyguard.db
./start-services.bat
```

---

## Security Considerations / Pertimbangan Keamanan

⚠️ **IMPORTANT / PENTING**: Change default credentials in production\!

1. Change `BG_ADMIN_PASS` in `.env`
2. Change `BG_ACTION_PIN` to a secure PIN
3. Generate strong `BG_SESSION_SECRET` with `openssl rand -base64 32`
4. Use HTTPS in production (via reverse proxy)
5. Keep ClamAV definitions updated
6. Review Suricata alerts regularly
7. Restrict `TELEGRAM_ALLOWED_USER_IDS`

---

## Contributing

Contributions are welcome\! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## License / Lisensi

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Support & Community / Dukungan & Komunitas

- **Documentation**: [DOCKER-START.md](DOCKER-START.md), [docker/README.md](docker/README.md)
- **Issue Tracker**: GitHub Issues
- **Updates**: Check the repository for latest releases

---

## Acknowledgments / Penghargaan

- **ClamAV** - Open source antivirus engine
- **Suricata** - Intrusion Detection System
- **OpenClaw** - AI Agent Platform
- **Next.js** - React framework
- **Go** - Backend programming language

---

**Made with ❤️ for home network security**

**Dibuat dengan ❤️ untuk keamanan jaringan rumah**
