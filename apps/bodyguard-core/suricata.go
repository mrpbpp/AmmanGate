package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SuricataManager manages Suricata IDS integration
type SuricataManager struct {
	db           *sql.DB
	eventDB      *EventDB
	hub          *WSHub
	telegram     *TelegramService
	enabled      bool
	socketPath   string
	eveFilePath  string
	version      string
	mu           sync.RWMutex
	alerts       []SuricataAlert
	alertsMu     sync.RWMutex
	running      bool
	ctx          context.Context
	cancel       context.CancelFunc
	stats        SuricataStats
	statsMu      sync.RWMutex
	sentAlerts   map[string]time.Time // Track sent alerts with timestamp for rate limiting
	sentAlertsMu sync.RWMutex
	lastPosition int64                  // Track last file position
	alertCooldown time.Duration         // Cooldown period for same alert type
}

// SuricataAlert represents a Suricata IDS alert
type SuricataAlert struct {
	Timestamp   time.Time              `json:"timestamp"`
	AlertID     string                 `json:"alert_id"`
	GID         int                    `json:"gid"`
	SignatureID int                    `json:"signature_id"`
	Rev         int                    `json:"rev"`
	Signature   string                 `json:"signature"`
	Category    string                 `json:"category"`
	Severity    int                    `json:"severity"`
	SrcIP       string                 `json:"src_ip"`
	SrcPort     int                    `json:"src_port"`
	DestIP      string                 `json:"dest_ip"`
	DestPort    int                    `json:"dest_port"`
	Proto       string                 `json:"proto"`
	PacketInfo  map[string]interface{} `json:"packet_info,omitempty"`
	GeoLocation *GeoLocation           `json:"geo_location,omitempty"`
}

// SuricataStats represents Suricata statistics
type SuricataStats struct {
	TotalAlerts    int64     `json:"total_alerts"`
	HighSeverity   int64     `json:"high_severity"`
	MediumSeverity int64     `json:"medium_severity"`
	LowSeverity    int64     `json:"low_severity"`
	LastAlert      time.Time `json:"last_alert"`
	PacketsSeen    int64     `json:"packets_seen"`
	BytesSeen      int64     `json:"bytes_seen"`
	StartTime      time.Time `json:"start_time"`
}

// EVE JSON format from Suricata
type EVEEvent struct {
	Timestamp string `json:"timestamp"`
	EventType string `json:"event_type"`
	SrcIP     string `json:"src_ip"`
	SrcPort   int    `json:"src_port"`
	DestIP    string `json:"dest_ip"`
	DestPort  int    `json:"dest_port"`
	Protocol  string `json:"proto"`
	Alert     *struct {
		Action      string `json:"action"`
		GID         int    `json:"gid"`
		SignatureID int    `json:"signature_id"`
		Rev         int    `json:"rev"`
		Signature   string `json:"signature"`
		Category    string `json:"category"`
		Severity    int    `json:"severity"`
	} `json:"alert"`
}

// NewSuricataManager creates a new Suricata manager
func NewSuricataManager(db *sql.DB, eventDB *EventDB, hub *WSHub) *SuricataManager {
	// Default EVE log path for Windows
	defaultEVEPath := "C:\\Suricata\\log\\eve.json"

	sm := &SuricataManager{
		db:          db,
		eventDB:     eventDB,
		hub:         hub,
		enabled:     env("SURICATA_ENABLED", "false") == "true",
		socketPath:  env("SURICATA_SOCKET", ""),
		eveFilePath: env("SURICATA_EVE_LOG", defaultEVEPath),
		version:     detectVersion(),
		alerts:      make([]SuricataAlert, 0, 1000),
		sentAlerts:     make(map[string]time.Time),
		alertCooldown:  5 * time.Minute, // 5 minute cooldown for same alert
		stats: SuricataStats{
			StartTime: time.Now(),
		},
	}

	return sm
}

// detectVersion attempts to detect the Suricata version
func detectVersion() string {
	// Try to get version from suricata.exe --version
	out, err := exec.Command("C:\\Program Files\\Suricata\\suricata.exe", "--version").Output()
	if err == nil {
		// Output format: "Suricata version X.Y.Z RELEASE"
		parts := strings.Split(string(out), " ")
		for i, part := range parts {
			if part == "version" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}

	// Fallback: try to parse from log file if it exists
	logPath := "d:\\AmmanGate\\suricata-logs\\suricata.log"
	if data, err := os.ReadFile(logPath); err == nil {
		// Look for version pattern in log: "This is Suricata version X.Y.Z"
		re := regexp.MustCompile(`This is Suricata version ([\d.]+)`)
		matches := re.FindSubmatch(data)
		if len(matches) > 1 {
			return string(matches[1])
		}
	}

	return "Unknown"
}

// Start begins monitoring Suricata
func (s *SuricataManager) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("suricata manager already running")
	}

	if !s.enabled {
		log.Println("[Suricata] Suricata integration is disabled")
		return nil
	}

	log.Println("[Suricata] Starting Suricata Manager...")

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	s.running = true

	// Start EVE log file monitoring
	go s.monitorEVELog()

	// Start statistics collector
	go s.collectStats()

	return nil
}

// Stop stops the Suricata manager
func (s *SuricataManager) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	log.Println("[Suricata] Stopping Suricata Manager...")
	s.running = false

	if s.cancel != nil {
		s.cancel()
	}
}

// IsRunning returns whether Suricata manager is running
func (s *SuricataManager) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// SetEnabled enables or disables Suricata monitoring
func (s *SuricataManager) SetEnabled(enabled bool) error {
	s.mu.Lock()
	wasRunning := s.running
	wasEnabled := s.enabled
	s.enabled = enabled
	s.mu.Unlock()

	if enabled {
		log.Println("[Suricata] Suricata integration enabled")
		if !wasRunning && wasEnabled {
			return s.Start()
		}
	} else {
		log.Println("[Suricata] Suricata integration disabled")
		if wasRunning {
			s.Stop()
		}
	}
	return nil
}

// monitorEVELog monitors the Suricata EVE JSON log file
func (s *SuricataManager) monitorEVELog() {
	// Open the EVE log file
	file, err := os.Open(s.eveFilePath)
	if err != nil {
		log.Printf("[Suricata] Failed to open EVE log: %v", err)
		log.Printf("[Suricata] EVE log path: %s", s.eveFilePath)
		return
	}
	defer file.Close()

	// Seek to end of file to start reading new entries
	_, err = file.Seek(0, 2)
	if err != nil {
		log.Printf("[Suricata] Failed to seek to end: %v", err)
		return
	}

	reader := bufio.NewReader(file)

	log.Printf("[Suricata] Monitoring EVE log: %s", s.eveFilePath)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse EVE event
		s.processEVEEvent(line)
	}
}

// processEVEEvent processes a single EVE JSON event
func (s *SuricataManager) processEVEEvent(line string) {
	var eveEvent EVEEvent
	if err := json.Unmarshal([]byte(line), &eveEvent); err != nil {
		log.Printf("[Suricata] Failed to parse EVE event: %v", err)
		return
	}

	// We're only interested in alert events
	if eveEvent.EventType != "alert" || eveEvent.Alert == nil {
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339Nano, eveEvent.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	// Create a unique ID based on alert content for deduplication
	// Using signature + source + destination + timestamp (rounded to second)
	uniqueID := fmt.Sprintf("%s|%s|%d|%s|%d|%d",
		eveEvent.Alert.Signature,
		eveEvent.SrcIP,
		eveEvent.SrcPort,
		eveEvent.DestIP,
		eveEvent.DestPort,
		timestamp.Unix())

	// Create SuricataAlert
	alert := SuricataAlert{
		Timestamp:   timestamp,
		AlertID:     fmt.Sprintf("suricata-%d", time.Now().UnixNano()),
		GID:         eveEvent.Alert.GID,
		SignatureID: eveEvent.Alert.SignatureID,
		Rev:         eveEvent.Alert.Rev,
		Signature:   eveEvent.Alert.Signature,
		Category:    eveEvent.Alert.Category,
		Severity:    eveEvent.Alert.Severity,
		SrcIP:       eveEvent.SrcIP,
		SrcPort:     eveEvent.SrcPort,
		DestIP:      eveEvent.DestIP,
		DestPort:    eveEvent.DestPort,
		Proto:       eveEvent.Protocol,
		PacketInfo: map[string]interface{}{
			"action": eveEvent.Alert.Action,
		},
	}

	// Add geolocation info
	if geo, err := geoLookup.Lookup(alert.SrcIP); err == nil {
		alert.GeoLocation = &geo
	}

	// Store alert
	s.alertsMu.Lock()
	s.alerts = append(s.alerts, alert)
	// Keep only last 1000 alerts
	if len(s.alerts) > 1000 {
		s.alerts = s.alerts[len(s.alerts)-1000:]
	}
	s.alertsMu.Unlock()

	// Update stats
	s.statsMu.Lock()
	s.stats.TotalAlerts++
	s.stats.LastAlert = timestamp
	switch {
	case alert.Severity >= 3:
		s.stats.HighSeverity++
	case alert.Severity == 2:
		s.stats.MediumSeverity++
	default:
		s.stats.LowSeverity++
	}
	s.statsMu.Unlock()

	// Log the alert
	log.Printf("[Suricata] ALERT: %s from %s:%d -> %s:%d (Severity: %d)",
		alert.Signature, alert.SrcIP, alert.SrcPort,
		alert.DestIP, alert.DestPort, alert.Severity)

	// Check if this alert was already sent to Telegram recently (within cooldown period)
	s.sentAlertsMu.RLock()
	lastSent, exists := s.sentAlerts[uniqueID]
	s.sentAlertsMu.RUnlock()

	// Only send if not sent within cooldown period
	shouldSend := !exists || time.Since(lastSent) > s.alertCooldown

	if shouldSend {
		s.mu.RLock()
		telegram := s.telegram
		s.mu.RUnlock()

		if telegram != nil && telegram.IsEnabled() {
			go func(a SuricataAlert) {
				if err := telegram.SendAlert(&a); err != nil {
					log.Printf("[Suricata] Failed to send Telegram alert: %v", err)
				} else {
					log.Printf("[Suricata] Telegram alert sent for: %s", a.Signature)
					// Mark as sent with current timestamp
					s.sentAlertsMu.Lock()
					s.sentAlerts[uniqueID] = time.Now()
					s.sentAlertsMu.Unlock()
				}
			}(alert)
		}
	} else {
		timeSinceLastSend := time.Since(lastSent)
		log.Printf("[Suricata] Alert sent recently (%.1f minutes ago), skipping: %s", timeSinceLastSend.Minutes(), alert.Signature)
	}

	// Create security event
	s.createEvent(alert)

	// Broadcast to WebSocket
	s.broadcastAlert(alert)
}

// createEvent creates a security event for the Suricata alert
func (s *SuricataManager) createEvent(alert SuricataAlert) {
	summary := fmt.Sprintf("Suricata Alert: %s from %s",
		alert.Signature, alert.SrcIP)

	raw := map[string]interface{}{
		"suricata_alert": true,
		"signature_id":   alert.SignatureID,
		"signature":      alert.Signature,
		"category":       alert.Category,
		"severity":       alert.Severity,
		"source_ip":      alert.SrcIP,
		"source_port":    alert.SrcPort,
		"dest_ip":        alert.DestIP,
		"dest_port":      alert.DestPort,
		"protocol":       alert.Proto,
	}

	// Add geolocation info if available
	if alert.GeoLocation != nil {
		raw["geo_country"] = alert.GeoLocation.CountryName
		raw["geo_city"] = alert.GeoLocation.City
		raw["geo_region"] = alert.GeoLocation.RegionName
		raw["geo_isp"] = alert.GeoLocation.ISP
		raw["geo_formatted"] = alert.GeoLocation.FormatLocation()
		raw["geo_is_risky"] = alert.GeoLocation.IsRiskyConnection()
	}

	// Map Suricata severity to our severity scale (1-100)
	severity := 50
	switch alert.Severity {
	case 1:
		severity = 30
	case 2:
		severity = 60
	case 3:
		severity = 80
	default:
		severity = 50
	}

	event := Event{
		ID:       fmt.Sprintf("evt-suricata-%d", alert.Timestamp.UnixNano()),
		TS:       alert.Timestamp.Format(time.RFC3339),
		DeviceID: nil, // External source
		Category: "suricata",
		Severity: severity,
		Summary:  summary,
		Raw:      raw,
	}

	err := s.eventDB.CreateEvent(event)
	if err != nil {
		log.Printf("Error creating Suricata event: %v", err)
	}
}

// broadcastAlert broadcasts the alert to WebSocket clients
func (s *SuricataManager) broadcastAlert(alert SuricataAlert) {
	broadcastData := map[string]interface{}{
		"timestamp":    alert.Timestamp.Format(time.RFC3339),
		"alert_id":     alert.AlertID,
		"signature":    alert.Signature,
		"category":     alert.Category,
		"severity":     alert.Severity,
		"source_ip":    alert.SrcIP,
		"source_port":  alert.SrcPort,
		"dest_ip":      alert.DestIP,
		"dest_port":    alert.DestPort,
		"protocol":     alert.Proto,
	}

	// Add geolocation if available
	if alert.GeoLocation != nil {
		broadcastData["geo_location"] = map[string]interface{}{
			"country":   alert.GeoLocation.CountryName,
			"city":      alert.GeoLocation.City,
			"region":    alert.GeoLocation.RegionName,
			"isp":       alert.GeoLocation.ISP,
			"is_risky":  alert.GeoLocation.IsRiskyConnection(),
			"formatted": alert.GeoLocation.FormatLocation(),
		}
	}

	s.hub.Broadcast("suricata_alert", broadcastData)
}

// collectStats periodically collects Suricata statistics and cleans up old entries
func (s *SuricataManager) collectStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Separate ticker for cleanup (every 5 minutes)
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateStats()
		case <-cleanupTicker.C:
			s.cleanupOldAlerts()
		}
	}
}

// cleanupOldAlerts removes old entries from sentAlerts to prevent memory bloat
// Removes entries older than the cooldown period
func (s *SuricataManager) cleanupOldAlerts() {
	s.sentAlertsMu.Lock()
	defer s.sentAlertsMu.Unlock()

	if len(s.sentAlerts) > 1000 {
		log.Printf("[Suricata] Cleaning up sentAlerts map (size: %d)", len(s.sentAlerts))
		cutoff := time.Now().Add(-s.alertCooldown)
		newMap := make(map[string]time.Time, len(s.sentAlerts)/2)

		for key, timestamp := range s.sentAlerts {
			// Keep only entries within cooldown period
			if timestamp.After(cutoff) {
				newMap[key] = timestamp
			}
		}

		s.sentAlerts = newMap
		log.Printf("[Suricata] Cleanup complete (new size: %d)", len(s.sentAlerts))
	}
}

// updateStats updates Suricata statistics from various sources
func (s *SuricataManager) updateStats() {
	// Update packet/byte counters if we have pcap access
	// This is a placeholder - real implementation would query Suricata stats
}

// GetRecentAlerts returns recent Suricata alerts
func (s *SuricataManager) GetRecentAlerts(limit int) []SuricataAlert {
	s.alertsMu.RLock()
	defer s.alertsMu.RUnlock()

	if limit > len(s.alerts) {
		limit = len(s.alerts)
	}

	if limit == 0 {
		return s.alerts
	}

	// Return most recent alerts
	start := len(s.alerts) - limit
	return s.alerts[start:]
}

// GetStats returns Suricata statistics
func (s *SuricataManager) GetStats() SuricataStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	return s.stats
}

// GetStatus returns the current status of Suricata integration
func (s *SuricataManager) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.GetStats()
	status := map[string]interface{}{
		"enabled":      s.enabled,
		"running":      s.running,
		"version":      s.version,
		"socket_path":  s.socketPath,
		"eve_log":      s.eveFilePath,
		"stats":        stats,
		"alerts_count": len(s.alerts),
	}

	// Check if EVE log file exists and is readable
	if _, err := os.Stat(s.eveFilePath); err == nil {
		status["eve_log_accessible"] = true
	} else {
		status["eve_log_accessible"] = false
		status["eve_log_error"] = err.Error()
	}

	return status
}

// Global geo lookup instance (will be set from main)
var geoLookup *GeoLookup

// SetGeoLookup sets the global geo lookup instance
func SetGeoLookup(geo *GeoLookup) {
	geoLookup = geo
}

// SetTelegramService sets the Telegram service for sending alerts
func (s *SuricataManager) SetTelegramService(telegram *TelegramService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.telegram = telegram
}
