package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// SensorManager manages all network sensors
type SensorManager struct {
	db       *sql.DB
	hub      *WSHub
	devDB    *DeviceDB
	eventDB  *EventDB
	sensors  map[string]EventSensor
	running  bool
	mu       sync.RWMutex
}

// NewSensorManager creates a new sensor manager
func NewSensorManager(db *sql.DB, hub *WSHub, devDB *DeviceDB, eventDB *EventDB) *SensorManager {
	return &SensorManager{
		db:      db,
		hub:     hub,
		devDB:   devDB,
		eventDB: eventDB,
		sensors: make(map[string]EventSensor),
		running: false,
	}
}

// Start starts all sensors
func (sm *SensorManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("sensors already running")
	}

	// Initialize sensors
	arpSensor := NewARPSensor(sm.devDB, sm.eventDB, sm.hub)
	sm.sensors["arp"] = arpSensor

	// Start each sensor
	for name, sensor := range sm.sensors {
		if err := sensor.Start(); err != nil {
			log.Printf("Failed to start sensor %s: %v", name, err)
			continue
		}
		log.Printf("Sensor %s started", name)
	}

	sm.running = true
	return nil
}

// Stop stops all sensors
func (sm *SensorManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return
	}

	for name, sensor := range sm.sensors {
		if err := sensor.Stop(); err != nil {
			log.Printf("Error stopping sensor %s: %v", name, err)
		}
	}

	sm.running = false
}

// IsSensorRunning checks if a specific sensor is running
func (sm *SensorManager) IsSensorRunning(name string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sensor, exists := sm.sensors[name]
	if !exists {
		return false
	}
	return sensor.IsHealthy()
}

// ARPSensor monitors ARP traffic for device discovery
type ARPSensor struct {
	devDB   *DeviceDB
	eventDB *EventDB
	hub     *WSHub
	quitCh  chan struct{}
	running bool
}

// NewARPSensor creates a new ARP sensor
func NewARPSensor(devDB *DeviceDB, eventDB *EventDB, hub *WSHub) *ARPSensor {
	return &ARPSensor{
		devDB:   devDB,
		eventDB: eventDB,
		hub:     hub,
		quitCh:  make(chan struct{}),
		running: false,
	}
}

// Start starts the ARP sensor
func (a *ARPSensor) Start() error {
	if a.running {
		return fmt.Errorf("ARP sensor already running")
	}

	a.running = true
	go a.monitor()

	return nil
}

// Stop stops the ARP sensor
func (a *ARPSensor) Stop() error {
	if !a.running {
		return nil
	}

	close(a.quitCh)
	a.running = false
	return nil
}

// IsHealthy returns the health status of the sensor
func (a *ARPSensor) IsHealthy() bool {
	return a.running
}

// monitor performs ARP monitoring
func (a *ARPSensor) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial scan
	a.scanNetwork()

	for {
		select {
		case <-ticker.C:
			a.scanNetwork()
		case <-a.quitCh:
			return
		}
	}
}

// scanNetwork scans the local network for devices
func (a *ARPSensor) scanNetwork() {
	// MVP: Simple subnet scan
	// v0.2: Use actual ARP scanning

	// Get local subnet
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Error getting interfaces: %v", err)
		return
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					a.scanSubnet(ipnet)
				}
			}
		}
	}
}

// scanSubnet scans a subnet for active hosts
func (a *ARPSensor) scanSubnet(ipnet *net.IPNet) {
	// Get the network size and determine scan strategy
	ones, bits := ipnet.Mask.Size()

	// For small networks (/<24 or smaller), do full parallel scan
	// For larger networks, sample to avoid excessive traffic
	if ones >= 24 || bits == 32 {
		// Full scan for /24 and smaller networks
		a.scanSubnetFull(ipnet)
	} else {
		// Sample scan for larger networks
		a.scanSubnetSample(ipnet)
	}
}

// scanSubnetFull performs a full parallel scan of the subnet
func (a *ARPSensor) scanSubnetFull(ipnet *net.IPNet) {
	// Get scan mode from environment
	scanMode := env("BG_SCAN_MODE", "fast")

	// Limit concurrent scans based on mode
	maxConcurrent := 50
	scanTimeout := 500 * time.Millisecond

	switch scanMode {
	case "thorough":
		maxConcurrent = 100
		scanTimeout = 2 * time.Second
	case "fast":
		maxConcurrent = 20
		scanTimeout = 200 * time.Millisecond
	case "minimal":
		maxConcurrent = 10
		scanTimeout = 100 * time.Millisecond
	}

	// Semaphore to limit concurrent scans
	sem := make(chan struct{}, maxConcurrent)

	// Iterate through all IPs in the subnet
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)

	for ipnet.Contains(ip) {
		// Skip network and broadcast addresses
		if !isUsableIP(ip, ipnet) {
			incIP(ip)
			continue
		}

		// Acquire semaphore
		sem <- struct{}{}
		go func(ipStr string) {
			defer func() { <-sem }()
			a.checkHostWithTimeout(ipStr, scanTimeout)
		}(ip.String())

		incIP(ip)
	}

	// Wait for all scans to complete
	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
}

// scanSubnetSample performs a sampled scan of larger subnets
func (a *ARPSensor) scanSubnetSample(ipnet *net.IPNet) {
	// Sample every nth IP to avoid excessive traffic
	sampleRate := 16 // Check 1 in 16 IPs

	// Get common network IPs to always check
	commonIPs := getGatewayAndDNSCandidates(ipnet)
	scanned := make(map[string]bool)

	// First, check common infrastructure IPs
	for _, ip := range commonIPs {
		if ipnet.Contains(net.ParseIP(ip)) {
			go a.checkHost(ip)
			scanned[ip] = true
		}
	}

	// Then sample the rest of the subnet
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)

	count := 0
	for ipnet.Contains(ip) {
		if !isUsableIP(ip, ipnet) {
			incIP(ip)
			continue
		}

		// Skip if already scanned
		ipStr := ip.String()
		if scanned[ipStr] {
			incIP(ip)
			continue
		}

		// Sample based on rate
		if count%sampleRate == 0 {
			go a.checkHost(ipStr)
		}

		count++
		incIP(ip)
	}
}

// checkHost checks if a host is alive
func (a *ARPSensor) checkHost(ipStr string) {
	// Try to connect to common ports
	ports := []int{80, 443, 22, 8080}

 alive := false
	for _, port := range ports {
		address := fmt.Sprintf("%s:%d", ipStr, port)
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			alive = true
			break
		}
	}

	if alive {
		a.recordDevice(ipStr)
	}
}

// recordDevice records a discovered device
func (a *ARPSensor) recordDevice(ipStr string) {
	// Check if device exists
	dev, err := a.devDB.GetDeviceByMAC(fmt.Sprintf("sim-%s", ipStr))
	if err != nil {
		return
	}

	if dev != nil {
		// Update last seen
		return
	}

	// Create new device
	now := time.Now().UTC().Format(time.RFC3339)
	newDevice := DeviceDetail{
		Device: Device{
			ID:        fmt.Sprintf("dev-%s", ipStr),
			MAC:       fmt.Sprintf("sim-%s", ipStr),
			IP:        ipStr,
			Hostname:  "Unknown",
			TypeGuess: "unknown",
			RiskScore: 0,
			LastSeen:  now,
		},
		FirstSeen: now,
		Tags:      []string{"discovered"},
	}

	if err := a.devDB.UpsertDevice(newDevice); err != nil {
		log.Printf("Error recording device: %v", err)
		return
	}

	// Notify via websocket
	a.hub.Broadcast("device_discovered", map[string]interface{}{
		"device": newDevice,
	})
}

// Subscribe subscribes to events from this sensor
func (a *ARPSensor) Subscribe(ch chan<- Event) error {
	// MVP: Implement in v0.2
	return nil
}

// checkHostWithTimeout checks if a host is alive with a timeout
func (a *ARPSensor) checkHostWithTimeout(ipStr string, timeout time.Duration) {
	// Try to connect to common ports with timeout
	ports := []int{80, 443, 22, 8080, 3389, 5900} // Add RDP and VNC for better detection

	alive := false
	for _, port := range ports {
		address := fmt.Sprintf("%s:%d", ipStr, port)
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			conn.Close()
			alive = true
			break
		}
	}

	if alive {
		a.recordDevice(ipStr)
	}
}

// isUsableIP checks if an IP is usable (not network/broadcast address)
func isUsableIP(ip net.IP, ipnet *net.IPNet) bool {
	// Convert to 4-byte representation for IPv4
	if ip4 := ip.To4(); ip4 != nil {
		// Get network and broadcast addresses
		network := ipnet.IP.To4()
		mask := ipnet.Mask

		// Calculate broadcast address
		broadcast := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			broadcast[i] = network[i] | (^mask[i])
		}

		// Check if IP is network or broadcast address
		return !ip.Equal(net.IP(network)) && !ip.Equal(broadcast)
	}
	return true
}

// incIP increments an IP address by 1
func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// getGatewayAndDNSCandidates returns likely gateway and DNS IPs
func getGatewayAndDNSCandidates(ipnet *net.IPNet) []string {
	ip := ipnet.IP.To4()
	if ip == nil {
		return []string{}
	}

	// Common gateway IPs (first .1 or .254 in subnet)
	candidates := []string{
		fmt.Sprintf("%d.%d.%d.1", ip[0], ip[1], ip[2]),
		fmt.Sprintf("%d.%d.%d.254", ip[0], ip[1], ip[2]),
		fmt.Sprintf("%d.%d.%d.253", ip[0], ip[1], ip[2]),
		// Common router IPs
		"192.168.1.1",
		"192.168.0.1",
		"10.0.0.1",
		"172.16.0.1",
	}

	return candidates
}
