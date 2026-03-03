package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// FilterEngine manages filter rules and matching logic
type FilterEngine struct {
	db    *sql.DB
	rules map[string]*FilterRule // key: ID
	mu    sync.RWMutex
}

// NewFilterEngine creates a new filter engine
func NewFilterEngine(db *sql.DB) *FilterEngine {
	fe := &FilterEngine{
		db:    db,
		rules: make(map[string]*FilterRule),
	}
	fe.LoadRules()
	return fe
}

// LoadRules loads all filter rules from database
func (f *FilterEngine) LoadRules() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	rows, err := f.db.Query(`
		SELECT id, name, type, pattern, enabled, created_at
		FROM filter_rules
		ORDER BY created_at DESC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	f.rules = make(map[string]*FilterRule)
	for rows.Next() {
		var rule FilterRule
		err := rows.Scan(&rule.ID, &rule.Name, &rule.Type, &rule.Pattern, &rule.Enabled, &rule.CreatedAt)
		if err != nil {
			log.Printf("[Filter] Failed to scan rule: %v", err)
			continue
		}
		f.rules[rule.ID] = &rule
	}

	log.Printf("[Filter] Loaded %d filter rules", len(f.rules))
	return nil
}

// MatchDomain checks if a domain matches any filter rule
func (f *FilterEngine) MatchDomain(domain string) *FilterRule {
	domain = strings.ToLower(domain)

	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, rule := range f.rules {
		if !rule.Enabled {
			continue
		}

		if rule.Type == "domain" {
			// Exact domain match (case-insensitive)
			if strings.EqualFold(domain, rule.Pattern) {
				return rule
			}
		} else if rule.Type == "category" {
			// Category match - check if domain contains any category keyword
			keywords := strings.Split(rule.Pattern, ",")
			for _, keyword := range keywords {
				keyword = strings.TrimSpace(strings.ToLower(keyword))
				if strings.Contains(domain, keyword) {
					return rule
				}
			}
		}
	}

	return nil
}

// AddRule adds a new filter rule
func (f *FilterEngine) AddRule(name, ruleType, pattern string) (*FilterRule, error) {
	id := fmt.Sprintf("rule-%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)

	rule := &FilterRule{
		ID:        id,
		Name:      name,
		Type:      ruleType,
		Pattern:   pattern,
		Enabled:   true,
		CreatedAt: now,
	}

	_, err := f.db.Exec(`
		INSERT INTO filter_rules (id, name, type, pattern, enabled, created_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, rule.ID, rule.Name, rule.Type, rule.Pattern, rule.CreatedAt)

	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.rules[rule.ID] = rule
	f.mu.Unlock()

	log.Printf("[Filter] Added rule: %s (%s)", rule.Name, rule.Type)
	return rule, nil
}

// RemoveRule removes a filter rule
func (f *FilterEngine) RemoveRule(id string) error {
	_, err := f.db.Exec(`DELETE FROM filter_rules WHERE id = ?`, id)
	if err != nil {
		return err
	}

	f.mu.Lock()
	delete(f.rules, id)
	f.mu.Unlock()

	log.Printf("[Filter] Removed rule: %s", id)
	return nil
}

// EnableRule enables a filter rule
func (f *FilterEngine) EnableRule(id string, enabled bool) error {
	_, err := f.db.Exec(`UPDATE filter_rules SET enabled = ? WHERE id = ?`, enabled, id)
	if err != nil {
		return err
	}

	f.mu.Lock()
	if rule, exists := f.rules[id]; exists {
		rule.Enabled = enabled
	}
	f.mu.Unlock()

	log.Printf("[Filter] %s rule: %s", map[bool]string{true: "Enabled", false: "Disabled"}[enabled], id)
	return nil
}

// GetRules returns all filter rules
func (f *FilterEngine) GetRules() []*FilterRule {
	f.mu.RLock()
	defer f.mu.RUnlock()

	rules := make([]*FilterRule, 0, len(f.rules))
	for _, rule := range f.rules {
		rules = append(rules, rule)
	}
	return rules
}

// GetRulesByType returns filter rules by type
func (f *FilterEngine) GetRulesByType(ruleType string) []*FilterRule {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var rules []*FilterRule
	for _, rule := range f.rules {
		if rule.Type == ruleType {
			rules = append(rules, rule)
		}
	}
	return rules
}

// SetDeviceProfile sets the parental control profile for a device
func (f *FilterEngine) SetDeviceProfile(deviceID, filterLevel string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Check if profile exists
	var count int
	err := f.db.QueryRow(`SELECT COUNT(*) FROM device_profiles WHERE device_id = ?`, deviceID).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Insert new profile
		_, err = f.db.Exec(`
			INSERT INTO device_profiles (device_id, filter_level, created_at, updated_at)
			VALUES (?, ?, ?, ?)
		`, deviceID, filterLevel, now, now)
	} else {
		// Update existing profile
		_, err = f.db.Exec(`
			UPDATE device_profiles
			SET filter_level = ?, updated_at = ?
			WHERE device_id = ?
		`, filterLevel, now, deviceID)
	}

	if err != nil {
		return err
	}

	log.Printf("[Filter] Set device profile: %s = %s", deviceID, filterLevel)
	return nil
}

// GetDeviceProfile gets the parental control profile for a device
func (f *FilterEngine) GetDeviceProfile(deviceID string) (filterLevel string, err error) {
	err = f.db.QueryRow(`
		SELECT COALESCE(filter_level, 'off')
		FROM device_profiles
		WHERE device_id = ?
	`, deviceID).Scan(&filterLevel)

	if err == sql.ErrNoRows {
		return "off", nil
	}

	return filterLevel, err
}

// GetDNSQueryLogs returns recent DNS query logs
func (f *FilterEngine) GetDNSQueryLogs(limit int) ([]map[string]interface{}, error) {
	rows, err := f.db.Query(`
		SELECT id, ts, device_id, domain, blocked, rule_id
		FROM dns_queries
		ORDER BY ts DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, ts, deviceID, domain string
		var blocked bool
		var ruleID sql.NullString

		err := rows.Scan(&id, &ts, &deviceID, &domain, &blocked, &ruleID)
		if err != nil {
			continue
		}

		logMap := map[string]interface{}{
			"id":        id,
			"ts":        ts,
			"device_id": deviceID,
			"domain":    domain,
			"blocked":   blocked,
			"rule_id":   ruleID.String,
		}
		logs = append(logs, logMap)
	}

	return logs, nil
}
