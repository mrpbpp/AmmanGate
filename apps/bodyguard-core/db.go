package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// openDB opens or creates the SQLite database
func openDB(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Open database with connection pooling settings
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// applyMigrations applies all SQL migrations from the migrations directory
func applyMigrations(db *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		b, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", e.Name(), err)
		}

		if _, err := db.Exec(string(b)); err != nil {
			return fmt.Errorf("migration %s failed: %w", e.Name(), err)
		}
	}

	return nil
}

// DeviceDB handles device database operations
type DeviceDB struct {
	db *sql.DB
}

// NewDeviceDB creates a new device database handler
func NewDeviceDB(db *sql.DB) *DeviceDB {
	return &DeviceDB{db: db}
}

// UpsertDevice inserts or updates a device
func (d *DeviceDB) UpsertDevice(device DeviceDetail) error {
	tagsJSON := "[]"
	if len(device.Tags) > 0 {
		// Simple JSON encoding for tags
		tagsJSON = "[\"" + strings.Join(device.Tags, "\",\"") + "\"]"
	}

	_, err := d.db.Exec(`
		INSERT INTO devices (id, mac, ip, hostname, vendor, type_guess, risk_score, first_seen, last_seen, tags, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			ip = excluded.ip,
			hostname = excluded.hostname,
			vendor = excluded.vendor,
			type_guess = excluded.type_guess,
			risk_score = excluded.risk_score,
			last_seen = excluded.last_seen,
			tags = excluded.tags,
			notes = excluded.notes
	`, device.ID, device.MAC, device.IP, device.Hostname, device.Vendor,
		device.TypeGuess, device.RiskScore, device.FirstSeen, device.LastSeen,
		tagsJSON, device.Notes)

	return err
}

// GetDeviceByMAC retrieves a device by MAC address
func (d *DeviceDB) GetDeviceByMAC(mac string) (*DeviceDetail, error) {
	var dev DeviceDetail
	var tagsJSON string

	err := d.db.QueryRow(`
		SELECT id, mac, COALESCE(ip,''), COALESCE(hostname,''), COALESCE(vendor,''),
		       COALESCE(type_guess,''), risk_score, first_seen, last_seen,
		       tags, COALESCE(notes,'')
		FROM devices WHERE mac = ?
	`, mac).Scan(&dev.ID, &dev.MAC, &dev.IP, &dev.Hostname, &dev.Vendor,
		&dev.TypeGuess, &dev.RiskScore, &dev.FirstSeen, &dev.LastSeen,
		&tagsJSON, &dev.Notes)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse tags JSON (simple implementation)
	dev.Tags = []string{}
	if tagsJSON != "[]" && tagsJSON != "" {
		tagsStr := strings.TrimSuffix(strings.TrimPrefix(tagsJSON, "[\""), "\"]")
		if tagsStr != "" {
			dev.Tags = strings.Split(tagsStr, "\",\"")
		}
	}

	return &dev, nil
}

// GetDeviceByID retrieves a device by ID
func (d *DeviceDB) GetDeviceByID(id string) (*DeviceDetail, error) {
	var dev DeviceDetail
	var tagsJSON string

	err := d.db.QueryRow(`
		SELECT id, mac, COALESCE(ip,''), COALESCE(hostname,''), COALESCE(vendor,''),
		       COALESCE(type_guess,''), risk_score, first_seen, last_seen,
		       tags, COALESCE(notes,'')
		FROM devices WHERE id = ?
	`, id).Scan(&dev.ID, &dev.MAC, &dev.IP, &dev.Hostname, &dev.Vendor,
		&dev.TypeGuess, &dev.RiskScore, &dev.FirstSeen, &dev.LastSeen,
		&tagsJSON, &dev.Notes)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse tags JSON
	dev.Tags = []string{}
	if tagsJSON != "[]" && tagsJSON != "" {
		tagsStr := strings.TrimSuffix(strings.TrimPrefix(tagsJSON, "[\""), "\"]")
		if tagsStr != "" {
			dev.Tags = strings.Split(tagsStr, "\",\"")
		}
	}

	return &dev, nil
}

// GetDeviceByIP retrieves a device by IP address
func (d *DeviceDB) GetDeviceByIP(ip string) (*DeviceDetail, error) {
	var dev DeviceDetail
	var tagsJSON string

	err := d.db.QueryRow(`
		SELECT id, mac, COALESCE(ip,''), COALESCE(hostname,''), COALESCE(vendor,''),
		       COALESCE(type_guess,''), risk_score, first_seen, last_seen,
		       tags, COALESCE(notes,'')
		FROM devices WHERE ip = ?
		ORDER BY last_seen DESC LIMIT 1
	`, ip).Scan(&dev.ID, &dev.MAC, &dev.IP, &dev.Hostname, &dev.Vendor,
		&dev.TypeGuess, &dev.RiskScore, &dev.FirstSeen, &dev.LastSeen,
		&tagsJSON, &dev.Notes)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse tags JSON
	dev.Tags = []string{}
	if tagsJSON != "[]" && tagsJSON != "" {
		tagsStr := strings.TrimSuffix(strings.TrimPrefix(tagsJSON, "[\""), "\"]")
		if tagsStr != "" {
			dev.Tags = strings.Split(tagsStr, "\",\"")
		}
	}

	return &dev, nil
}

// GetDeviceActivity retrieves activity statistics for a device
func (d *DeviceDB) GetDeviceActivity(deviceID string) (*DeviceActivity, error) {
	activity := &DeviceActivity{}

	// Get total events count for this device
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM events WHERE device_id = ?
	`, deviceID).Scan(&activity.TotalEvents)
	if err != nil {
		activity.TotalEvents = 0
	}

	// Get alerts count
	err = d.db.QueryRow(`
		SELECT COUNT(*) FROM alerts WHERE device_id = ?
	`, deviceID).Scan(&activity.AlertsCount)
	if err != nil {
		activity.AlertsCount = 0
	}

	// Get first and last activity
	err = d.db.QueryRow(`
		SELECT
			(SELECT COALESCE(MIN(ts), '') FROM events WHERE device_id = ?) as first,
			(SELECT COALESCE(MAX(ts), '') FROM events WHERE device_id = ?) as last
	`, deviceID, deviceID).Scan(&activity.FirstSeen, &activity.LastActivity)
	if err != nil {
		activity.FirstSeen = ""
		activity.LastActivity = ""
	}

	// Count connections (estimate from unique IPs/ports in events)
	err = d.db.QueryRow(`
		SELECT COUNT(DISTINCT ts) FROM events WHERE device_id = ? AND category = 'network'
	`, deviceID).Scan(&activity.ConnectionCount)
	if err != nil {
		activity.ConnectionCount = 0
	}

	return activity, nil
}

// ListDevices returns all devices with optional filtering
func (d *DeviceDB) ListDevices(limit int, search string) ([]Device, error) {
	query := `
		SELECT id, mac, COALESCE(ip,''), COALESCE(hostname,''), COALESCE(vendor,''),
		       COALESCE(type_guess,''), risk_score, last_seen
		FROM devices
	`
	args := []interface{}{}

	if search != "" {
		query += ` WHERE mac LIKE '%'||?||'%' OR hostname LIKE '%'||?||'%' OR ip LIKE '%'||?||'%'`
		args = append(args, search, search, search)
	}

	query += ` ORDER BY last_seen DESC LIMIT ?`
	args = append(args, limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.ID, &dev.MAC, &dev.IP, &dev.Hostname,
			&dev.Vendor, &dev.TypeGuess, &dev.RiskScore, &dev.LastSeen); err != nil {
			return nil, err
		}
		devices = append(devices, dev)
	}

	return devices, nil
}

// EventDB handles event database operations
type EventDB struct {
	db *sql.DB
}

// NewEventDB creates a new event database handler
func NewEventDB(db *sql.DB) *EventDB {
	return &EventDB{db: db}
}

// CreateEvent creates a new event
func (e *EventDB) CreateEvent(event Event) error {
	rawJSON := "{}"
	if event.Raw != nil {
		// Proper JSON encoding
		jsonBytes, err := json.Marshal(event.Raw)
		if err == nil {
			rawJSON = string(jsonBytes)
		}
	}

	// Handle device_id pointer
	var deviceIDVal interface{}
	if event.DeviceID != nil {
		deviceIDVal = *event.DeviceID
	} else {
		deviceIDVal = nil
	}

	_, err := e.db.Exec(`
		INSERT INTO events (id, ts, device_id, category, severity, summary, raw)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.TS, deviceIDVal, event.Category, event.Severity,
		event.Summary, rawJSON)

	return err
}

// ListEvents returns events with filtering
func (e *EventDB) ListEvents(limit int, since string, minSeverity int, deviceID string) ([]Event, error) {
	query := `
		SELECT id, ts, device_id, category, severity, summary, raw
		FROM events
		WHERE ts >= ? AND severity >= ?
	`
	args := []interface{}{since, minSeverity}

	if deviceID != "" {
		query += ` AND device_id = ?`
		args = append(args, deviceID)
	}

	query += ` ORDER BY ts DESC LIMIT ?`
	args = append(args, limit)

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var ev Event
		var rawJSON string
		var deviceID sql.NullString

		if err := rows.Scan(&ev.ID, &ev.TS, &deviceID, &ev.Category,
			&ev.Severity, &ev.Summary, &rawJSON); err != nil {
			return nil, err
		}

		if deviceID.Valid {
			ev.DeviceID = &deviceID.String
		}

		ev.Raw = map[string]interface{}{"data": rawJSON}
		events = append(events, ev)
	}

	return events, nil
}
