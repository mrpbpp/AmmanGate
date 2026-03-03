package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// HoneypotConfig defines ports to monitor
type HoneypotConfig struct {
	Port         int
	Name         string
	Description  string
	ShouldLog    bool
}

// Default honeypot ports - commonly attacked services
var defaultHoneypotPorts = []HoneypotConfig{
	{Port: 23, Name: "telnet", Description: "Fake Telnet Server", ShouldLog: true},
	{Port: 135, Name: "msrpc", Description: "Fake MSRPC Endpoint", ShouldLog: true},
	{Port: 139, Name: "netbios", Description: "Fake NetBIOS Service", ShouldLog: true},
	{Port: 445, Name: "smb", Description: "Fake SMB Server", ShouldLog: true},
	{Port: 1433, Name: "mssql", Description: "Fake SQL Server", ShouldLog: true},
	{Port: 3306, Name: "mysql", Description: "Fake MySQL Server", ShouldLog: true},
	{Port: 3389, Name: "rdp", Description: "Fake RDP Service", ShouldLog: true},
	{Port: 5900, Name: "vnc", Description: "Fake VNC Server", ShouldLog: true},
	{Port: 8080, Name: "http-alt", Description: "Fake HTTP Alt", ShouldLog: true},
	{Port: 25565, Name: "minecraft", Description: "Fake Minecraft Server", ShouldLog: true},
}

// HoneypotHit represents a detected connection attempt
type HoneypotHit struct {
	ID          string                `json:"id"`
	Timestamp   time.Time             `json:"timestamp"`
	Port        int                   `json:"port"`
	Service     string                `json:"service"`
	RemoteIP    string                `json:"remote_ip"`
	RemoteMAC   string                `json:"remote_mac,omitempty"`
	LocalIP     string                `json:"local_ip,omitempty"`
	LocalPort   int                   `json:"local_port,omitempty"`
	Data        string                `json:"data,omitempty"`
	Severity    int                   `json:"severity"`
	GeoLocation *GeoLocation          `json:"geo_location,omitempty"`
	Raw         map[string]interface{} `json:"raw,omitempty"`
}

// HoneypotManager manages all honeypot listeners
type HoneypotManager struct {
	db        *sql.DB
	devDB     *DeviceDB
	eventDB   *EventDB
	hub       *WSHub
	aiEngine  *AIEngine
	geoLookup *GeoLookup
	hits      []HoneypotHit
	hitsMu    sync.RWMutex
	listeners map[int]net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	mu        sync.RWMutex
}

// NewHoneypotManager creates a new honeypot manager
func NewHoneypotManager(db *sql.DB, devDB *DeviceDB, eventDB *EventDB, hub *WSHub, aiEngine *AIEngine, geoLookup *GeoLookup) *HoneypotManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &HoneypotManager{
		db:        db,
		devDB:     devDB,
		eventDB:   eventDB,
		hub:       hub,
		aiEngine:  aiEngine,
		geoLookup: geoLookup,
		hits:      make([]HoneypotHit, 0, 100),
		listeners: make(map[int]net.Listener),
		ctx:       ctx,
		cancel:    cancel,
		running:   false,
	}
}

// Start begins listening on all honeypot ports
func (h *HoneypotManager) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return fmt.Errorf("honeypot already running")
	}

	log.Printf("Starting Honeypot Manager with %d ports...", len(defaultHoneypotPorts))

	for _, config := range defaultHoneypotPorts {
		if err := h.startListener(config); err != nil {
			log.Printf("Warning: Failed to start honeypot on port %d: %v", config.Port, err)
			// Continue with other ports
		}
	}

	h.running = true
	go h.cleanupRoutine()

	return nil
}

// Stop stops all honeypot listeners
func (h *HoneypotManager) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	log.Println("Stopping Honeypot Manager...")
	h.cancel()
	h.running = false

	// Close all listeners
	for port, listener := range h.listeners {
		log.Printf("Closing honeypot on port %d", port)
		listener.Close()
	}

	h.listeners = make(map[int]net.Listener)
}

// IsRunning returns whether honeypot is running
func (h *HoneypotManager) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

// GetActivePorts returns list of active honeypot ports
func (h *HoneypotManager) GetActivePorts() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ports := make([]int, 0, len(h.listeners))
	for port := range h.listeners {
		ports = append(ports, port)
	}
	return ports
}

// GetRecentHits returns recent honeypot hits
func (h *HoneypotManager) GetRecentHits(limit int) []HoneypotHit {
	h.hitsMu.RLock()
	defer h.hitsMu.RUnlock()

	if limit > len(h.hits) {
		limit = len(h.hits)
	}

	if limit == 0 {
		return h.hits
	}

	// Return most recent hits
	start := len(h.hits) - limit
	return h.hits[start:]
}

// startListener starts a single honeypot listener
func (h *HoneypotManager) startListener(config HoneypotConfig) error {
	addr := fmt.Sprintf("0.0.0.0:%d", config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", addr, err)
	}

	h.listeners[config.Port] = listener
	log.Printf("[HONEYPOT] Listening on port %d (%s)", config.Port, config.Name)

	go h.handleConnection(config, listener)

	return nil
}

// handleConnection handles incoming connections to a honeypot port
func (h *HoneypotManager) handleConnection(config HoneypotConfig, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-h.ctx.Done():
				return
			default:
				log.Printf("[HONEYPOT] Error accepting on port %d: %v", config.Port, err)
				continue
			}
		}

		go h.handleHit(config, conn)
	}
}

// handleHit processes a single honeypot hit
func (h *HoneypotManager) handleHit(config HoneypotConfig, conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
	remoteIP := remoteAddr.IP.String()
	remotePort := remoteAddr.Port

	// Get the local IP that received this connection (LAN IP)
	localAddr := conn.LocalAddr().(*net.TCPAddr)
	localIP := localAddr.IP.String()

	// If local IP is 127.0.0.1 or ::1, try to find actual LAN IP
	if localIP == "127.0.0.1" || localIP == "::1" {
		localIP = getLANIP()
	}

	// Set read timeout to prevent hanging
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Try to read initial data (may fail for connection-only scans)
	var data string
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err == nil && n > 0 {
		data = string(buf[:n])
		// Check for common exploit patterns
		data = sanitizeData(data)
	}

	// Try to find MAC address from device DB
	var remoteMAC string
	if device, err := h.devDB.GetDeviceByIP(remoteIP); err == nil && device != nil {
		remoteMAC = device.MAC
	}

	// Perform geolocation lookup for the remote IP
	var geoLocation *GeoLocation
	if h.geoLookup != nil {
		if geo, err := h.geoLookup.Lookup(remoteIP); err == nil {
			geoLocation = &geo
			log.Printf("[HONEYPOT] Geo lookup for %s: %s (%s)",
				remoteIP, geo.FormatLocation(), geo.FormatISP())
		} else {
			log.Printf("[HONEYPOT] Geo lookup failed for %s: %v", remoteIP, err)
		}
	}

	// Create hit record with LAN IP info
	hit := HoneypotHit{
		ID:          generateHitID(),
		Timestamp:   time.Now().UTC(),
		Port:        config.Port,
		Service:     config.Name,
		RemoteIP:    remoteIP,
		RemoteMAC:   remoteMAC,
		LocalIP:     localIP,
		LocalPort:   localAddr.Port,
		Data:        data,
		Severity:    calculateHoneypotSeverity(config, data),
		GeoLocation: geoLocation,
		Raw: map[string]interface{}{
			"local_ip":   localIP,
			"local_port": localAddr.Port,
		},
	}

	// Store hit
	h.hitsMu.Lock()
	h.hits = append(h.hits, hit)
	// Keep only last 1000 hits
	if len(h.hits) > 1000 {
		h.hits = h.hits[len(h.hits)-1000:]
	}
	h.hitsMu.Unlock()

	// Log the hit
	log.Printf("[HONEYPOT] Hit on port %d (%s) from %s:%d - Data: %q",
		config.Port, config.Name, remoteIP, remotePort, truncateString(data, 50))

	// Create security event
	h.createEvent(hit)

	// Trigger AI analysis asynchronously
	if h.aiEngine != nil {
		go func() {
			// Build context - this is a HONEYPOT detection, not general network analysis
			question := fmt.Sprintf(
				"**HONEYPOT ALERT** - This is a honeypot detection, not general network activity!\n\n"+
				"A HONEYPOT (decoy service) was hit by an external attacker. Analyze this SPECIFIC threat:\n\n"+
				"**Attack Details:**\n"+
				"- Attacker IP: %s\n"+
				"- Target Port: %d (%s)\n"+
				"- Attack Type: Port scanning/service probing\n"+
				"- Severity: %d/100\n",
				remoteIP, hit.Port, hit.Service, hit.Severity,
			)

			// Add geolocation info if available
			if hit.GeoLocation != nil {
				question += fmt.Sprintf(
					"\n**Attacker Location:** %s\n"+
					"**ISP:** %s\n",
					hit.GeoLocation.FormatLocation(), hit.GeoLocation.FormatISP(),
				)
				if hit.GeoLocation.IsRiskyConnection() {
					question += "**WARNING:** Attacker is using VPN/Proxy/Tor!\n"
				}
			}

			// Add context about what honeypot is
			question += "\n"+
				"**Context:** A honeypot is a decoy service that detects malicious scanning. "+
				"Since this IP connected to our honeypot, they are actively scanning for vulnerabilities.\n\n"+
				"**Your Task:** Analyze THIS SPECIFIC ATTACKER and threat. Do NOT give generic network security advice. "+
				"Focus on:\n"+
				"1. What type of attacker is this (script kiddie, automated bot, targeted attack)?\n"+
				"2. What was their likely intent (port scanning, exploit attempt, reconnaissance)?\n"+
				"3. What should the user do about THIS SPECIFIC THREAT (block IP, monitor, ignore)?"

			explainReq := ExplainRequest{
				Question: question,
			}

			// Call AI for analysis
			response, err := h.aiEngine.Explain(explainReq)
			if err != nil {
				log.Printf("AI analysis failed for honeypot hit: %v", err)
				return
			}

			// Log AI analysis result
			log.Printf("[HONEYPOT-AI] AI Analysis: %s - Confidence: %.0f",
				response.SuspectedCause, response.Confidence)

			// Broadcast AI analysis to WebSocket for dashboard
			broadcastData := map[string]interface{}{
				"timestamp":       time.Now().UTC().Format(time.RFC3339),
				"honeypot_hit_id": hit.ID,
				"remote_ip":       hit.RemoteIP,
				"port":            hit.Port,
				"service":         hit.Service,
				"ai_analysis":     response,
			}

			// Add geolocation to broadcast if available
			if hit.GeoLocation != nil {
				broadcastData["geo_location"] = map[string]interface{}{
					"country":    hit.GeoLocation.CountryName,
					"city":       hit.GeoLocation.City,
					"region":     hit.GeoLocation.RegionName,
					"isp":        hit.GeoLocation.ISP,
					"is_risky":   hit.GeoLocation.IsRiskyConnection(),
					"formatted":  hit.GeoLocation.FormatLocation(),
				}
			}

			h.hub.Broadcast("honeypot_ai_analysis", broadcastData)
		}()
	}

	// Broadcast to WebSocket with geolocation info
	broadcastData := map[string]interface{}{
		"timestamp":  hit.Timestamp.Format(time.RFC3339),
		"port":       hit.Port,
		"service":    hit.Service,
		"remote_ip":  hit.RemoteIP,
		"remote_mac": hit.RemoteMAC,
		"severity":   hit.Severity,
	}

	// Add geolocation to broadcast if available
	if hit.GeoLocation != nil {
		broadcastData["geo_location"] = map[string]interface{}{
			"country":    hit.GeoLocation.CountryName,
			"city":       hit.GeoLocation.City,
			"region":     hit.GeoLocation.RegionName,
			"isp":        hit.GeoLocation.ISP,
			"is_risky":   hit.GeoLocation.IsRiskyConnection(),
			"formatted":  hit.GeoLocation.FormatLocation(),
		}
	}

	h.hub.Broadcast("honeypot_hit", broadcastData)

	// Send fake response to keep them engaged (optional)
	h.sendFakeResponse(config, conn)
}

// sendFakeResponse sends a fake service response
func (h *HoneypotManager) sendFakeResponse(config HoneypotConfig, conn net.Conn) {
	var response string

	switch config.Name {
	case "telnet":
		response = "Welcome to Telnet Server\r\nLogin: "
	case "ftp":
		response = "220 FTP Server ready\r\n"
	case "smtp":
		response = "220 SMTP Server ready\r\n"
	case "http-alt":
		response = "HTTP/1.1 401 Unauthorized\r\nWWW-Authenticate: Basic realm=\"Secure Area\"\r\n\r\n"
	case "smb", "netbios":
		// SMB requires binary protocol, just close
		return
	case "rdp":
		// RDP requires binary handshake, just close
		return
	default:
		response = fmt.Sprintf("Connected to %s\r\n", config.Description)
	}

	if response != "" {
		conn.Write([]byte(response))
	}
}

// createEvent creates a security event for the honeypot hit
func (h *HoneypotManager) createEvent(hit HoneypotHit) {
	summary := fmt.Sprintf("Honeypot hit on port %d (%s) from %s",
		hit.Port, hit.Service, hit.RemoteIP)

	raw := map[string]interface{}{
		"honeypot_hit": true,
		"port":         hit.Port,
		"service":      hit.Service,
		"remote_ip":    hit.RemoteIP,
		"remote_port":  "unknown",
		"data":         hit.Data,
	}

	// Add geolocation info if available
	if hit.GeoLocation != nil {
		raw["geo_country"] = hit.GeoLocation.CountryName
		raw["geo_city"] = hit.GeoLocation.City
		raw["geo_region"] = hit.GeoLocation.RegionName
		raw["geo_isp"] = hit.GeoLocation.ISP
		raw["geo_formatted"] = hit.GeoLocation.FormatLocation()
		raw["geo_is_risky"] = hit.GeoLocation.IsRiskyConnection()
	}

	// Check if we should create an alert (high severity hits)
	if hit.Severity >= 70 {
		h.createAlert(hit)
	}

	// Generate event ID using format
	eventID := fmt.Sprintf("evt-%d", hit.Timestamp.UnixNano())

	// Store event - use nil for device_id since this is an unknown attacker
	event := Event{
		ID:       eventID,
		TS:       hit.Timestamp.Format(time.RFC3339),
		DeviceID: nil, // Unknown attacker, no device record
		Category: "honeypot",
		Severity: hit.Severity,
		Summary:  summary,
		Raw:      raw,
	}

	err := h.eventDB.CreateEvent(event)
	if err != nil {
		log.Printf("Error creating honeypot event: %v", err)
	}
}

// createAlert creates an alert for critical honeypot hits
func (h *HoneypotManager) createAlert(hit HoneypotHit) {
	// Check if alert already exists for this IP+port combination recently
	var existingCount int
	err := h.db.QueryRow(`
		SELECT COUNT(*) FROM alerts
		WHERE status='active'
		AND title LIKE ?
		AND ts > datetime('now', '-1 hour')
	`, "%"+hit.RemoteIP+"% port "+fmt.Sprintf("%d", hit.Port)).Scan(&existingCount)

	if err == nil && existingCount > 0 {
		// Alert already exists, don't duplicate
		return
	}

	title := fmt.Sprintf("Honeypot Alert: %s scanning port %d (%s)",
		hit.RemoteIP, hit.Port, hit.Service)

	relatedEvents := []string{hit.ID}

	_, err = h.db.Exec(`
		INSERT INTO alerts (id, ts, device_id, severity, title, status, related_event_ids)
		VALUES (?, ?, NULL, ?, ?, 'active', ?)
	`, generateAlertID(), hit.Timestamp.Format(time.RFC3339),
		hit.Severity, title, jsonToString(relatedEvents))

	if err != nil {
		log.Printf("Error creating honeypot alert: %v", err)
	}
}

// calculateHoneypotSeverity calculates severity based on port and data
func calculateHoneypotSeverity(config HoneypotConfig, data string) int {
	baseSeverity := 50

	// Critical ports - more concerning
	switch config.Port {
	case 23, 135, 139, 445: // Windows/SMB exploits
		baseSeverity = 70
	case 3389: // RDP - very concerning
		baseSeverity = 80
	case 1433, 3306: // Database attacks
		baseSeverity = 75
	}

	// Increase severity if exploit data detected
	if data != "" {
		dataLower := strings.ToLower(data)
		if strings.Contains(dataLower, "exploit") ||
			strings.Contains(dataLower, "../") ||
			strings.Contains(dataLower, "union select") ||
			strings.Contains(dataLower, "cmd.exe") ||
			strings.Contains(dataLower, "/bin/sh") {
			baseSeverity += 20
		}
	}

	if baseSeverity > 100 {
		baseSeverity = 100
	}

	return baseSeverity
}

// cleanupRoutine periodically cleans up old hits
func (h *HoneypotManager) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.hitsMu.Lock()
			cutoff := time.Now().UTC().Add(-24 * time.Hour)
			newHits := make([]HoneypotHit, 0, len(h.hits))
			for _, hit := range h.hits {
				if hit.Timestamp.After(cutoff) {
					newHits = append(newHits, hit)
				}
			}
			h.hits = newHits
			h.hitsMu.Unlock()
		}
	}
}

// Helper functions

// getLANIP returns the first non-loopback IPv4 address
func getLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				// Exclude link-local addresses (169.254.x.x)
				if !ipnet.IP.IsLinkLocalUnicast() {
					return ipv4.String()
				}
			}
		}
	}

	return "127.0.0.1"
}

func generateHitID() string {
	return fmt.Sprintf("hp-%d", time.Now().UnixNano())
}

func generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}

func sanitizeData(data string) string {
	// Remove null bytes and limit length
	data = strings.ReplaceAll(data, "\x00", "")
	if len(data) > 200 {
		data = data[:200] + "..."
	}
	return strings.TrimSpace(data)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func jsonToString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
