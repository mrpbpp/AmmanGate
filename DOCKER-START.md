# AmmanGate Docker Quick Start

## Langkah Cepat (Quick Start) untuk Windows/Linux/Mac

### 1. Persiapan (Prerequisites)

Pastikan sudah terinstall:
- **Docker Desktop** - Download dari https://www.docker.com/products/docker-desktop
- **Git** (opsional) - Untuk clone repository

### 2. Build dan Jalankan (Build and Run)

#### Menggunakan Script (Linux/Mac/Windows Git Bash):

```bash
# 1. Masuk ke directory project
cd AmmanGate

# 2. Build image Docker
./docker-manage.sh build

# 3. Jalankan container
./docker-manage.sh start
```

#### Menggunakan Batch Script (Windows CMD):

```cmd
REM 1. Masuk ke directory project
cd AmmanGate

REM 2. Build image Docker
docker-manage.bat build

REM 3. Jalankan container
docker-manage.bat start
```

#### Menggunakan Docker Compose Langsung:

```bash
# Build dan jalankan
docker compose up -d --build
```

### 3. Verifikasi (Verify)

```bash
# Cek status container
./docker-manage.sh status

# Lihat logs
./docker-manage.sh logs

# Test API
curl http://localhost:8787/v1/health
```

### 4. Akses AmmanGate

Buka browser dan akses:
- **Frontend UI**: http://localhost:3000
- **API Documentation**: http://localhost:8787/v1
- **Default Login**:
  - Username: `admin`
  - Password: `admin123`

### 5. Komando Management (Management Commands)

```bash
./docker-manage.sh build      # Build images
./docker-manage.sh start      # Start containers
./docker-manage.sh stop       # Stop containers
./docker-manage.sh restart    # Restart containers
./docker-manage.sh logs       # View logs
./docker-manage.sh status     # Show status
./docker-manage.sh shell      # Open shell in container
./docker-manage.sh update-av  # Update ClamAV definitions
./docker-manage.sh clean      # Remove everything
```

## Komponen yang Terinstall (Installed Components)

✅ **Ubuntu 22.04 LTS Minimal**
- Base OS yang ringan dan aman

✅ **Go 1.21.5**
- Runtime untuk AmmanGate backend

✅ **Node.js 20 + Next.js**
- Frontend UI untuk dashboard

✅ **ClamAV Antivirus**
- Real-time virus scanning
- Auto-update virus definitions

✅ **Suricata IDS**
- Intrusion Detection System
- EVE JSON logging

✅ **AmmanGate Bodyguard Core (Backend)**
- API Server on port 8787
- WebSocket support
- Telegram integration

✅ **AmmanGate Frontend (UI)**
- Next.js web interface on port 3000
- User management dashboard
- Real-time updates

## Troubleshooting

### Container tidak mau start
```bash
# Cek logs
./docker-manage.sh logs

# Cek status
docker ps -a | grep ammangate
```

### Port 8787 sudah digunakan
```bash
# Kill process yang menggunakan port 8787
# Windows:
netstat -ano | findstr :8787
taskkill /PID [PID] /F

# Linux/Mac:
lsof -ti:8787 | xargs kill -9
```

### Update ClamAV virus definitions
```bash
./docker-manage.sh update-av
```

### Shell ke dalam container
```bash
./docker-manage.sh shell

# Di dalam container:
clamdscan --version           # Test ClamAV
suricata -V                    # Test Suricata
tail -f /var/log/suricata/eve.json  # Lihat Suricata logs
```

### Reset everything (Hapus semua data)
```bash
./docker-manage.sh clean
./docker-manage.sh build
./docker-manage.sh start
```

## Konfigurasi Lanjutan

Edit file `.env` untuk mengubah konfigurasi:

```bash
# Copy file env example
cp .env.example .env

# Edit dengan text editor
notepad .env        # Windows
nano .env           # Linux/Mac
```

Konfigurasi penting:
- `BG_ADMIN_PASS` - Password admin (ubah di production!)
- `BG_ACTION_PIN` - PIN untuk approve security action
- `TELEGRAM_BOT_TOKEN` - Token bot Telegram
- `TELEGRAM_CHAT_ID` - Chat ID untuk notifikasi

## Support

Untuk bantuan dan pertanyaan:
- Cek logs: `./docker-manage.sh logs`
- Baca dokumentasi: `docker/README.md`
- Issue tracker: GitHub Issues
