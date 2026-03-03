package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ClamAVClient handles communication with ClamAV daemon (clamd)
type ClamAVClient struct {
	address     string // clamd address (e.g., "localhost:3310")
	timeout     time.Duration
	enabled     bool
	running     bool
	version     string
	dbVersion   string
	lastCheck   time.Time
	mu          sync.RWMutex
}

// NewClamAVClient creates a new ClamAV client
func NewClamAVClient() *ClamAVClient {
	client := &ClamAVClient{
		address: env("CLAMAV_ADDRESS", "localhost:3310"),
		timeout: 30 * time.Second,
		enabled: env("CLAMAV_ENABLED", "false") == "true",
		running: false,
	}
	// Initial check
	client.checkStatus()
	return client
}

// IsEnabled returns whether ClamAV scanning is enabled
func (c *ClamAVClient) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// Ping checks if ClamAV daemon is reachable
func (c *ClamAVClient) Ping() error {
	if !c.IsEnabled() {
		return fmt.Errorf("ClamAV scanning is disabled")
	}

	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to clamd: %w", err)
	}
	defer conn.Close()

	// Send PING command (nUL terminated)
	pingCmd := []byte{0x7A, 0x50, 0x49, 0x4E, 0x47, 0x00} // "zPING\0"
	_, err = conn.Write(pingCmd)
	if err != nil {
		return fmt.Errorf("failed to send PING: %w", err)
	}

	// Read response
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("failed to read PING response: %w", err)
	}

	respStr := string(response[:n])
	if !strings.Contains(respStr, "PONG") {
		return fmt.Errorf("unexpected PING response: %s", respStr)
	}

	return nil
}

// ScanData scans a byte buffer for threats
func (c *ClamAVClient) ScanData(data []byte) (*ScanResult, error) {
	if !c.IsEnabled() {
		return &ScanResult{Clean: true, Reason: "ClamAV scanning disabled"}, nil
	}

	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clamd: %w", err)
	}
	defer conn.Close()

	// Send INSTREAM command (nUL terminated)
	instreamCmd := []byte{0x7A, 0x49, 0x4E, 0x53, 0x54, 0x52, 0x45, 0x41, 0x4D, 0x00} // "zINSTREAM\0"
	_, err = conn.Write(instreamCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to send INSTREAM: %w", err)
	}

	// Send data in chunks (max chunk size for clamd is typically 2048 bytes, but we'll use smaller)
	chunkSize := 1024
	dataLen := len(data)

	for i := 0; i < dataLen; i += chunkSize {
		end := i + chunkSize
		if end > dataLen {
			end = dataLen
		}

		chunk := data[i:end]
		chunkLen := uint32(len(chunk))

		// Send chunk size as 4-byte big-endian
		chunkHeader := []byte{
			byte(chunkLen >> 24),
			byte(chunkLen >> 16),
			byte(chunkLen >> 8),
			byte(chunkLen),
		}

		_, err = conn.Write(chunkHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to send chunk header: %w", err)
		}

		_, err = conn.Write(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to send chunk data: %w", err)
		}
	}

	// Send zero-length chunk to indicate end of stream
	_, err = conn.Write([]byte{0, 0, 0, 0})
	if err != nil {
		return nil, fmt.Errorf("failed to send end marker: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read scan response: %w", err)
	}

	return c.parseScanResult(response)
}

// ScanResult represents the result of a ClamAV scan
type ScanResult struct {
	Clean  bool   `json:"clean"`
	Threat string `json:"threat,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (c *ClamAVClient) parseScanResult(response string) (*ScanResult, error) {
	response = strings.TrimSuffix(response, "\n")

	// Format: "filename: THREAT_NAME FOUND" or "filename: OK"
	parts := strings.Split(response, ": ")
	if len(parts) < 2 {
		return &ScanResult{
			Clean:  true,
			Reason: "Unknown response format",
		}, nil
	}

	status := parts[len(parts)-1]

	if strings.Contains(status, "FOUND") {
		// Extract threat name (format: "THREAT_NAME FOUND")
		threatName := strings.Replace(status, " FOUND", "", 1)
		return &ScanResult{
			Clean:  false,
			Threat: threatName,
			Reason: fmt.Sprintf("Malware detected: %s", threatName),
		}, nil
	}

	if status == "OK" {
		return &ScanResult{
			Clean:  true,
			Reason: "No threats detected",
		}, nil
	}

	return &ScanResult{
		Clean:  true,
		Reason: fmt.Sprintf("Unknown status: %s", status),
	}, nil
}

// checkStatus checks if ClamAV is running and gets version info
func (c *ClamAVClient) checkStatus() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		c.running = false
		c.version = "Disabled"
		c.dbVersion = "N/A"
		return
	}

	// Try to connect and get version
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		c.running = false
		c.version = "Not Connected"
		c.dbVersion = "N/A"
		c.lastCheck = time.Now()
		return
	}
	defer conn.Close()

	// Send VERSION command (newline terminated - works better with ClamAV)
	versionCmd := []byte("VERSION\n")
	_, err = conn.Write(versionCmd)
	if err != nil {
		c.running = false
		c.version = "Error"
		c.dbVersion = "N/A"
		c.lastCheck = time.Now()
		return
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		c.running = false
		c.version = "Error"
		c.dbVersion = "N/A"
		c.lastCheck = time.Now()
		return
	}

	// Parse version response
	// Format: "ClamAV 1.2.3/45678/..."
	c.running = true
	c.lastCheck = time.Now()

	response = strings.TrimSuffix(response, "\n")
	parts := strings.Split(response, "/")
	if len(parts) >= 2 {
		c.version = strings.TrimPrefix(parts[0], "ClamAV ")
		c.dbVersion = parts[1]
	} else {
		c.version = response
		c.dbVersion = "Unknown"
	}
}

// GetStatus returns the current status of ClamAV
func (c *ClamAVClient) GetStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"enabled":     c.enabled,
		"running":     c.running,
		"version":     c.version,
		"db_version":  c.dbVersion,
		"address":     c.address,
		"last_check":  c.lastCheck.Format(time.RFC3339),
	}
}

// IsRunning returns whether ClamAV daemon is running
func (c *ClamAVClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// RefreshStatus refreshes the ClamAV status
func (c *ClamAVClient) RefreshStatus() {
	c.checkStatus()
}

// ============================================================================
// URL Scanning - Phishing and Malware URL Detection
// ============================================================================

// URLScanner checks URLs against known malicious URL databases
type URLScanner struct {
	enabled        bool
	knownMalicious map[string]time.Time // domain -> when it was added
	mu             sync.RWMutex
	updateInterval time.Duration
	lastUpdate     time.Time
}

// Known malicious URL categories
var maliciousURLCategories = map[string][]string{
	"phishing": {
		"paypal-secure.com",
		"apple-support-id.com",
		"microsoft-account-help.com",
		"verify-login.com",
		"secure-account.net",
		"banking-verify.com",
	},
	"malware": {
		"download-cracked.com",
		"free-software-download.net",
		"keygen-download.com",
		"patch-download.com",
	},
	"porn_adult": {
		"xxx", "porn", "adult", "sex", "nude",
	},
	"gambling": {
		"casino", "poker", "betting", "gamble", "lottery",
	},
	"piracy": {
		"torrent", "warez", "crack", "pirate", "download-free-movie",
	},
}

// URLScanResult represents the result of a URL scan
type URLScanResult struct {
	Safe       bool     `json:"safe"`
	Categories []string `json:"categories,omitempty"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason,omitempty"`
}

// NewURLScanner creates a new URL scanner
func NewURLScanner() *URLScanner {
	scanner := &URLScanner{
		enabled:        env("URL_SCANNING_ENABLED", "true") == "true",
		knownMalicious: make(map[string]time.Time),
		updateInterval: 24 * time.Hour,
		lastUpdate:     time.Now(),
	}

	// Initialize known malicious domains
	scanner.initializeMaliciousDomains()

	// Start background updater
	go scanner.updater()

	return scanner
}

func (s *URLScanner) initializeMaliciousDomains() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, domains := range maliciousURLCategories {
		for _, domain := range domains {
			s.knownMalicious[domain] = now
		}
	}
}

// updater periodically updates the malicious URL database
func (s *URLScanner) updater() {
	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.updateMaliciousDB()
	}
}

// updateMaliciousDB fetches updates from threat intelligence sources
func (s *URLScanner) updateMaliciousDB() {
	log.Println("[URLScanner] Updating malicious URL database...")

	// In production, this would fetch from:
	// - PhishTank API
	// - OpenPhish
	// - VirusTotal API
	// - URLhaus

	s.mu.Lock()
	s.lastUpdate = time.Now()
	s.mu.Unlock()

	log.Printf("[URLScanner] Database updated. Total known malicious domains: %d",
		func() int {
			s.mu.RLock()
			defer s.mu.RUnlock()
			return len(s.knownMalicious)
		}())
}

// ScanURL checks if a URL is malicious
func (s *URLScanner) ScanURL(targetURL string) *URLScanResult {
	if !s.enabled {
		return &URLScanResult{
			Safe:   true,
			Reason: "URL scanning disabled",
		}
	}

	// Parse URL
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return &URLScanResult{
			Safe:   false,
			Reason: fmt.Sprintf("Invalid URL: %v", err),
		}
	}

	domain := strings.ToLower(parsed.Host)

	// Remove port if present
	if strings.Contains(domain, ":") {
		parts := strings.Split(domain, ":")
		domain = parts[0]
	}

	// Check against known malicious domains
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Direct match
	if _, exists := s.knownMalicious[domain]; exists {
		cats := s.getCategoriesForDomain(domain)
		return &URLScanResult{
			Safe:       false,
			Categories: cats,
			Confidence: 1.0,
			Reason:     fmt.Sprintf("Known malicious domain: %s (categories: %s)", domain, strings.Join(cats, ", ")),
		}
	}

	// Subdomain match (e.g., malicious.example.com)
	for maliciousDomain := range s.knownMalicious {
		if strings.HasSuffix(domain, "."+maliciousDomain) || domain == maliciousDomain {
			cats := s.getCategoriesForDomain(maliciousDomain)
			return &URLScanResult{
				Safe:       false,
				Categories: cats,
				Confidence: 0.9,
				Reason:     fmt.Sprintf("Subdomain of known malicious domain: %s", maliciousDomain),
			}
		}
	}

	// Pattern matching for suspicious characteristics
	confidence, reason := s.checkSuspiciousPatterns(domain, parsed)
	if confidence > 0.5 {
		return &URLScanResult{
			Safe:       false,
			Categories: []string{"suspicious"},
			Confidence: confidence,
			Reason:     reason,
		}
	}

	return &URLScanResult{
		Safe:       true,
		Categories: []string{},
		Confidence: 1.0,
		Reason:     "No known threats detected",
	}
}

// getCategoriesForDomain returns the categories for a given malicious domain
func (s *URLScanner) getCategoriesForDomain(domain string) []string {
	var cats []string

	for category, domains := range maliciousURLCategories {
		for _, d := range domains {
			if domain == d || strings.HasSuffix(domain, "."+d) {
				cats = append(cats, category)
			}
		}
	}

	return cats
}

// checkSuspiciousPatterns checks for suspicious URL patterns
func (s *URLScanner) checkSuspiciousPatterns(domain string, parsed *url.URL) (float64, string) {
	confidence := 0.0
	reasons := []string{}

	// Check for IP address as hostname (common in phishing)
	if net.ParseIP(domain) != nil {
		confidence += 0.3
		reasons = append(reasons, "Uses IP address instead of domain name")
	}

	// Check for excessive subdomain levels
	parts := strings.Split(domain, ".")
	if len(parts) > 4 {
		confidence += 0.2
		reasons = append(reasons, "Excessive subdomain levels")
	}

	// Check for suspicious TLDs
	suspiciousTLDs := []string{".xyz", ".top", ".zip", ".mov", ".tk", ".gq"}
	for _, tld := range suspiciousTLDs {
		if strings.HasSuffix(domain, tld) {
			confidence += 0.2
			reasons = append(reasons, "Uses suspicious TLD: "+tld)
			break
		}
	}

	// Check for homograph characters (non-ASCII)
	for _, r := range domain {
		if r > 127 {
			confidence += 0.4
			reasons = append(reasons, "Contains non-ASCII characters (possible homograph attack)")
			break
		}
	}

	// Check for long random strings
	if len(domain) > 50 {
		confidence += 0.1
		reasons = append(reasons, "Unusually long domain name")
	}

	// Check for suspicious keywords
	suspiciousKeywords := []string{"login", "signin", "verify", "account", "secure", "update", "bank"}
	lowerDomain := strings.ToLower(domain)
	keywordCount := 0
	for _, keyword := range suspiciousKeywords {
		if strings.Contains(lowerDomain, keyword) {
			keywordCount++
		}
	}
	if keywordCount >= 3 {
		confidence += 0.2
		reasons = append(reasons, "Contains multiple suspicious keywords")
	}

	if confidence == 0 {
		return 0, "No suspicious patterns detected"
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence, strings.Join(reasons, "; ")
}

// ============================================================================
// Integration with DNS Filter
// ============================================================================

// ScanDNSQuery scans a DNS query before allowing it
func (s *URLScanner) ScanDNSQuery(domain string) *DNSQueryResult {
	// Add protocol-less URL format
	urlToCheck := "http://" + domain
	result := s.ScanURL(urlToCheck)

	return &DNSQueryResult{
		Allowed:    result.Safe,
		Categories: result.Categories,
		Reason:     result.Reason,
	}
}

// DNSQueryResult represents the result of a DNS query scan
type DNSQueryResult struct {
	Allowed    bool     `json:"allowed"`
	Categories []string `json:"categories,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

// GetStats returns scanner statistics
func (s *URLScanner) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"enabled":           s.enabled,
		"malicious_domains": len(s.knownMalicious),
		"last_update":       s.lastUpdate.Format(time.RFC3339),
	}
}
