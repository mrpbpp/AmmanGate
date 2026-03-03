package main

import "time"

// SystemStatus represents the current system status
type SystemStatus struct {
	UptimeSec   int64           `json:"uptime_sec"`
	CpuLoad     float64         `json:"cpu_load"`
	MemUsedMB   int64           `json:"mem_used_mb"`
	Sensors     map[string]bool `json:"sensors"`
	LastEventTS string          `json:"last_event_ts"`
	ClamAV      map[string]interface{} `json:"clamav,omitempty"`
}

// Device represents a network device
type Device struct {
	ID        string `json:"id"`
	MAC       string `json:"mac"`
	IP        string `json:"ip"`
	Hostname  string `json:"hostname"`
	Vendor    string `json:"vendor"`
	TypeGuess string `json:"type_guess"`
	RiskScore int    `json:"risk_score"`
	LastSeen  string `json:"last_seen"`
}

// DeviceDetail represents detailed device information
type DeviceDetail struct {
	Device
	FirstSeen    string            `json:"first_seen"`
	Tags         []string          `json:"tags"`
	Notes        string            `json:"notes"`
	Fingerprint  *DeviceFingerprint `json:"fingerprint,omitempty"`
	ActivityStats *DeviceActivity  `json:"activity_stats,omitempty"`
}

// DeviceActivity represents device activity statistics
type DeviceActivity struct {
	TotalEvents    int    `json:"total_events"`
	AlertsCount    int    `json:"alerts_count"`
	LastActivity   string `json:"last_activity"`
	FirstSeen      string `json:"first_seen"`
	ConnectionCount int   `json:"connection_count"`
}

// Event represents a security event
type Event struct {
	ID       string                 `json:"id"`
	TS       string                 `json:"ts"`
	DeviceID *string                `json:"device_id"`
	Category string                 `json:"category"`
	Severity int                    `json:"severity"`
	Summary  string                 `json:"summary"`
	Raw      map[string]interface{} `json:"raw"`
}

// Alert represents a security alert
type Alert struct {
	ID              string   `json:"id"`
	TS              string   `json:"ts"`
	DeviceID        *string  `json:"device_id"`
	Severity        int      `json:"severity"`
	Title           string   `json:"title"`
	Status          string   `json:"status"`
	RelatedEventIDs []string `json:"related_event_ids"`
}

// ActionRequest represents a request for action approval
type ActionRequest struct {
	ActionType  string                 `json:"action_type"`
	Target      map[string]interface{} `json:"target"`
	TTLsec      int                    `json:"ttl_sec"`
	RequestedBy string                 `json:"requested_by"`
}

// ApprovalChallenge represents the approval challenge response
type ApprovalChallenge struct {
	ActionID   string `json:"action_id"`
	ApprovalID string `json:"approval_id"`
	ExpiresAt  string `json:"expires_at"`
	Message    string `json:"message"`
}

// ActionResult represents the result of an approved action
type ActionResult struct {
	ActionID string `json:"action_id"`
	Status   string `json:"status"`
	Detail   string `json:"detail"`
}

// WSMessage represents a websocket message
type WSMessage struct {
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data,omitempty"`
	TS      string                 `json:"ts,omitempty"`
}

// SensorStatus represents the status of a sensor
type SensorStatus struct {
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
}

// DeviceSensor is the interface for device detection sensors
type DeviceSensor interface {
	Start() error
	Stop() error
	GetDevices() ([]Device, error)
	IsHealthy() bool
}

// FilterRule represents a content filtering rule
type FilterRule struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Pattern   string `json:"pattern"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

// DeviceProfile represents parental control settings for a device
type DeviceProfile struct {
	DeviceID    string `json:"device_id"`
	FilterLevel string `json:"filter_level"` // off, light, moderate, strict
	Schedule    string `json:"schedule"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// DNSQueryLog represents a DNS query log entry
type DNSQueryLog struct {
	ID        string  `json:"id"`
	Timestamp string  `json:"ts"`
	DeviceID  string  `json:"device_id"`
	Domain    string  `json:"domain"`
	Blocked   bool    `json:"blocked"`
	RuleID    string  `json:"rule_id"`
}

// EventSensor is the interface for event detection sensors
type EventSensor interface {
	Start() error
	Stop() error
	Subscribe(chan<- Event) error
	IsHealthy() bool
}
