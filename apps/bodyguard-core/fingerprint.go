package main

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DeviceFingerprint contains detailed fingerprint information
type DeviceFingerprint struct {
	MACVendor       string           `json:"mac_vendor"`
	OSGuess         string           `json:"os_guess"`
	OSConfidence    float64          `json:"os_confidence"`
	OpenPorts       []int            `json:"open_ports"`
	Services        map[int]string   `json:"services"`
	HTTPFingerprint *HTTPFingerprint `json:"http_fingerprint,omitempty"`
	LastScan        string           `json:"last_scan"`
	ScanStatus      string           `json:"scan_status"`
}

// HTTPFingerprint contains HTTP service fingerprinting results
type HTTPFingerprint struct {
	Server          string   `json:"server"`
	Technologies    []string `json:"technologies"`
	Title           string   `json:"title"`
	StatusCode      int      `json:"status_code"`
	HasAuth         bool     `json:"has_auth"`
	IsHTTPS         bool     `json:"is_https"`
	SecurityHeaders []string `json:"security_headers"`
}

// Fingerprinter handles device fingerprinting
type Fingerprinter struct {
	client      *http.Client
	macDatabase map[string]string
	mu          sync.RWMutex
}

// NewFingerprinter creates a new device fingerprinter
func NewFingerprinter() *Fingerprinter {
	f := &Fingerprinter{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		macDatabase: make(map[string]string),
	}
	f.loadMACDatabase()
	return f
}

// loadMACDatabase loads MAC address vendor prefixes (carefully organized, no duplicates)
func (f *Fingerprinter) loadMACDatabase() {
	commonVendors := map[string]string{
		// Apple
		"00:03:93": "Apple", "00:0A:95": "Apple", "00:0E:35": "Apple",
		"00:11:24": "Apple", "00:14:51": "Apple", "00:15:C5": "Apple",
		"00:17:F2": "Apple", "00:1A:11": "Apple", "00:1B:63": "Apple",
		"00:1E:52": "Apple", "00:1E:C2": "Apple", "00:23:32": "Apple",
		"00:23:DF": "Apple", "00:25:00": "Apple", "00:25:4B": "Apple",
		"00:25:BC": "Apple", "00:26:B0": "Apple", "28:CF:E9": "Apple",
		"3C:15:C2": "Apple", "54:83:3A": "Apple", "58:55:CA": "Apple",
		"60:03:08": "Apple", "64:20:3C": "Apple", "70:73:CB": "Apple",
		"7C:C3:A1": "Apple", "84:85:93": "Apple", "88:C6:26": "Apple",
		"8C:85:90": "Apple", "98:01:A7": "Apple", "A4:D1:D2": "Apple",
		"A8:88:CC": "Apple", "B0:A4:DA": "Apple",

		// Samsung
		"00:12:FB": "Samsung", "00:15:B9": "Samsung", "00:16:6C": "Samsung",
		"00:1D:72": "Samsung", "00:22:F7": "Samsung", "00:23:2D": "Samsung",
		"00:24:54": "Samsung", "00:E0:FC": "Samsung", "08:21:EF": "Samsung",
		"14:DD:A9": "Samsung", "20:F3:9C": "Samsung", "28:C2:DD": "Samsung",
		"3C:A8:2A": "Samsung", "58:44:98": "Samsung",
		"88:25:93": "Samsung", "A0:8C:71": "Samsung", "A4:C3:F0": "Samsung",
		"B4:9E:44": "Samsung", "C0:EE:FB": "Samsung",
		"C8:7B:2A": "Samsung", "CC:3A:61": "Samsung", "D0:03:4B": "Samsung",
		"D4:3F:1D": "Samsung", "D4:9E:25": "Samsung", "E0:DB:55": "Samsung",
		"E4:70:B8": "Samsung", "E8:8D:28": "Samsung", "EC:1F:72": "Samsung",
		"F0:27:2D": "Samsung", "F4:4E:30": "Samsung",

		// SpaceX Starlink
		"74:24:9F": "SpaceX Starlink",

		// Google/Nest
		"A4:77:58": "Google", "B4:Fb:77": "Google",
		"C8:BE:FC": "Google", "E4:BF:01": "Google", "F4:F5:DB": "Google",
		"6A:64:51": "Google",

		// One Plus
		"E0:DC:FF": "OnePlus", "F0:EE:30": "OnePlus",

		// Oppo
		"30:FC:B9": "Oppo", "80:89:0C": "Oppo", "AC:19:28": "Oppo",
		"E0:63:DA": "Oppo", "F2:CB:E1": "Oppo",

		// Vivo
		"2C:F7:33": "Vivo", "44:03:2C": "Vivo", "60:AB:D2": "Vivo",
		"AC:5D:81": "Vivo",

		// Realme
		"10:2E:78": "Realme", "3C:63:95": "Realme", "82:2C:1F": "Realme",
		"D8:99:F0": "Realme", "F6:06:79": "Realme",

		// Motorola
		"00:0C:BF": "Motorola", "00:1A:9E": "Motorola", "00:1E:6A": "Motorola",
		"00:21:E9": "Motorola", "00:26:82": "Motorola",

		// Lenovo
		"6C:FD:31": "Lenovo", "88:43:DF": "Lenovo",
		"AC:5F:3E": "Lenovo", "F4:75:89": "Lenovo",

		// Sony
		"00:02:6F": "Sony", "00:12:EE": "Sony",
		"00:18:0B": "Sony", "00:1B:D9": "Sony",
		"00:21:99": "Sony", "00:26:B4": "Sony",
		"08:1C:DA": "Sony",
		"20:64:32": "Sony",
		"40:F3:08": "Sony", "64:B4:9B": "Sony",

		// LG
		"00:0D:B9": "LG", "00:0E:E4": "LG",
		"38:AA:3C": "LG", "AC:DF:89": "LG",
		"B0:39:E6": "LG",
		"D0:AE:5C": "LG", "F0:2F:A9": "LG",

		// Nintendo
		"E0:95:61": "Nintendo", "FC:EE:37": "Nintendo",

		// Microsoft/Xbox
		"BC:D1:D3": "Microsoft", "E8:EB:E3": "Microsoft",

		// TP-Link
		"00:1B:A5": "TP-Link", "00:1E:73": "TP-Link", "00:25:86": "TP-Link",
		"00:27:1D": "TP-Link", "04:BD:59": "TP-Link", "50:C7:BF": "TP-Link",
		"64:70:02": "TP-Link", "78:44:FD": "TP-Link", "84:A8:E4": "TP-Link",
		"98:48:27": "TP-Link", "A0:15:70": "TP-Link", "A0:F3:C1": "TP-Link",
		"A4:2B:B0": "TP-Link", "B4:75:0E": "TP-Link", "C0:4A:00": "TP-Link",
		"C8:3A:35": "TP-Link", "D0:37:B3": "TP-Link", "E0:70:60": "TP-Link",
		"E8:60:DF": "TP-Link", "F4:EC:38": "TP-Link", "F8:8D:6D": "TP-Link",

		// Dell
		"00:00:0E": "Dell", "00:01:E8": "Dell", "00:0C:29": "Dell",
		"00:14:22": "Dell", "00:16:76": "Dell", "00:18:8B": "Dell",
		"00:19:B9": "Dell", "00:1B:21": "Dell", "00:1C:CD": "Dell",
		"00:1E:C9": "Dell", "00:21:5A": "Dell", "00:21:70": "Dell",
		"00:21:9B": "Dell", "00:23:AE": "Dell", "00:26:18": "Dell",
		"08:00:27": "Dell", "08:94:EF": "Dell", "14:FE:B5": "Dell",
		"18:03:73": "Dell", "1C:6F:65": "Dell", "24:B6:FD": "Dell",
		"28:92:4A": "Dell", "2C:41:38": "Dell", "30:9C:23": "Dell",
		"34:17:E9": "Dell", "40:B3:95": "Dell",
		"44:8A:5B": "Dell", "48:DF:37": "Dell", "4C:4C:02": "Dell",
		"50:46:5D": "Dell", "58:8D:09": "Dell", "6C:F0:49": "Dell",
		"78:2B:CB": "Dell", "84:2B:2B": "Dell", "88:51:FB": "Dell",
		"8C:AE:4C": "Dell", "A0:CE:C8": "Dell",

		// HP
		"00:00:83": "HP", "00:01:E6": "HP", "00:04:75": "HP",
		"00:08:02": "HP", "00:0B:CD": "HP", "00:0E:7F": "HP",
		"00:11:85": "HP", "00:13:A9": "HP", "00:14:C2": "HP",
		"00:15:60": "HP", "00:15:F5": "HP", "00:18:71": "HP",
		"00:19:BB": "HP", "00:1A:A0": "HP", "00:1B:78": "HP",
		"00:1E:8F": "HP", "00:22:64": "HP", "00:23:7D": "HP",
		"00:25:B3": "HP", "00:26:9E": "HP", "78:2B:46": "HP",
		"BC:5F:F4": "HP", "D4:AE:52": "HP", "F0:1D:2F": "HP",

		// Huawei
		"00:1E:EC": "Huawei", "00:25:9E": "Huawei", "08:10:74": "Huawei",
		"18:E8:29": "Huawei", "34:12:78": "Huawei", "40:16:9E": "Huawei",
		"44:D1:FA": "Huawei", "78:1D:BA": "Huawei", "80:1F:02": "Huawei",
		"A0:96:30": "Huawei", "C0:8A:95": "Huawei", "D4:28:BF": "Huawei",
		"D8:50:E6": "Huawei", "E0:24:7F": "Huawei", "E8:C2:EA": "Huawei",
		"F0:59:26": "Huawei",

		// Xiaomi
		"34:CE:00": "Xiaomi", "38:BC:1A": "Xiaomi", "44:DA:E7": "Xiaomi",
		"58:AF:35": "Xiaomi", "60:AB:67": "Xiaomi", "64:09:80": "Xiaomi",
		"7C:70:BC": "Xiaomi", "84:16:F9": "Xiaomi", "98:FA:E3": "Xiaomi",
		"A0:21:37": "Xiaomi", "A4:67:06": "Xiaomi", "A8:15:74": "Xiaomi",
		"B0:61:AA": "Xiaomi", "B4:7B:2D": "Xiaomi", "C8:57:C2": "Xiaomi",
		"D4:6B:36": "Xiaomi", "D8:15:0D": "Xiaomi", "F4:8C:EB": "Xiaomi",
		"FC:64:BA": "Xiaomi",

		// Realtek
		"00:00:D8": "Realtek", "00:E0:4C": "Realtek", "00:E0:6C": "Realtek",
		"88:AE:DD": "Realtek", "C0:61:63": "Realtek",

		// Google
		"F0:F8:74": "Google", "3C:D9:2B": "Google",
		"44:D9:E7": "Google", "78:4F:43": "Google",

		// Microsoft
		"00:50:56": "Microsoft", "00:15:5D": "Microsoft",

		// Asus
		"00:1B:FC": "Asus", "04:92:26": "Asus", "08:10:77": "Asus",
		"24:05:0F": "Asus", "38:1C:9A": "Asus", "44:07:0B": "Asus",
		"60:A4:4C": "Asus", "78:24:AF": "Asus", "AC:22:05": "Asus",
		"B4:2E:99": "Asus", // Note: Also Realtek OUI, but listing as Asus here


		// Cisco
		"00:00:0C": "Cisco", "00:01:42": "Cisco", "00:0B:BE": "Cisco",
		"00:0E:B5": "Cisco", "00:0F:24": "Cisco", "00:12:7F": "Cisco",
		"00:14:BF": "Cisco", "00:15:63": "Cisco", "00:15:C6": "Cisco",
		"00:16:C8": "Cisco", "00:17:94": "Cisco", "00:18:74": "Cisco",
		"00:19:55": "Cisco", "00:1B:D5": "Cisco", "00:1C:C4": "Cisco",
		"00:1D:AA": "Cisco", "00:1E:14": "Cisco", "00:1E:A7": "Cisco",
		"00:21:B7": "Cisco", "00:22:90": "Cisco", "00:23:AB": "Cisco",
		"00:24:D6": "Cisco", "00:26:72": "Cisco", "00:27:0E": "Cisco",
		"F0:29:29": "Cisco",

		// Sony
		"00:02:A7": "Sony", "00:04:76": "Sony", "00:05:94": "Sony",
		"00:0E:6C": "Sony", "00:12:AA": "Sony", "00:1E:E6": "Sony",
		"00:21:E8": "Sony", "00:23:45": "Sony", "00:24:7C": "Sony",
		"28:C7:9D": "Sony",

		// LG
		"00:0B:52": "LG", "00:0E:A6": "LG", "00:1F:96": "LG",
		"00:23:63": "LG", "00:24:E9": "LG",
		"00:26:CB": "LG",

		// Intel
		"A0:36:9F": "Intel",

		// Roku
		"00:0D:CA": "Roku", "B0:4E:26": "Roku", "C4:43:1D": "Roku",

		// Amazon
		"00:25:9C": "Amazon", "44:65:DA": "Amazon", "6C:E5:1D": "Amazon",
		"84:D6:D0": "Amazon", "FC:A1:3E": "Amazon",

		// Netgear
		"00:09:70": "Netgear", "00:0F:B5": "Netgear", "00:14:6C": "Netgear",
		"00:1E:90": "Netgear", "00:22:3F": "Netgear", "00:24:B2": "Netgear",
		"00:26:F2": "Netgear", "30:46:9A": "Netgear", "34:40:72": "Netgear",
		"40:F4:EC": "Netgear", "50:7E:5D": "Netgear", "84:8B:FD": "Netgear",
		"A4:91:1F": "Netgear", "AC:1B:44": "Netgear",

		// AzureWave Technology (Taiwan) - wireless chips
		"48:E7:DA": "AzureWave Technology", "84:4B:A5": "AzureWave",

		// Randomized MAC prefixes (iOS, Android privacy features)
		"C6:AE:5F": "Randomized MAC", "62:6B:04": "Randomized MAC",
		"9A:0E:8C": "Randomized MAC", "E6:FA:80": "Randomized MAC",
		"D2:9F:B9": "Randomized MAC", "A6:5E:BD": "Randomized MAC",
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	for oui, vendor := range commonVendors {
		f.macDatabase[oui] = vendor
	}
}

// GetMACVendor looks up the vendor for a MAC address
func (f *Fingerprinter) GetMACVendor(mac string) string {
	if mac == "" {
		return "Unknown"
	}

	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	parts := strings.Split(mac, ":")

	if len(parts) < 3 {
		return "Unknown"
	}

	oui := fmt.Sprintf("%s:%s:%s", parts[0], parts[1], parts[2])

	f.mu.RLock()
	defer f.mu.RUnlock()

	if vendor, exists := f.macDatabase[oui]; exists {
		return vendor
	}

	// Try 2-character prefix for broader match
	prefix2 := fmt.Sprintf("%s:%s", parts[0], parts[1])
	for ouiKey, vendor := range f.macDatabase {
		if strings.HasPrefix(ouiKey, prefix2) {
			return vendor
		}
	}

	return "Unknown Vendor"
}

// ScanPorts scans common ports on the target IP
func (f *Fingerprinter) ScanPorts(ip string) []int {
	commonPorts := []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 3389, 8080}

	openPorts := []int{} // Initialize as empty slice instead of nil
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, port := range commonPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			address := fmt.Sprintf("%s:%d", ip, p)
			conn, err := net.DialTimeout("tcp", address, 1*time.Second)
			if err == nil {
				conn.Close()
				mu.Lock()
				openPorts = append(openPorts, p)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()
	return openPorts
}

// GetServiceName returns the common service name for a port
func (f *Fingerprinter) GetServiceName(port int) string {
	services := map[int]string{
		21: "FTP", 22: "SSH", 23: "Telnet", 25: "SMTP",
		53: "DNS", 80: "HTTP", 110: "POP3", 143: "IMAP",
		443: "HTTPS", 445: "SMB", 3389: "RDP", 8080: "HTTP-Alt",
	}

	if service, exists := services[port]; exists {
		return service
	}
	return fmt.Sprintf("Port-%d", port)
}

// HTTPFingerprint attempts to fingerprint HTTP services
func (f *Fingerprinter) HTTPFingerprint(ip string, useHTTPS bool) *HTTPFingerprint {
	protocol := "http"
	if useHTTPS {
		protocol = "https"
	}

	url := fmt.Sprintf("%s://%s/", protocol, ip)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	fingerprint := &HTTPFingerprint{
		StatusCode: resp.StatusCode,
		IsHTTPS:     useHTTPS,
	}

	if server := resp.Header.Get("Server"); server != "" {
		fingerprint.Server = server
	}

	securityHeaders := []string{
		"X-Frame-Options", "X-Content-Type-Options", "X-XSS-Protection",
		"Strict-Transport-Security", "Content-Security-Policy",
	}

	for _, header := range securityHeaders {
		if resp.Header.Get(header) != "" {
			fingerprint.SecurityHeaders = append(fingerprint.SecurityHeaders, header)
		}
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		fingerprint.HasAuth = true
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	titleRe := regexp.MustCompile(`<title>(.*?)</title>`)
	if match := titleRe.FindStringSubmatch(bodyStr); len(match) > 1 {
		fingerprint.Title = strings.TrimSpace(match[1])
	}

	techDetect := map[string]string{
		"wordpress": "WordPress", "drupal": "Drupal", "joomla": "Joomla",
		"bootstrap": "Bootstrap", "jquery": "jQuery", "react": "React",
		"vue": "Vue.js", "angular": "Angular", "laravel": "Laravel",
		"express": "Express", "django": "Django", "flask": "Flask",
		"asp\\.net": "ASP.NET", "php": "PHP", "nginx": "Nginx",
		"apache": "Apache", "iis": "IIS", "tomcat": "Tomcat",
	}

	bodyLower := strings.ToLower(bodyStr)
	for pattern, tech := range techDetect {
		if matched, _ := regexp.MatchString(pattern, bodyLower); matched {
			fingerprint.Technologies = append(fingerprint.Technologies, tech)
		}
	}

	return fingerprint
}

// GuessOS attempts to guess the operating system based on network behavior
func (f *Fingerprinter) GuessOS(macVendor string, openPorts []int) (string, float64) {
	scores := map[string]float64{
		"Windows": 0, "Linux": 0, "macOS": 0,
		"Android": 0, "iOS": 0, "Router": 0, "IoT": 0, "Unknown": 0,
	}

	for _, port := range openPorts {
		switch port {
		case 23:
			scores["Router"] += 30
			scores["IoT"] += 20
		case 53:
			scores["Router"] += 20
		case 80, 8080:
			scores["Router"] += 10
			scores["IoT"] += 10
		case 443:
			scores["Router"] += 10
		case 445:
			scores["Windows"] += 40
		case 3389:
			scores["Windows"] += 50
		case 22:
			scores["Linux"] += 20
			scores["macOS"] += 20
			scores["Router"] += 15
		}
	}

	vendorLower := strings.ToLower(macVendor)
	if strings.Contains(vendorLower, "apple") {
		scores["macOS"] += 50
		scores["iOS"] += 30
	}
	if strings.Contains(vendorLower, "samsung") ||
		strings.Contains(vendorLower, "xiaomi") ||
		strings.Contains(vendorLower, "huawei") {
		scores["Android"] += 40
	}
	if strings.Contains(vendorLower, "tp-link") ||
		strings.Contains(vendorLower, "dlink") ||
		strings.Contains(vendorLower, "netgear") ||
		strings.Contains(vendorLower, "linksys") {
		scores["Router"] += 60
	}

	maxScore := 0.0
	osGuess := "Unknown"
	for os, score := range scores {
		if score > maxScore {
			maxScore = score
			osGuess = os
		}
	}

	confidence := (maxScore / 100.0) * 100
	if confidence > 95 {
		confidence = 95
	}

	return osGuess, confidence
}

// FingerprintDevice performs comprehensive fingerprinting on a device
func (f *Fingerprinter) FingerprintDevice(ip, mac string) DeviceFingerprint {
	result := DeviceFingerprint{
		Services:  make(map[int]string),
		LastScan:  time.Now().UTC().Format(time.RFC3339),
		ScanStatus: "complete",
	}

	result.MACVendor = f.GetMACVendor(mac)

	openPorts := f.ScanPorts(ip)
	result.OpenPorts = openPorts

	for _, port := range openPorts {
		result.Services[port] = f.GetServiceName(port)
	}

	if containsInt(openPorts, 80) {
		result.HTTPFingerprint = f.HTTPFingerprint(ip, false)
	}
	if containsInt(openPorts, 443) {
		httpsFingerprint := f.HTTPFingerprint(ip, true)
		if httpsFingerprint != nil {
			if result.HTTPFingerprint == nil {
				result.HTTPFingerprint = httpsFingerprint
			} else {
				if httpsFingerprint.Server != "" {
					result.HTTPFingerprint.Server = httpsFingerprint.Server
				}
				if httpsFingerprint.StatusCode != 0 {
					result.HTTPFingerprint.StatusCode = httpsFingerprint.StatusCode
				}
			}
		}
	}

	result.OSGuess, result.OSConfidence = f.GuessOS(result.MACVendor, openPorts)

	return result
}

func containsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
