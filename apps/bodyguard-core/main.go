package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("[DEBUG] .env file not found or error loading: %v", err)
	} else {
		log.Printf("[DEBUG] .env file loaded successfully")
	}
	// Debug: print CLAMAV_ADDRESS value
	log.Printf("[DEBUG] CLAMAV_ADDRESS=%s", os.Getenv("CLAMAV_ADDRESS"))
}

// App represents the application
type App struct {
	db            *sql.DB
	hub           *WSHub
	actionEng     *ActionEngine
	deviceDB      *DeviceDB
	eventDB       *EventDB
	startedAt     time.Time
	auth          *AuthManager
	userManager   *UserManager
	sensorManager *SensorManager
	aiEngine      *AIEngine
	honeypot      *HoneypotManager
	fingerprinter *Fingerprinter
	filterEngine  *FilterEngine
	dnsServer     *DNSServer
	clamAV        *ClamAVClient
	suricata      *SuricataManager
	telegram      *TelegramService
}

func main() {
	// Configuration from environment variables with defaults
	addr := env("BG_ADDR", "127.0.0.1:8787")
	migrationsDir := env("BG_MIGRATIONS", "./migrations")

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting AmmanGate Bodyguard Core...")

	// Get database configuration from environment
	dbConfig := GetDBConfig()
	log.Printf("Database type: %s", dbConfig.Type)

	// Initialize database
	db, err := OpenDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Apply migrations
	if err := applyMigrations(db, migrationsDir); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
	log.Println("Database initialized")

	// Initialize components
	authMgr := NewAuthManager()
	if err := authMgr.Initialize(); err != nil {
		log.Fatalf("Failed to initialize auth: %v", err)
	}

	// Initialize user manager with default admin
	userMgr := NewUserManager(authMgr.sessionSecret)
	defaultUsername, defaultPassword, _ := authMgr.GetDefaultCredentials()
	if err := userMgr.Initialize(defaultUsername, defaultPassword); err != nil {
		log.Fatalf("Failed to initialize user manager: %v", err)
	}

	hub := NewHub()
	actionEng := NewActionEngine(db, hub, authMgr)
	deviceDB := NewDeviceDB(db)
	eventDB := NewEventDB(db)
	sensorMgr := NewSensorManager(db, hub, deviceDB, eventDB)
	filterEngine := NewFilterEngine(db)
	aiEngine := NewAIEngine(db, hub, deviceDB, eventDB, filterEngine)
	geoLookup := NewGeoLookup()
	fingerprinter := NewFingerprinter()
	honeypotMgr := NewHoneypotManager(db, deviceDB, eventDB, hub, aiEngine, geoLookup)
	dnsServer := NewDNSServer(db, filterEngine)
	clamAV := NewClamAVClient()
	suricataMgr := NewSuricataManager(db, eventDB, hub)

	// Initialize LM Studio client for AI features
	lmClient := NewLMStudioClient()

	// Initialize Telegram service with LM Studio client
	telegramSvc := NewTelegramService(lmClient)

	// Set WebSocket hub for Telegram service
	telegramSvc.SetHub(hub)

	// Set Telegram service for Suricata alerts
	suricataMgr.SetTelegramService(telegramSvc)

	// Set global geo lookup for Suricata
	SetGeoLookup(geoLookup)

	app := &App{
		db:            db,
		hub:           hub,
		actionEng:     actionEng,
		deviceDB:      deviceDB,
		eventDB:       eventDB,
		startedAt:     time.Now(),
		auth:          authMgr,
		userManager:   userMgr,
		sensorManager: sensorMgr,
		aiEngine:      aiEngine,
		honeypot:      honeypotMgr,
		fingerprinter: fingerprinter,
		filterEngine:  filterEngine,
		dnsServer:     dnsServer,
		clamAV:        clamAV,
		suricata:      suricataMgr,
		telegram:      telegramSvc,
	}

	// Start background tasks
	go app.startSensors()

	// Start sensor manager
	if err := sensorMgr.Start(); err != nil {
		log.Printf("Warning: Failed to start sensor manager: %v", err)
	}

	// Start honeypot
	if err := honeypotMgr.Start(); err != nil {
		log.Printf("Warning: Failed to start honeypot: %v", err)
	}

	// Start DNS server if enabled
	if env("BG_DNS_ENABLED", "false") == "true" {
		if err := dnsServer.Start(); err != nil {
			log.Printf("Warning: Failed to start DNS server: %v", err)
		} else {
			defer dnsServer.Stop()
		}
	}

	// Start Suricata manager if enabled
	if err := suricataMgr.Start(); err != nil {
		log.Printf("Warning: Failed to start Suricata manager: %v", err)
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.SetHeader("X-Content-Type-Options", "nosniff"))

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			// Allow requests from frontend
			if origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" ||
			   origin == "http://localhost:3001" || origin == "http://127.0.0.1:3001" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// API Routes
	r.Route("/v1", func(r chi.Router) {
		// Authentication routes
		r.Post("/auth/login", app.handleLogin)
		r.Post("/auth/logout", app.handleLogout)

		r.Get("/health", app.handleHealth)
		r.Get("/system/status", app.handleSystemStatus)
		r.Get("/system/network", app.handleNetworkInfo)

		r.Get("/devices", app.handleDevicesList)
		r.Get("/devices/{id}", app.handleDeviceDetail)
		r.Post("/devices/{id}/fingerprint", app.handleDeviceFingerprint)
		r.Post("/devices/seed", app.handleSeedDevices) // For testing

		r.Post("/ai/analyze", app.handleAIAnalyze) // General security analysis
		r.Post("/explain", app.handleExplain) // Legacy endpoint

		r.Get("/events", app.handleEvents)
		r.Get("/alerts/active", app.handleActiveAlerts)

		r.Get("/honeypot/status", app.handleHoneypotStatus)
		r.Get("/honeypot/hits", app.handleHoneypotHits)

		// ClamAV routes
		r.Get("/clamav/status", app.handleClamAVStatus)
		r.Post("/clamav/refresh", app.handleClamAVRefresh)
		r.Post("/clamav/scan", app.handleClamAVScan)

		// Suricata routes
		r.Get("/suricata/status", app.handleSuricataStatus)
		r.Get("/suricata/alerts", app.handleSuricataAlerts)

		// Telegram routes
		r.Get("/telegram/status", app.handleTelegramStatus)
		r.Post("/telegram/test", app.handleTelegramTest)
		r.Get("/telegram/conversations", app.handleTelegramConversations)

		// Parental Control routes
		r.Get("/filters", app.handleGetFilters)
		r.Post("/filters", app.handleAddFilter)
		r.Delete("/filters/{id}", app.handleDeleteFilter)
		r.Put("/filters/{id}/toggle", app.handleToggleFilter)

		r.Get("/devices/{id}/profile", app.handleGetDeviceProfile)
		r.Put("/devices/{id}/profile", app.handleSetDeviceProfile)

		// MAC-based Device Blocking routes
		r.Get("/blocked-devices", app.handleGetBlockedDevices)
		r.Post("/block-device", app.handleBlockDevice)
		r.Delete("/block-device/{mac}", app.handleUnblockDevice)
		r.Put("/blocked-device/{mac}/toggle", app.handleToggleBlockedDevice)

		r.Get("/dns-logs", app.handleGetDNSLogs)

		r.Post("/explain", app.handleExplain)

		r.Post("/actions/request-approval", app.handleRequestApproval)
		r.Post("/actions/approve", app.handleApprove)
		r.Get("/actions/pending", app.handlePendingActions)

		// User Management routes
		r.Get("/users", app.handleListUsers)
		r.Post("/users", app.handleAddUser)
		r.Delete("/users/{id}", app.handleDeleteUser)
		r.Put("/users/{id}", app.handleUpdateUser)
		r.Post("/users/change-password", app.handleChangePassword)

		// User Profile routes
		r.Get("/me", app.handleGetCurrentUser)
		r.Put("/me/profile", app.handleUpdateProfile)
		r.Put("/me/profile-picture", app.handleUpdateProfilePicture)

		r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
			app.hub.HandleWS(w, r)
		})
	})

	// Start server
	log.Printf("Bodyguard Core listening on %s", addr)
	log.Printf("API: http://%s/v1", addr)
	log.Printf("WebSocket: ws://%s/v1/ws", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

// startSensors starts the network sensors
func (a *App) startSensors() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Send initial sensor status
	a.broadcastSensorStatus()

	for range ticker.C {
		a.broadcastSensorStatus()
	}
}

// broadcastSensorStatus broadcasts the current sensor status
func (a *App) broadcastSensorStatus() {
	sensors := map[string]bool{
		"arp":  a.sensorManager != nil && a.sensorManager.IsSensorRunning("arp"),
		"dhcp": false, // Coming in v0.2
		"dns":  false, // Coming in v0.2
	}

	a.hub.Broadcast("sensor_heartbeat", map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"sensors":   sensors,
	})
}

// Handlers

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"ok": true,
		"ts": time.Now().UTC().Format(time.RFC3339),
		"version": "0.1.0",
	})
}

func (a *App) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var lastEventTS string
	_ = a.db.QueryRow(`SELECT COALESCE(MAX(ts), '') FROM events`).Scan(&lastEventTS)

	// Refresh ClamAV status for latest info
	a.clamAV.RefreshStatus()

	status := SystemStatus{
		UptimeSec:   int64(time.Since(a.startedAt).Seconds()),
		CpuLoad:     0.0, // MVP: fill later
		MemUsedMB:   int64(m.Alloc / 1024 / 1024),
		Sensors:     map[string]bool{"dhcp": true, "arp": true, "dns": a.dnsServer.IsRunning(), "suricata": a.suricata.IsRunning()},
		LastEventTS: lastEventTS,
		ClamAV:      a.clamAV.GetStatus(),
	}

	writeJSON(w, status)
}

func (a *App) handleNetworkInfo(w http.ResponseWriter, r *http.Request) {
	// Get local IP addresses
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error": "Failed to get network interfaces",
			"ips":   []string{"127.0.0.1"},
			"primary_ip": "127.0.0.1",
			"hostname": "Unknown",
			"dns_running": false,
		})
		return
	}

	var ips []string
	for _, addr := range addrs {
		// Check if the address is a valid IP address
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// Only include IPv4 addresses
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				// Exclude link-local addresses (169.254.x.x)
				if !ipnet.IP.IsLinkLocalUnicast() {
					ips = append(ips, ipv4.String())
				}
			}
		}
	}

	// If no local IPs found, at least return localhost
	primaryIP := "127.0.0.1"
	if len(ips) == 0 {
		ips = []string{"127.0.0.1"}
		log.Printf("[Network] No LAN IP found, using localhost")
	} else {
		primaryIP = ips[0]
		log.Printf("[Network] Found LAN IP: %s", primaryIP)
	}

	// Get the hostname
	hostname, _ := os.Hostname()

	// Check DNS server status
	dnsRunning := a.dnsServer != nil && a.dnsServer.IsRunning()

	writeJSON(w, map[string]interface{}{
		"hostname":       hostname,
		"ips":            ips,
		"dns_port":       53,
		"web_port":       8787,
		"primary_ip":     primaryIP,
		"dns_running":    dnsRunning,
	})
}

func (a *App) handleDevicesList(w http.ResponseWriter, r *http.Request) {
	limit := intQuery(r, "limit", 100)
	q := r.URL.Query().Get("q")

	devices, err := a.deviceDB.ListDevices(limit, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"items": devices})
}

func (a *App) handleDeviceDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	device, err := a.deviceDB.GetDeviceByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	// Get activity stats
	activity, err := a.deviceDB.GetDeviceActivity(id)
	if err == nil {
		device.ActivityStats = activity
	}

	writeJSON(w, device)
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	limit := intQuery(r, "limit", 200)
	since := r.URL.Query().Get("since")
	minSev := intQuery(r, "min_severity", 1)
	deviceID := r.URL.Query().Get("device_id")

	if since == "" {
		since = time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	}

	events, err := a.eventDB.ListEvents(limit, since, minSev, deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"items": events})
}

func (a *App) handleActiveAlerts(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`
		SELECT id, ts, device_id, severity, title, status, related_event_ids
		FROM alerts WHERE status='active'
		ORDER BY severity DESC, ts DESC
		LIMIT 50
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Alert
	for rows.Next() {
		var al Alert
		var relatedJSON string
		var device sql.NullString

		err := rows.Scan(&al.ID, &al.TS, &device, &al.Severity, &al.Title,
			&al.Status, &relatedJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if device.Valid {
			al.DeviceID = &device.String
		}

		_ = json.Unmarshal([]byte(relatedJSON), &al.RelatedEventIDs)
		items = append(items, al)
	}

	writeJSON(w, map[string]interface{}{"items": items})
}

func (a *App) handleExplain(w http.ResponseWriter, r *http.Request) {
	var req ExplainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If request body is empty, treat as general system status query
		req = ExplainRequest{}
	}

	response, err := a.aiEngine.Explain(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, response)
}

func (a *App) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	// General security analysis for the dashboard
	var req struct {
		Question string `json:"question,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	explainReq := ExplainRequest{
		Question: req.Question,
	}

	response, err := a.aiEngine.Explain(explainReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, response)
}

func (a *App) handleRequestApproval(w http.ResponseWriter, r *http.Request) {
	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	challenge, err := a.actionEng.RequestApproval(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, challenge)
}

func (a *App) handleApprove(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ApprovalID string `json:"approval_id"`
		PIN        string `json:"pin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := a.actionEng.Approve(body.ApprovalID, body.PIN, "api")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, result)
}

func (a *App) handlePendingActions(w http.ResponseWriter, r *http.Request) {
	actions, err := a.actionEng.GetPendingActions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"items": actions})
}

func (a *App) handleDeviceFingerprint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get device info first
	device, err := a.deviceDB.GetDeviceByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if device == nil {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	// Perform fingerprinting
	fingerprint := a.fingerprinter.FingerprintDevice(device.IP, device.MAC)

	writeJSON(w, map[string]interface{}{
		"device_id":  id,
		"ip":         device.IP,
		"mac":        device.MAC,
		"fingerprint": fingerprint,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

func (a *App) handleSeedDevices(w http.ResponseWriter, r *http.Request) {
	// Seed some dummy devices for testing
	now := time.Now().UTC().Format(time.RFC3339)

	dummyDevices := []DeviceDetail{
		{
			Device: Device{
				ID:        "dev-001",
				MAC:       "00:11:22:33:44:55",
				IP:        "192.168.1.100",
				Hostname:  "iPhone-12",
				Vendor:    "Apple",
				TypeGuess: "mobile",
				RiskScore: 10,
				LastSeen:  now,
			},
			FirstSeen: now,
			Tags:      []string{"mobile", "apple"},
		},
		{
			Device: Device{
				ID:        "dev-002",
				MAC:       "AA:BB:CC:DD:EE:FF",
				IP:        "192.168.1.101",
				Hostname:  "Samsung-SmartTV",
				Vendor:    "Samsung",
				TypeGuess: "iot",
				RiskScore: 35,
				LastSeen:  now,
			},
			FirstSeen: now,
			Tags:      []string{"iot", "tv"},
		},
		{
			Device: Device{
				ID:        "dev-003",
				MAC:       "11:22:33:44:55:66",
				IP:        "192.168.1.102",
				Hostname:  "MacBook-Pro",
				Vendor:    "Apple",
				TypeGuess: "laptop",
				RiskScore: 5,
				LastSeen:  now,
			},
			FirstSeen: now,
			Tags:      []string{"laptop", "apple"},
		},
		{
			Device: Device{
				ID:        "dev-004",
				MAC:       "22:33:44:55:66:77",
				IP:        "192.168.1.103",
				Hostname:  "Desktop-PC",
				Vendor:    "Dell",
				TypeGuess: "desktop",
				RiskScore: 15,
				LastSeen:  now,
			},
			FirstSeen: now,
			Tags:      []string{"desktop", "windows"},
		},
		{
			Device: Device{
				ID:        "dev-005",
				MAC:       "33:44:55:66:77:88",
				IP:        "192.168.1.1",
				Hostname:  "Router-Gateway",
				Vendor:    "TP-Link",
				TypeGuess: "router",
				RiskScore: 0,
				LastSeen:  now,
			},
			FirstSeen: now,
			Tags:      []string{"router", "infrastructure"},
		},
	}

	count := 0
	for _, dev := range dummyDevices {
		if err := a.deviceDB.UpsertDevice(dev); err != nil {
			log.Printf("Error seeding device: %v", err)
		} else {
			count++
		}
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Seeded %d devices", count),
	})
}

// Utility functions

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func intQuery(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// Honeypot handlers

func (a *App) handleHoneypotStatus(w http.ResponseWriter, r *http.Request) {
	if a.honeypot == nil {
		writeJSON(w, map[string]interface{}{
			"running": false,
			"ports":   []int{},
			"error":   "Honeypot not initialized",
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"running": a.honeypot.IsRunning(),
		"ports":   a.honeypot.GetActivePorts(),
	})
}

func (a *App) handleHoneypotHits(w http.ResponseWriter, r *http.Request) {
	if a.honeypot == nil {
		http.Error(w, "Honeypot not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := intQuery(r, "limit", 100)
	hits := a.honeypot.GetRecentHits(limit)

	writeJSON(w, map[string]interface{}{"items": hits})
}

// ClamAV handlers

func (a *App) handleClamAVStatus(w http.ResponseWriter, r *http.Request) {
	// Refresh status for latest info
	a.clamAV.RefreshStatus()
	writeJSON(w, a.clamAV.GetStatus())
}

func (a *App) handleClamAVRefresh(w http.ResponseWriter, r *http.Request) {
	a.clamAV.RefreshStatus()
	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "ClamAV status refreshed",
		"status":  a.clamAV.GetStatus(),
	})
}

func (a *App) handleClamAVScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data string `json:"data"` // Base64 encoded data to scan
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		http.Error(w, "invalid base64 data", http.StatusBadRequest)
		return
	}

	result, err := a.clamAV.ScanData(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("scan failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

// Suricata handlers

func (a *App) handleSuricataStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.suricata.GetStatus())
}

func (a *App) handleSuricataAlerts(w http.ResponseWriter, r *http.Request) {
	limit := intQuery(r, "limit", 100)
	alerts := a.suricata.GetRecentAlerts(limit)
	writeJSON(w, map[string]interface{}{"items": alerts})
}

// Parental Control handlers

func (a *App) handleGetFilters(w http.ResponseWriter, r *http.Request) {
	rules := a.filterEngine.GetRules()
	writeJSON(w, map[string]interface{}{"items": rules})
}

func (a *App) handleAddFilter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Pattern string `json:"pattern"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Type == "" || req.Pattern == "" {
		http.Error(w, "name, type, and pattern are required", http.StatusBadRequest)
		return
	}

	if req.Type != "domain" && req.Type != "category" {
		http.Error(w, "type must be 'domain' or 'category'", http.StatusBadRequest)
		return
	}

	rule, err := a.filterEngine.AddRule(req.Name, req.Type, req.Pattern)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, rule)
}

func (a *App) handleDeleteFilter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := a.filterEngine.RemoveRule(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"success": true})
}

func (a *App) handleToggleFilter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := a.filterEngine.EnableRule(id, req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"success": true, "enabled": req.Enabled})
}

func (a *App) handleGetDeviceProfile(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")

	filterLevel, err := a.filterEngine.GetDeviceProfile(deviceID)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"device_id":    deviceID,
		"filter_level": filterLevel,
	})
}

func (a *App) handleSetDeviceProfile(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")

	var req struct {
		FilterLevel string `json:"filter_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.FilterLevel == "" {
		http.Error(w, "filter_level is required", http.StatusBadRequest)
		return
	}

	validLevels := map[string]bool{
		"off":      true,
		"light":    true,
		"moderate": true,
		"strict":   true,
	}
	if !validLevels[req.FilterLevel] {
		http.Error(w, "filter_level must be one of: off, light, moderate, strict", http.StatusBadRequest)
		return
	}

	if err := a.filterEngine.SetDeviceProfile(deviceID, req.FilterLevel); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"success":     true,
		"device_id":   deviceID,
		"filter_level": req.FilterLevel,
	})
}

// MAC-based Device Blocking handlers

func (a *App) handleGetBlockedDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`
		SELECT id, mac_address, device_name, blocked, block_reason, blocked_at, blocked_by
		FROM blocked_devices
		ORDER BY blocked_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var devices []map[string]interface{}
	for rows.Next() {
		var id, mac, name, reason, blockedBy string
		var blocked bool
		var blockedAt string
		if err := rows.Scan(&id, &mac, &name, &blocked, &reason, &blockedAt, &blockedBy); err != nil {
			continue
		}
		devices = append(devices, map[string]interface{}{
			"id":          id,
			"mac_address": mac,
			"device_name": name,
			"blocked":     blocked,
			"block_reason": reason,
			"blocked_at":  blockedAt,
			"blocked_by":   blockedBy,
		})
	}

	writeJSON(w, map[string]interface{}{"items": devices})
}

func (a *App) handleBlockDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MACAddress  string `json:"mac_address"`
		DeviceName  string `json:"device_name"`
		Reason       string `json:"reason"`
		BlockReason  string `json:"block_reason"`
		BlockedBy    string `json:"blocked_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MACAddress == "" {
		http.Error(w, "mac_address is required", http.StatusBadRequest)
		return
	}

	// Validate MAC address format
	mac := strings.ToUpper(strings.ReplaceAll(req.MACAddress, "-", ":"))
	if len(mac) != 12 {
		http.Error(w, "invalid MAC address format", http.StatusBadRequest)
		return
	}

	// Format as XX:XX:XX:XX:XX:XX
	formattedMAC := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])

	// Use block_reason if reason is not provided
	reason := req.Reason
	if reason == "" {
		reason = req.BlockReason
	}
	if reason == "" {
		reason = "Blocked"
	}

	// Check if already blocked
	var existingCount int
	err := a.db.QueryRow(`SELECT COUNT(*) FROM blocked_devices WHERE mac_address = ?`, formattedMAC).Scan(&existingCount)
	if err == nil && existingCount > 0 {
		// Already blocked, update instead
		_, err = a.db.Exec(`
			UPDATE blocked_devices
			SET blocked = 1, block_reason = ?, blocked_by = ?, blocked_at = datetime('now')
			WHERE mac_address = ?
		`, reason, req.BlockedBy, formattedMAC)
	} else {
		// Insert new blocked device
		id := fmt.Sprintf("block-%d", time.Now().UnixNano())
		_, err = a.db.Exec(`
			INSERT INTO blocked_devices (id, mac_address, device_name, blocked, block_reason, blocked_at, blocked_by)
			VALUES (?, ?, ?, 1, ?, datetime('now'), ?)
		`, id, formattedMAC, req.DeviceName, reason, req.BlockedBy)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create event
	event := Event{
		ID:       fmt.Sprintf("evt-block-%d", time.Now().UnixNano()),
		TS:       time.Now().Format(time.RFC3339),
		DeviceID: nil,
		Category: "parental",
		Severity: 50,
		Summary:  fmt.Sprintf("Device blocked: %s (%s)", req.DeviceName, formattedMAC),
		Raw: map[string]interface{}{
			"mac_address":  formattedMAC,
			"device_name":  req.DeviceName,
			"action":       "block",
			"blocked_by":   req.BlockedBy,
		},
	}
	a.eventDB.CreateEvent(event)

	// Broadcast to WebSocket
	a.hub.Broadcast("device_blocked", map[string]interface{}{
		"mac_address": formattedMAC,
		"device_name": req.DeviceName,
		"action":       "blocked",
		"blocked_by":   req.BlockedBy,
		"timestamp":     event.TS,
	})

	writeJSON(w, map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("Device %s (%s) has been blocked", req.DeviceName, formattedMAC),
		"mac_address": formattedMAC,
	})
}

func (a *App) handleUnblockDevice(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")

	if mac == "" {
		http.Error(w, "mac address is required", http.StatusBadRequest)
		return
	}

	// Format MAC address
	mac = strings.ToUpper(strings.ReplaceAll(mac, "-", ":"))
	if len(mac) != 12 {
		http.Error(w, "invalid MAC address format", http.StatusBadRequest)
		return
	}

	formattedMAC := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])

	// Get device info before unblocking
	var deviceName string
	err := a.db.QueryRow(`SELECT device_name FROM blocked_devices WHERE mac_address = ?`, formattedMAC).Scan(&deviceName)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "device not found in blocklist", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Unblock device
	_, err = a.db.Exec(`
		UPDATE blocked_devices
		SET blocked = 0, unblocked_at = datetime('now')
		WHERE mac_address = ?
	`, formattedMAC)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create event
	event := Event{
		ID:       fmt.Sprintf("evt-unblock-%d", time.Now().UnixNano()),
		TS:       time.Now().Format(time.RFC3339),
		DeviceID: nil,
		Category: "parental",
		Severity: 30,
		Summary:  fmt.Sprintf("Device unblocked: %s (%s)", deviceName, formattedMAC),
		Raw: map[string]interface{}{
			"mac_address": formattedMAC,
			"device_name": deviceName,
			"action":      "unblock",
		},
	}
	a.eventDB.CreateEvent(event)

	// Broadcast to WebSocket
	a.hub.Broadcast("device_unblocked", map[string]interface{}{
		"mac_address": formattedMAC,
		"device_name": deviceName,
		"action":       "unblocked",
		"timestamp":     event.TS,
	})

	writeJSON(w, map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("Device %s (%s) has been unblocked", deviceName, formattedMAC),
		"mac_address": formattedMAC,
	})
}

func (a *App) handleToggleBlockedDevice(w http.ResponseWriter, r *http.Request) {
	mac := chi.URLParam(r, "mac")

	if mac == "" {
		http.Error(w, "mac address is required", http.StatusBadRequest)
		return
	}

	// Format MAC address
	mac = strings.ToUpper(strings.ReplaceAll(mac, "-", ":"))
	if len(mac) != 12 {
		http.Error(w, "invalid MAC address format", http.StatusBadRequest)
		return
	}

	formattedMAC := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])

	// Check current status
	var currentBlocked bool
	var deviceName string
	err := a.db.QueryRow(`SELECT blocked, device_name FROM blocked_devices WHERE mac_address = ?`, formattedMAC).Scan(&currentBlocked, &deviceName)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Toggle
	newStatus := !currentBlocked
	var action string
	if newStatus {
		action = "blocked"
	} else {
		action = "unblocked"
	}

	_, err = a.db.Exec(`
		UPDATE blocked_devices
		SET blocked = ?,
		    ${action}_at = datetime('now')
		WHERE mac_address = ?
	`, newStatus, formattedMAC)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create event
	severity := 50
	if !newStatus {
		severity = 30
	}

	event := Event{
		ID:       fmt.Sprintf("evt-toggle-block-%d", time.Now().UnixNano()),
		TS:       time.Now().Format(time.RFC3339),
		DeviceID: nil,
		Category: "parental",
		Severity: severity,
		Summary:  fmt.Sprintf("Device %s: %s", deviceName, action),
		Raw: map[string]interface{}{
			"mac_address": formattedMAC,
			"device_name": deviceName,
			"action":      action,
		},
	}
	a.eventDB.CreateEvent(event)

	// Broadcast to WebSocket
	a.hub.Broadcast("device_toggled", map[string]interface{}{
		"mac_address": formattedMAC,
		"device_name": deviceName,
		"blocked":     newStatus,
		"timestamp":   event.TS,
	})

	writeJSON(w, map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("Device %s (%s) is now %s", deviceName, formattedMAC, action),
		"mac_address": formattedMAC,
		"blocked":     newStatus,
	})
}

func (a *App) handleGetDNSLogs(w http.ResponseWriter, r *http.Request) {
	limit := intQuery(r, "limit", 100)

	logs, err := a.filterEngine.GetDNSQueryLogs(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"items": logs})
}

func (a *App) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	if a.telegram == nil {
		writeJSON(w, map[string]interface{}{
			"enabled":   false,
			"configured": false,
		})
		return
	}

	status := a.telegram.GetStatus()
	writeJSON(w, status)
}

func (a *App) handleTelegramTest(w http.ResponseWriter, r *http.Request) {
	if a.telegram == nil {
		http.Error(w, "Telegram service not initialized", http.StatusServiceUnavailable)
		return
	}

	if err := a.telegram.SendTestMessage(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Test message sent to Telegram",
	})
}

func (a *App) handleTelegramConversations(w http.ResponseWriter, r *http.Request) {
	if a.telegram == nil {
		http.Error(w, "Telegram service not initialized", http.StatusServiceUnavailable)
		return
	}

	conversations := a.telegram.GetConversations()
	writeJSON(w, conversations)
}
