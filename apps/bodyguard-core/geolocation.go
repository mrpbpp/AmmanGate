package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// GeoLocation represents geographical information for an IP address
type GeoLocation struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	ZipCode     string  `json:"zip_code"`
	ISP         string  `json:"isp"`
	Organization string `json:"organization"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	IsVPN       bool    `json:"is_vpn"`
	IsProxy     bool    `json:"is_proxy"`
	IsTor       bool    `json:"is_tor"`
}

// geoCacheEntry represents a cached geolocation result
type geoCacheEntry struct {
	location GeoLocation
	timestamp time.Time
}

// GeoLookup provides IP geolocation services
type GeoLookup struct {
	cache     map[string]geoCacheEntry
	cacheMu   sync.RWMutex
	client    *http.Client
	cacheTTL  time.Duration
}

// NewGeoLookup creates a new geolocation service
func NewGeoLookup() *GeoLookup {
	return &GeoLookup{
		cache: make(map[string]geoCacheEntry),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		cacheTTL: 24 * time.Hour, // Cache for 24 hours
	}
}

// Lookup performs geolocation lookup for an IP address
func (g *GeoLookup) Lookup(ip string) (GeoLocation, error) {
	// Check cache first
	g.cacheMu.RLock()
	if entry, exists := g.cache[ip]; exists {
		if time.Since(entry.timestamp) < g.cacheTTL {
			g.cacheMu.RUnlock()
			return entry.location, nil
		}
	}
	g.cacheMu.RUnlock()

	// Skip local/private IPs
	if isPrivateIP(ip) || isLocalIP(ip) {
		location := GeoLocation{
			IP:          ip,
			CountryName: "Local/Private",
			City:        "Private Network",
		}
		g.cacheLocation(ip, location)
		return location, nil
	}

	// Perform lookup using free ip-api.com API
	location, err := g.lookupIPAPI(ip)
	if err != nil {
		// Fallback to ipinfo.io
		location, err = g.lookupIPInfo(ip)
		if err != nil {
			return GeoLocation{}, err
		}
	}

	// Cache the result
	g.cacheLocation(ip, location)

	return location, nil
}

// lookupIPAPI uses ip-api.com for geolocation
func (g *GeoLookup) lookupIPAPI(ip string) (GeoLocation, error) {
	// ip-api.com free API (no key needed for non-commercial use)
	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)

	resp, err := g.client.Get(url)
	if err != nil {
		return GeoLocation{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeoLocation{}, err
	}

	var result struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Region      string  `json:"region"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		Zip         string  `json:"zip"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
		Query       string  `json:"query"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return GeoLocation{}, err
	}

	if result.Status != "success" {
		return GeoLocation{}, fmt.Errorf("ip-api returned status: %s", result.Status)
	}

	return GeoLocation{
		IP:           result.Query,
		CountryCode:  result.CountryCode,
		CountryName:  result.Country,
		RegionCode:   result.Region,
		RegionName:   result.RegionName,
		City:         result.City,
		ZipCode:      result.Zip,
		ISP:          result.ISP,
		Organization: result.Org,
		Latitude:     result.Lat,
		Longitude:    result.Lon,
		IsVPN:        false,
		IsProxy:      false,
		IsTor:        false,
	}, nil
}

// lookupIPInfo uses ipinfo.io as fallback
func (g *GeoLookup) lookupIPInfo(ip string) (GeoLocation, error) {
	// ipinfo.io free API (limited requests)
	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)

	resp, err := g.client.Get(url)
	if err != nil {
		return GeoLocation{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeoLocation{}, err
	}

	var result struct {
		IP       string `json:"ip"`
		City     string `json:"city"`
		Region   string `json:"region"`
		Country  string `json:"country"`
		Loc      string `json:"loc"` // "lat,lon" format
		Org      string `json:"org"`
		Postal   string `json:"postal"`
		Timezone string `json:"timezone"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return GeoLocation{}, err
	}

	// Parse location
	var lat, lon float64
	fmt.Sscanf(result.Loc, "%f,%f", &lat, &lon)

	return GeoLocation{
		IP:           result.IP,
		CountryName:  result.Country,
		RegionName:   result.Region,
		City:         result.City,
		ZipCode:      result.Postal,
		ISP:          result.Org,
		Organization: result.Org,
		Latitude:     lat,
		Longitude:    lon,
		IsVPN:        false,
		IsProxy:      false,
		IsTor:        false,
	}, nil
}

// cacheLocation stores a location in the cache
func (g *GeoLookup) cacheLocation(ip string, location GeoLocation) {
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()
	g.cache[ip] = geoCacheEntry{
		location:  location,
		timestamp: time.Now(),
	}
}

// FormatLocation returns a human-readable location string
func (g *GeoLocation) FormatLocation() string {
	if g.CountryName == "Local/Private" {
		return "Private Network"
	}

	parts := []string{}
	if g.City != "" {
		parts = append(parts, g.City)
	}
	if g.RegionName != "" && g.RegionName != g.City {
		parts = append(parts, g.RegionName)
	}
	if g.CountryName != "" {
		parts = append(parts, g.CountryName)
	}

	if len(parts) == 0 {
		return "Unknown Location"
	}

	return fmt.Sprintf("%s", joinWithCommas(parts))
}

// FormatISP returns ISP information
func (g *GeoLocation) FormatISP() string {
	if g.ISP != "" {
		return g.ISP
	}
	if g.Organization != "" {
		return g.Organization
	}
	return "Unknown ISP"
}

// IsRiskyConnection returns true if the IP appears to be from VPN/Proxy/Tor
func (g *GeoLocation) IsRiskyConnection() bool {
	return g.IsVPN || g.IsProxy || g.IsTor
}

// isPrivateIP checks if an IP is in private ranges
func isPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.", "172.20.",
		"172.21.", "172.22.", "172.23.", "172.24.", "172.25.", "172.26.",
		"172.27.", "172.28.", "172.29.", "172.30.", "172.31.", "192.168.",
	}

	for _, prefix := range privateRanges {
		if len(ip) >= len(prefix) && ip[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// isLocalIP checks if an IP is localhost
func isLocalIP(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}

// joinWithCommas joins strings with commas
func joinWithCommas(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// GetCacheSize returns the current cache size (for monitoring)
func (g *GeoLookup) GetCacheSize() int {
	g.cacheMu.RLock()
	defer g.cacheMu.RUnlock()
	return len(g.cache)
}

// ClearCache clears the geolocation cache
func (g *GeoLookup) ClearCache() {
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()
	g.cache = make(map[string]geoCacheEntry)
}
