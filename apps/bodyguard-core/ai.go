package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// AIEngine provides AI-powered security analysis using LM Studio
type AIEngine struct {
	db           *sql.DB
	hub          *WSHub
	devDB        *DeviceDB
	eventDB      *EventDB
	filterEngine *FilterEngine
	apiURL       string
	model        string
	token        string
	client       *http.Client
}

// NewAIEngine creates a new AI engine
func NewAIEngine(db *sql.DB, hub *WSHub, devDB *DeviceDB, eventDB *EventDB, filterEngine *FilterEngine) *AIEngine {
	apiURL := env("BG_LM_STUDIO_URL", "http://localhost:1234/v1")
	model := env("BG_LM_STUDIO_MODEL", "huihui-ai_-_qwen2.5-coder-7b-instruct-abliterated")
	token := env("BG_LM_STUDIO_TOKEN", "") // Optional API token

	return &AIEngine{
		db:           db,
		hub:          hub,
		devDB:        devDB,
		eventDB:      eventDB,
		filterEngine: filterEngine,
		apiURL:       apiURL,
		model:        model,
		token:        token,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// LM Studio API request/response structures
type LMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LMRequest struct {
	Model    string     `json:"model"`
	Messages []LMMessage `json:"messages"`
	Stream   bool       `json:"stream"`
}

type LMResponse struct {
	Choices []struct {
		Message LMMessage `json:"message"`
	} `json:"choices"`
}

// ExplainRequest represents a request for explanation
type ExplainRequest struct {
	AlertID  string   `json:"alert_id,omitempty"`
	DeviceID string   `json:"device_id,omitempty"`
	EventIDs []string `json:"event_ids,omitempty"`
	Question string   `json:"question,omitempty"`
}

// ExplainResponse represents the AI explanation
type ExplainResponse struct {
	Narrative          string                 `json:"narrative"`
	SuspectedCause     string                 `json:"suspected_cause"`
	RecommendedActions []string               `json:"recommended_actions"`
	Confidence         float64                `json:"confidence"`
	Command            map[string]interface{} `json:"command,omitempty"`
	RelatedEvents      []Event                `json:"related_events,omitempty"`
	RelatedDevices     []Device               `json:"related_devices,omitempty"`
}

// Explain generates an explanation for security events using LM Studio
func (ai *AIEngine) Explain(req ExplainRequest) (ExplainResponse, error) {
	// Build context from database
	context := ai.buildContext(req)

	// Build prompt for LM Studio
	prompt := ai.buildPrompt(req, context)

	// Call LM Studio API
	response, err := ai.callLMStudio(prompt)
	if err != nil {
		// Fallback to rule-based if LM Studio fails
		return ai.fallbackExplain(req, context)
	}

	// Parse AI response
	resp, err := ai.parseAIResponse(req, response, context)
	if err != nil {
		return resp, err
	}

	// Execute command if present
	if resp.Command != nil {
		ai.executeCommand(resp.Command)
	}

	return resp, nil
}

// buildContext gathers relevant context from the database
func (ai *AIEngine) buildContext(req ExplainRequest) map[string]interface{} {
	context := make(map[string]interface{})

	// Get device count
	if count, err := ai.getDeviceCount(); err == nil {
		context["device_count"] = count
	}

	// Get active alert count
	if count, err := ai.getActiveAlertCount(); err == nil {
		context["alert_count"] = count
	}

	// Get device info if device_id is provided
	if req.DeviceID != "" {
		if device, err := ai.devDB.GetDeviceByID(req.DeviceID); err == nil {
			context["device"] = device
		}
		// Get recent events for this device
		if events, err := ai.eventDB.ListEvents(20, time.Now().Add(-24*time.Hour).UTC().Format(time.RFC3339), 1, req.DeviceID); err == nil {
			context["device_events"] = events
		}
	}

	// Get recent events
	if events, err := ai.eventDB.ListEvents(20, time.Now().Add(-1*time.Hour).UTC().Format(time.RFC3339), 1, ""); err == nil {
		context["recent_events"] = events
	}

	return context
}

// buildPrompt creates the prompt for LM Studio
func (ai *AIEngine) buildPrompt(req ExplainRequest, context map[string]interface{}) string {
	systemPrompt := `You are a cybersecurity AI assistant for AmmanGate, a home network security system.
Your task is to analyze network security data and provide clear, actionable insights.

For general queries, respond in this JSON format:
{
  "narrative": "Human-readable explanation with markdown formatting",
  "suspected_cause": "Brief explanation of what might be happening",
  "recommended_actions": ["action1", "action2", "action3"],
  "confidence": 0.0-1.0
}

For PARENTAL CONTROL commands, respond in this JSON format:
{
  "narrative": "Response message",
  "command": {
    "action": "set_filter|add_rule|remove_rule|list_rules|set_profile",
    "device_id": "device-id (for set_profile)",
    "filter_level": "off|light|moderate|strict (for set_filter/set_profile)",
    "rule_name": "rule name (for add_rule)",
    "rule_type": "domain|category (for add_rule)",
    "pattern": "pattern (for add_rule)",
    "rule_id": "rule-id (for remove_rule/toggle_rule)",
    "enabled": true/false (for toggle_rule)"
  },
  "confidence": 0.0-1.0
}

PARENTAL CONTROL COMMANDS:
- "set parental control to strict/medium/light for device X" -> set_profile action
- "block domain example.com" -> add_rule action with type=domain
- "block gambling sites" -> add_rule action with type=category
- "unblock domain example.com" -> remove_rule action
- "show parental control rules" -> list_rules action
- "disable parental control for device X" -> set_profile with level=off

Keep responses concise and practical. Use emoji for better readability.
Risk scores: 0-30 (low), 31-60 (medium), 61-100 (high).`

	var userPrompt string

	if req.DeviceID != "" {
		device, _ := context["device"].(*DeviceDetail)
		if device != nil {
			userPrompt = fmt.Sprintf(`Analyze this network device:

Device: %s
IP: %s
MAC: %s
Vendor: %s
Type: %s
Risk Score: %d/100
Last Seen: %s

Recent activity: %d events in the last 24 hours

Provide a security assessment and recommendations.`,
				device.Hostname,
				device.IP,
				device.MAC,
				device.Vendor,
				device.TypeGuess,
				device.RiskScore,
				device.LastSeen,
				len(context["device_events"].([]Event)),
			)
		}
	} else if req.Question != "" {
		// Check if this is a HONEYPOT alert - use question directly without network context
		if strings.Contains(req.Question, "**HONEYPOT ALERT**") {
			userPrompt = req.Question
		} else {
			deviceCount, _ := context["device_count"].(int)
			alertCount, _ := context["alert_count"].(int)

			userPrompt = fmt.Sprintf(`Current network status:
- Total devices: %d
- Active alerts: %d

User question: %s

Provide a helpful response.`,
				deviceCount,
				alertCount,
				req.Question,
			)
		}
	} else {
		// General system overview
		deviceCount, _ := context["device_count"].(int)
		alertCount, _ := context["alert_count"].(int)

		userPrompt = fmt.Sprintf(`Analyze this home network security status:

Current status:
- Total devices: %d
- Active alerts: %d

Provide a comprehensive security assessment.`,
			deviceCount,
			alertCount,
		)
	}

	return fmt.Sprintf("%s\n\n%s", systemPrompt, userPrompt)
}

// callLMStudio calls the LM Studio API
func (ai *AIEngine) callLMStudio(prompt string) (string, error) {
	reqBody := LMRequest{
		Model: ai.model,
		Messages: []LMMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", ai.apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authorization header if token is provided
	if ai.token != "" {
		req.Header.Set("Authorization", "Bearer "+ai.token)
	}

	resp, err := ai.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call LM Studio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LM Studio returned status %d: %s", resp.StatusCode, string(body))
	}

	var lmResp LMResponse
	if err := json.NewDecoder(resp.Body).Decode(&lmResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(lmResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LM Studio")
	}

	return lmResp.Choices[0].Message.Content, nil
}

// parseAIResponse parses the AI response into ExplainResponse
func (ai *AIEngine) parseAIResponse(req ExplainRequest, aiResponse string, context map[string]interface{}) (ExplainResponse, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(aiResponse, "{")
	jsonEnd := strings.LastIndex(aiResponse, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		// No JSON found, use entire response as narrative
		return ExplainResponse{
			Narrative:          aiResponse,
			SuspectedCause:     "AI analysis completed",
			RecommendedActions: []string{"Review the analysis above"},
			Confidence:         0.75,
		}, nil
	}

	jsonStr := aiResponse[jsonStart : jsonEnd+1]

	var result ExplainResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Invalid JSON, use raw response
		return ExplainResponse{
			Narrative:          aiResponse,
			SuspectedCause:     "AI analysis completed",
			RecommendedActions: []string{"Review the analysis above"},
			Confidence:         0.70,
		}, nil
	}

	// Add related devices if available
	if req.DeviceID != "" {
		if device, ok := context["device"].(*DeviceDetail); ok {
			result.RelatedDevices = []Device{device.Device}
		}
	}

	// Add related events if available
	if events, ok := context["device_events"].([]Event); ok {
		result.RelatedEvents = events
	}

	return result, nil
}

// fallbackExplain provides rule-based explanation when LM Studio is unavailable
func (ai *AIEngine) fallbackExplain(req ExplainRequest, context map[string]interface{}) (ExplainResponse, error) {
	if req.DeviceID != "" {
		return ai.explainDeviceRuleBased(req.DeviceID, context)
	}

	deviceCount, _ := context["device_count"].(int)
	alertCount, _ := context["alert_count"].(int)

	return ExplainResponse{
		Narrative: fmt.Sprintf(
			"📊 **AmmanGate Network Status**\n\n"+
				"- **Total Devices:** %d\n"+
				"- **Active Alerts:** %d\n\n"+
				"⚠️ **Note:** AI analysis is currently unavailable. Using fallback mode.\n\n"+
				"Please ensure LM Studio is running on %s",
			deviceCount, alertCount, ai.apiURL,
		),
		SuspectedCause:     "System status overview (fallback mode)",
		RecommendedActions: []string{"Start LM Studio for AI analysis", "Check network configuration"},
		Confidence:         0.50,
	}, nil
}

// explainDeviceRuleBased provides rule-based device explanation
func (ai *AIEngine) explainDeviceRuleBased(deviceID string, context map[string]interface{}) (ExplainResponse, error) {
	device, _ := context["device"].(*DeviceDetail)
	if device == nil {
		return ExplainResponse{
			Narrative:        "Device not found",
			SuspectedCause:   "Unknown device",
			RecommendedActions: []string{"Check if device is still on network"},
			Confidence:       0.0,
		}, nil
	}

	events := []Event{}
	if e, ok := context["device_events"].([]Event); ok {
		events = e
	}

	response := ExplainResponse{
		Confidence: 0.70,
	}

	if device.RiskScore >= 70 {
		response.Narrative = fmt.Sprintf(
			"🔴 **High Risk Device: %s**\n\nRisk Score: %d/100\n\n"+
				"⚠️ **Note:** AI analysis unavailable. Using rule-based assessment.\n\n"+
				"This device exhibits concerning patterns that warrant investigation.",
			device.Hostname, device.RiskScore,
		)
		response.SuspectedCause = fmt.Sprintf("High risk device (score: %d)", device.RiskScore)
		response.RecommendedActions = []string{
			"🔍 Conduct thorough security investigation",
			"🚨 Consider isolating from network",
			"📋 Review recent activity patterns",
		}
	} else if device.RiskScore >= 40 {
		response.Narrative = fmt.Sprintf(
			"🟡 **Medium Risk Device: %s**\n\nRisk Score: %d/100\n\n"+
				"⚠️ **Note:** AI analysis unavailable. Using rule-based assessment.",
			device.Hostname, device.RiskScore,
		)
		response.SuspectedCause = fmt.Sprintf("Medium risk factors detected (score: %d)", device.RiskScore)
		response.RecommendedActions = []string{
			"👀 Monitor device activity",
			"📊 Review recent events",
		}
	} else {
		response.Narrative = fmt.Sprintf(
			"🟢 **Normal Device: %s**\n\nRisk Score: %d/100\n\n"+
				"Device appears to be operating normally within expected parameters.",
			device.Hostname, device.RiskScore,
		)
		response.SuspectedCause = "Normal device behavior"
		response.RecommendedActions = []string{
			"✅ Continue normal monitoring",
		}
	}

	response.RelatedDevices = []Device{device.Device}
	response.RelatedEvents = events

	return response, nil
}

// Helper functions

func (ai *AIEngine) getDeviceCount() (int, error) {
	var count int
	err := ai.db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	return count, err
}

func (ai *AIEngine) getActiveAlertCount() (int, error) {
	var count int
	err := ai.db.QueryRow("SELECT COUNT(*) FROM alerts WHERE status='active'").Scan(&count)
	return count, err
}

func formatTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}

// executeCommand executes a parental control command from AI response
func (ai *AIEngine) executeCommand(cmd map[string]interface{}) {
	action, _ := cmd["action"].(string)
	if action == "" {
		return
	}

	switch action {
	case "set_profile":
		deviceID, _ := cmd["device_id"].(string)
		filterLevel, _ := cmd["filter_level"].(string)
		if deviceID != "" && filterLevel != "" {
			if err := ai.filterEngine.SetDeviceProfile(deviceID, filterLevel); err != nil {
				log.Printf("[AI] Failed to set device profile: %v", err)
			} else {
				log.Printf("[AI] Set device %s profile to %s", deviceID, filterLevel)
				// Broadcast update
				ai.hub.Broadcast("profile_updated", map[string]interface{}{
					"device_id":    deviceID,
					"filter_level": filterLevel,
				})
			}
		}

	case "add_rule":
		name, _ := cmd["rule_name"].(string)
		ruleType, _ := cmd["rule_type"].(string)
		pattern, _ := cmd["pattern"].(string)
		if name != "" && ruleType != "" && pattern != "" {
			if _, err := ai.filterEngine.AddRule(name, ruleType, pattern); err != nil {
				log.Printf("[AI] Failed to add rule: %v", err)
			} else {
				log.Printf("[AI] Added rule: %s (%s)", name, ruleType)
				// Broadcast update
				ai.hub.Broadcast("rule_added", map[string]interface{}{
					"name":    name,
					"type":    ruleType,
					"pattern": pattern,
				})
			}
		}

	case "remove_rule":
		ruleID, _ := cmd["rule_id"].(string)
		if ruleID != "" {
			if err := ai.filterEngine.RemoveRule(ruleID); err != nil {
				log.Printf("[AI] Failed to remove rule: %v", err)
			} else {
				log.Printf("[AI] Removed rule: %s", ruleID)
				// Broadcast update
				ai.hub.Broadcast("rule_removed", map[string]interface{}{
					"rule_id": ruleID,
				})
			}
		}

	case "toggle_rule":
		ruleID, _ := cmd["rule_id"].(string)
		enabled, _ := cmd["enabled"].(bool)
		if ruleID != "" {
			if err := ai.filterEngine.EnableRule(ruleID, enabled); err != nil {
				log.Printf("[AI] Failed to toggle rule: %v", err)
			} else {
				log.Printf("[AI] %s rule: %s", map[bool]string{true: "Enabled", false: "Disabled"}[enabled], ruleID)
				// Broadcast update
				ai.hub.Broadcast("rule_toggled", map[string]interface{}{
					"rule_id": ruleID,
					"enabled": enabled,
				})
			}
		}
	}
}
