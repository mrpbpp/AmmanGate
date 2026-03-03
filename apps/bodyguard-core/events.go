package main

// Event Classification System for AmmanGate
//
// Severity Levels:
// 1-30: INFO - Normal network activities, no security concern
// 31-60: WARNING - Suspicious activities that should be monitored
// 61-100: CRITICAL - Active threats that require immediate attention

// EventTypes defines all event types and their default severity
var EventTypes = map[string]EventType{
	// === INFO EVENTS (1-30) ===
	"device_discovered": {
		Category: "device",
		Severity: 10,
		Title:    "New Device Discovered",
	},
	"device_online": {
		Category: "device",
		Severity: 5,
		Title:    "Device Came Online",
	},
	"device_offline": {
		Category: "device",
		Severity: 15,
		Title:    "Device Went Offline",
	},
	"dns_query_normal": {
		Category: "network",
		Severity: 1,
		Title:    "Normal DNS Query",
	},
	"dhcp_lease": {
		Category: "network",
		Severity: 5,
		Title:    "DHCP Lease Renewed",
	},
	"port_scan_benign": {
		Category: "network",
		Severity: 20,
		Title:    "Benign Port Scan (Local Network)",
	},

	// === WARNING EVENTS (31-60) ===
	"device_fingerprint_changed": {
		Category: "device",
		Severity: 40,
		Title:    "Device Fingerprint Changed",
	},
	"device_high_risk": {
		Category: "device",
		Severity: 50,
		Title:    "High Risk Device Detected",
	},
	"new_device_unknown_vendor": {
		Category: "device",
		Severity: 35,
		Title:    "Unknown Vendor Device Detected",
	},
	"dns_query_suspicious": {
		Category: "network",
		Severity: 45,
		Title:    "Suspicious DNS Query",
	},
	"port_scan_external": {
		Category: "network",
		Severity: 50,
		Title:    "External Port Scan Detected",
	},
	"unusual_traffic_pattern": {
		Category: "network",
		Severity: 40,
		Title:    "Unusual Traffic Pattern",
	},
	"connection_blocked": {
		Category: "network",
		Severity: 35,
		Title:    "Connection Blocked by Firewall",
	},
	"multiple_failed_auth": {
		Category: "auth",
		Severity: 45,
		Title:    "Multiple Failed Authentication Attempts",
	},
	"honeypot_hit_low": {
		Category: "honeypot",
		Severity: 50,
		Title:    "Honeypot Access (Low Risk)",
	},

	// === CRITICAL EVENTS (61-100) ===
	"honeypot_hit_smb": {
		Category: "honeypot",
		Severity: 70,
		Title:    "SMB Exploit Attempt Detected",
	},
	"honeypot_hit_rdp": {
		Category: "honeypot",
		Severity: 80,
		Title:    "RDP Brute Force Attack",
	},
	"honeypot_hit_sql": {
		Category: "honeypot",
		Severity: 75,
		Title:    "SQL Injection Attempt",
	},
	"honeypot_hit_exploit": {
		Category: "honeypot",
		Severity: 90,
		Title:    "Exploit Payload Detected",
	},
	"malware_detected": {
		Category: "threat",
		Severity: 95,
		Title:    "Malware Signature Detected",
	},
	"command_injection": {
		Category: "threat",
		Severity: 90,
		Title:    "Command Injection Attempt",
	},
	"data_exfiltration": {
		Category: "threat",
		Severity: 85,
		Title:    "Possible Data Exfiltration",
	},
	"device_compromised": {
		Category: "threat",
		Severity: 100,
		Title:    "Device Compromised",
	},
	"unauthorized_access": {
		Category: "auth",
		Severity: 90,
		Title:    "Unauthorized Access Attempt",
	},
	"ddos_attack": {
		Category: "threat",
		Severity: 95,
		Title:    "DDoS Attack Detected",
	},
	"mitm_attack": {
		Category: "threat",
		Severity: 85,
		Title:    "Man-in-the-Middle Attack Detected",
	},

	// === PARENTAL CONTROL EVENTS ===
	"dns_blocked_adult": {
		Category: "parental",
		Severity: 50,
		Title:    "Adult Content Blocked",
	},
	"dns_blocked_gambling": {
		Category: "parental",
		Severity: 50,
		Title:    "Gambling Site Blocked",
	},
	"dns_blocked_violence": {
		Category: "parental",
		Severity: 55,
		Title:    "Violent Content Blocked",
	},
	"dns_blocked_custom": {
		Category: "parental",
		Severity: 40,
		Title:    "Custom Filter Rule Triggered",
	},
	"parental_rule_violation": {
		Category: "parental",
		Severity: 60,
		Title:    "Parental Control Rule Violation",
	},
}

// EventType defines the metadata for an event type
type EventType struct {
	Category string // Category for grouping events
	Severity int    // Default severity level (1-100)
	Title    string // Human-readable title
}

// GetEventType returns the event type definition, or a default if not found
func GetEventType(eventType string) EventType {
	if def, exists := EventTypes[eventType]; exists {
		return def
	}
	// Default for unknown events
	return EventType{
		Category: "unknown",
		Severity: 50,
		Title:    "Unknown Event",
	}
}

// GetSeverityLevel returns the severity level as a string
func GetSeverityLevel(severity int) string {
	switch {
	case severity <= 30:
		return "info"
	case severity <= 60:
		return "warning"
	default:
		return "critical"
	}
}

// IsAlertworthy determines if an event should create an alert based on severity
func IsAlertworthy(severity int) bool {
	return severity >= 70
}

// EventSeverityOptions returns all severity levels for UI/display
func EventSeverityOptions() []string {
	return []string{"info (1-30)", "warning (31-60)", "critical (61-100)"}
}
