package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// TelegramService handles Telegram bot notifications and chat
type TelegramService struct {
	botToken      string
	chatID        string
	allowedUser   string
	enabled       bool
	aiClient      *LMStudioClient
	lastAlertID   string
	conversations map[string]*Conversation
	mu            sync.RWMutex
	hub           *WSHub
	apiURL        string        // Clawbot API URL
	httpClient    *http.Client  // HTTP client for API calls
}

// Conversation stores chat history for context
type Conversation struct {
	UserID      string
	Messages    []ChatMessage
	LastUpdated time.Time
	mu          sync.Mutex
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
	Time    time.Time
}

// TelegramMessage represents a message sent to Telegram
type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// TelegramUpdate represents an incoming update from Telegram
type TelegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		MessageID int64  `json:"message_id"`
		From      *struct {
			ID        int64  `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		} `json:"from"`
		Chat *struct {
			ID    int64  `json:"id"`
			Type  string `json:"type"`
		} `json:"chat"`
		Text  string `json:"text"`
	} `json:"message"`
}

// NewTelegramService creates a new Telegram notification service
// NOTE: Chat interaktif sekarang ditangani oleh OpenClaw, bukan bodyguard-core
// Service ini HANYA mengirim alert notifikasi, tidak menerima atau memproses pesan chat
func NewTelegramService(aiClient *LMStudioClient) *TelegramService {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	allowedUser := os.Getenv("TELEGRAM_ALLOWED_USER_IDS")
	enabledStr := os.Getenv("TELEGRAM_ALERTS_ENABLED")

	// Cek apakah chat interaktif dinonaktifkan (default: true untuk nonaktif)
	chatDisabled := os.Getenv("TELEGRAM_CHAT_DISABLED") == "true"

	enabled := enabledStr == "true" && botToken != "" && chatID != ""

	ts := &TelegramService{
		botToken:      botToken,
		chatID:        chatID,
		allowedUser:   allowedUser,
		enabled:       enabled,
		aiClient:      aiClient,
		conversations: make(map[string]*Conversation),
		apiURL:        "http://127.0.0.1:8787/v1",
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}

	// JANGAN start polling untuk updates - chat interaktif ditangani OpenClaw
	// Polling hanya dijalankan jika chat interaktif eksplisit diaktifkan
	if enabled && !chatDisabled {
		log.Printf("[Telegram] WARNING: Interactive chat is deprecated. Use OpenClaw for chat functionality.")
		go ts.pollForUpdates()
	} else if enabled {
		log.Printf("[Telegram] Alert-only mode (chat handled by OpenClaw)")
	}

	return ts
}

// SetHub sets the WebSocket hub for broadcasting chat events
func (t *TelegramService) SetHub(hub *WSHub) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hub = hub
}

// pollForUpdates continuously polls Telegram for new messages
func (t *TelegramService) pollForUpdates() {
	var lastUpdateID int64

	for {
		select {
		case <-time.After(3 * time.Second):
		}

		// Get updates from Telegram
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=10", t.botToken, lastUpdateID+1)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("[Telegram] Error getting updates: %v", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[Telegram] Error reading response: %v", err)
			continue
		}

		var updatesResp struct {
			Ok     bool               `json:"ok"`
			Result []TelegramUpdate `json:"result"`
		}

		if err := json.Unmarshal(body, &updatesResp); err != nil {
			log.Printf("[Telegram] Error parsing updates: %v", err)
			continue
		}

		if !updatesResp.Ok {
			continue
		}

		// Process each update
		for _, update := range updatesResp.Result {
			lastUpdateID = update.UpdateID

			if update.Message != nil && update.Message.Text != "" {
				// Check if user is allowed
				if t.isAllowedUser(update.Message.From.ID) {
					go t.handleUserMessage(update.Message)
				} else {
					log.Printf("[Telegram] Unauthorized user: %d", update.Message.From.ID)
				}
			}
		}
	}
}

// isAllowedUser checks if the user is allowed to interact with the bot
func (t *TelegramService) isAllowedUser(userID int64) bool {
	if t.allowedUser == "" {
		return true // No restriction if not configured
	}

	// Check if userID is in the allowed list
	allowedIDs := strings.Split(t.allowedUser, ",")
	for _, idStr := range allowedIDs {
		var id int64
		fmt.Sscanf(idStr, "%d", &id)
		if id == userID {
			return true
		}
	}

	return false
}

// handleUserMessage processes an incoming message from a user
func (t *TelegramService) handleUserMessage(msg *struct {
	MessageID int64  `json:"message_id"`
	From      *struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
	} `json:"from"`
	Chat *struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
	Text string `json:"text"`
}) {
	userID := fmt.Sprintf("%d", msg.Chat.ID)
	userText := strings.TrimSpace(msg.Text)

	// Log the message
	log.Printf("[Telegram] Message from %s (%s): %s", msg.From.FirstName, msg.From.Username, userText)

	// Handle commands
	if strings.HasPrefix(userText, "/") {
		t.handleCommand(userID, userText, msg.From.FirstName)
		return
	}

	// Handle chat with AI
	t.handleChatMessage(userID, userText)
}

// handleCommand handles bot commands
func (t *TelegramService) handleCommand(userID, command, firstName string) {
	commands := map[string]func(string){
		"/start": func(uid string) {
			t.sendReply(uid, `👋 *Selamat datang di AmmanGate Security Bot!*

Saya adalah asisten AI keamanan rumah Anda. Saya bisa:

💬 *Chat* - Tanya apa saja tentang keamanan
🔍 *Analisis* - Jelaskan alert keamanan
📊 *Status* - Cek status sistem
📱 *Perangkat* - Lihat perangkat yang terhubung
❓ *Help* - Lihat semua perintah

Ketik pesan Anda untuk mulai ngobrol dengan saya!`)
		},
		"/help": func(uid string) {
			t.sendReply(uid, `📚 *Daftar Perintah*

/start - Mulai bot
/status - Status sistem keamanan
/devices - Daftar perangkat yang terhubung
/alerts - Alert terakhir
/clear - Bersihkan percakapan
/help - Bantuan

*Tips:* Anda bisa langsung ngobrol dengan saya tanpa perintah!`)
		},
		"/status": func(uid string) {
			status, err := t.getSystemStatus()
			if err != nil {
				t.sendReply(uid, fmt.Sprintf("❌ Gagal mengambil status: %v", err))
				return
			}

			// Format uptime
			uptime := "N/A"
			if status.UptimeSec > 0 {
				hours := status.UptimeSec / 3600
				mins := (status.UptimeSec % 3600) / 60
				uptime = fmt.Sprintf("%d jam %d menit", hours, mins)
			}

			// Format sensor status
			sensors := "🔴 Offline"
			if len(status.Sensors) > 0 {
				activeSensors := []string{}
				if status.Sensors["arp"] { activeSensors = append(activeSensors, "ARP") }
				if status.Sensors["dhcp"] { activeSensors = append(activeSensors, "DHCP") }
				if status.Sensors["suricata"] { activeSensors = append(activeSensors, "IDS") }
				if len(activeSensors) > 0 {
					sensors = "✅ " + strings.Join(activeSensors, ", ")
				}
			}

			// Format ClamAV status
			clamav := "🔴 Tidak aktif"
			if status.ClamAV != nil {
				if enabled, ok := status.ClamAV["enabled"].(bool); ok && enabled {
					clamav = "✅ Aktif"
				}
			}

			t.sendReply(uid, fmt.Sprintf(`📊 *Status Sistem AmmanGate*

⏱️ *Uptime:* %s
💻 *CPU Load:* %.1f%%
🧠 *Memory:* %d MB
🛡️ *Sensor:* %s
🦠 *ClamAV:* %s

Sistem aktif dan memantau keamanan jaringan Anda.`,
				uptime, status.CpuLoad, status.MemUsedMB, sensors, clamav))
		},
		"/devices": func(uid string) {
			devices, err := t.getDevices()
			if err != nil {
				t.sendReply(uid, fmt.Sprintf("❌ Gagal mengambil daftar perangkat: %v", err))
				return
			}

			if len(devices) == 0 {
				t.sendReply(uid, "📱 *Perangkat Terhubung*\n\nBelum ada perangkat yang terdeteksi.")
				return
			}

			// Count online devices
			onlineCount := 0
			deviceList := ""
			for i, dev := range devices {
				if i >= 10 { break } // Limit to 10 devices
				statusIcon := "🔴"
				if dev.IP != "" {
					statusIcon = "🟢"
					onlineCount++
				}
				deviceList += fmt.Sprintf("%s %s - %s\n", statusIcon, dev.Hostname, dev.IP)
			}

			t.sendReply(uid, fmt.Sprintf(`📱 *Perangkat Terhubung* (%d/%d online)

%s
%s`,
				onlineCount, len(devices), deviceList,
				map[bool]string{true: "\nKetik 'tampilkan semua perangkat' untuk melihat lebih banyak.", false: ""}[len(devices) > 10]))
		},
		"/alerts": func(uid string) {
			alerts, err := t.getAlerts(5)
			if err != nil {
				t.sendReply(uid, fmt.Sprintf("❌ Gagal mengambil alert: %v", err))
				return
			}

			if len(alerts) == 0 {
				t.sendReply(uid, "✅ *Tidak Ada Alert*\n\nSistem aman, tidak ada ancaman terdeteksi baru-baru ini.")
				return
			}

			alertList := ""
			for _, alert := range alerts {
				severity := "⚪"
				if alert.Severity >= 3 { severity = "🟡" }
				if alert.Severity >= 5 { severity = "🔴" }
				alertList += fmt.Sprintf("%s %s\n", severity, alert.Summary)
			}

			t.sendReply(uid, fmt.Sprintf(`🚨 *Alert Terakhir* (%d alert)

%s`,
				len(alerts), alertList))
		},
		"/clear": func(uid string) {
			t.mu.Lock()
			defer t.mu.Unlock()
			delete(t.conversations, uid)
			t.sendReply(uid, `✅ Percakapan dibersihkan. Mari mulai percakapan baru!`)
		},
	}

	cmdHandler, exists := commands[command]
	if exists {
		cmdHandler(userID)
	} else {
		t.sendReply(userID, fmt.Sprintf("❓ Perintah tidak dikenal: %s\n\nKetik /help untuk daftar perintah.", command))
	}
}

// handleChatMessage processes a chat message with AI
func (t *TelegramService) handleChatMessage(userID, userText string) {
	// Show "typing" indicator
	_ = t.sendChatAction(userID, "typing")

	// Get or create conversation
	conv := t.getOrCreateConversation(userID)

	// Add user message to conversation
	conv.mu.Lock()
	conv.Messages = append(conv.Messages, ChatMessage{
		Role:    "user",
		Content: userText,
		Time:    time.Now(),
	})
	conv.LastUpdated = time.Now()

	// Keep only last 20 messages for context
	if len(conv.Messages) > 20 {
		conv.Messages = conv.Messages[len(conv.Messages)-20:]
	}
	conv.mu.Unlock()

	// Generate AI response
	aiResponse, err := t.generateChatResponse(conv)
	if err != nil {
		_ = t.sendReply(userID, fmt.Sprintf("❌ Maaf, terjadi kesalahan: %v\n\nSilakan coba lagi atau hubungi admin.", err))
		return
	}

	// Add assistant response to conversation
	conv.mu.Lock()
	conv.Messages = append(conv.Messages, ChatMessage{
		Role:    "assistant",
		Content: aiResponse,
		Time:    time.Now(),
	})
	conv.LastUpdated = time.Now()
	conv.mu.Unlock()

	// Send reply
	_ = t.sendReply(userID, aiResponse)

	// Broadcast chat event to WebSocket
	if t.hub != nil {
		t.hub.Broadcast("telegram_chat", map[string]interface{}{
			"user_id":     userID,
			"user_message": userText,
			"ai_response":  aiResponse,
			"timestamp":    time.Now().Format(time.RFC3339),
		})
	}
}

// generateChatResponse generates AI response for chat with system context
func (t *TelegramService) generateChatResponse(conv *Conversation) (string, error) {
	if t.aiClient == nil {
		return "", fmt.Errorf("AI not available")
	}

	// Build system context with real data
	var context strings.Builder
	context.WriteString("Anda adalah asisten AI untuk AmmanGate Security System. Anda memiliki akses ke data sistem secara real-time.\n\n")

	// Fetch system data
	status, err := t.getSystemStatus()
	if err == nil && status != nil {
		context.WriteString("=== STATUS SISTEM SAAT INI ===\n")
		uptime := "N/A"
		if status.UptimeSec > 0 {
			hours := status.UptimeSec / 3600
			mins := (status.UptimeSec % 3600) / 60
			uptime = fmt.Sprintf("%d jam %d menit", hours, mins)
		}
		context.WriteString(fmt.Sprintf("Uptime: %s\n", uptime))
		context.WriteString(fmt.Sprintf("CPU Load: %.1f%%\n", status.CpuLoad))
		context.WriteString(fmt.Sprintf("Memory: %d MB\n", status.MemUsedMB))

		if len(status.Sensors) > 0 {
			activeSensors := []string{}
			if status.Sensors["arp"] { activeSensors = append(activeSensors, "ARP") }
			if status.Sensors["dhcp"] { activeSensors = append(activeSensors, "DHCP") }
			if status.Sensors["suricata"] { activeSensors = append(activeSensors, "Suricata IDS") }
			if status.Sensors["dns"] { activeSensors = append(activeSensors, "DNS") }
			if len(activeSensors) > 0 {
				context.WriteString(fmt.Sprintf("Sensor Aktif: %s\n", strings.Join(activeSensors, ", ")))
			}
		}
		context.WriteString("\n")
	}

	// Fetch devices count
	devices, err := t.getDevices()
	if err == nil && devices != nil {
		onlineCount := 0
		for _, dev := range devices {
			if dev.IP != "" {
				onlineCount++
			}
		}
		context.WriteString(fmt.Sprintf("=== PERANGKAT ===\nTotal: %d perangkat\nOnline: %d perangkat\n\n", len(devices), onlineCount))
	}

	// Fetch recent alerts
	alerts, err := t.getAlerts(3)
	if err == nil && alerts != nil && len(alerts) > 0 {
		context.WriteString("=== ALERT TERAKHIR ===\n")
		for _, alert := range alerts {
			severity := "Low"
			if alert.Severity >= 3 { severity = "Medium" }
			if alert.Severity >= 5 { severity = "High" }
			context.WriteString(fmt.Sprintf("- [%s] %s\n", severity, alert.Summary))
		}
		context.WriteString("\n")
	}

	// Add conversation history
	context.WriteString("=== PERCAKAPAN SEBELUMNYA ===\n")
	conv.mu.Lock()
	for _, msg := range conv.Messages {
		if msg.Role == "user" {
			context.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
		} else {
			context.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
		}
	}
	conv.mu.Unlock()

	// Get last user message as the actual prompt
	lastMsg := conv.Messages[len(conv.Messages)-1]
	prompt := fmt.Sprintf("%s\nUser: %s\n\nJawab pertanyaan user dengan data sistem di atas. Berikan jawaban yang singkat, jelas, dan membantu.", context.String(), lastMsg.Content)

	response, err := t.aiClient.Complete(prompt)
	if err != nil {
		return "", err
	}

	// Clean up response - remove any extra "Assistant:" prefix
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "Assistant:")
	response = strings.TrimPrefix(response, "assistant:")
	response = strings.TrimSpace(response)

	return response, nil
}

// getOrCreateConversation gets or creates a conversation for a user
func (t *TelegramService) getOrCreateConversation(userID string) *Conversation {
	t.mu.Lock()
	defer t.mu.Unlock()

	if conv, exists := t.conversations[userID]; exists {
		return conv
	}

	conv := &Conversation{
		UserID:      userID,
		Messages:    make([]ChatMessage, 0, 20),
		LastUpdated: time.Now(),
	}

	t.conversations[userID] = conv
	return conv
}

// sendReply sends a reply to a user
func (t *TelegramService) sendReply(userID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	message := TelegramMessage{
		ChatID:    userID,
		Text:      text,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendChatAction sends a chat action (typing, etc.)
func (t *TelegramService) sendChatAction(userID, action string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", t.botToken)

	actionData := map[string]interface{}{
		"chat_id": userID,
		"action":  action,
	}

	jsonData, err := json.Marshal(actionData)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// getSystemStatus fetches current system status from Clawbot API
func (t *TelegramService) getSystemStatus() (*SystemStatus, error) {
	resp, err := t.httpClient.Get(t.apiURL + "/system/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status SystemStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

// getDevices fetches list of devices from Clawbot API
func (t *TelegramService) getDevices() ([]Device, error) {
	resp, err := t.httpClient.Get(t.apiURL + "/devices?limit=100")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var devices []Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// getAlerts fetches recent alerts from Clawbot API
func (t *TelegramService) getAlerts(limit int) ([]Event, error) {
	resp, err := t.httpClient.Get(fmt.Sprintf("%s/alerts/active?limit=%d", t.apiURL, limit))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var alerts []Event
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

// getSuricataStatus fetches Suricata IDS status
func (t *TelegramService) getSuricataStatus() (map[string]interface{}, error) {
	resp, err := t.httpClient.Get(t.apiURL + "/suricata/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return status, nil
}

// SendAlert sends a security alert to Telegram with AI explanation
func (t *TelegramService) SendAlert(alert *SuricataAlert) error {
	if !t.enabled {
		return nil
	}

	// Generate AI explanation
	explanation, err := t.generateAlertExplanation(alert)
	if err != nil {
		explanation = fmt.Sprintf("*Unable to generate AI explanation: %v*\n\n", err)
	}

	// Format the alert message
	message := t.formatAlertMessage(alert, explanation)

	// Send to Telegram
	return t.sendMessageToChat(t.chatID, message)
}

// SendTestMessage sends a test message to verify Telegram integration
func (t *TelegramService) SendTestMessage() error {
	if !t.enabled {
		return fmt.Errorf("telegram notifications are disabled")
	}

	message := fmt.Sprintf(`🤖 *AmmanGate Security Alert Test*

✅ AmmanGate Security System is online and connected!

*System Status:*
- Suricata IDS: Active
- ClamAV Antivirus: Active
- Telegram Notifications: Enabled

🤖 *AI Chat:* Aktif - Kirim pesan untuk ngobrol dengan AI!

You will receive security alerts here when threats are detected.

_Test sent at: %s_`, time.Now().Format("2006-01-02 15:04:05"))

	return t.sendMessageToChat(t.chatID, message)
}

// generateAlertExplanation uses AI to explain the alert
func (t *TelegramService) generateAlertExplanation(alert *SuricataAlert) (string, error) {
	if t.aiClient == nil {
		return "", fmt.Errorf("AI client not available")
	}

	prompt := fmt.Sprintf(`Analyze this Suricata IDS security alert and provide a concise explanation in Indonesian:

ALERT DETAILS:
- Signature: %s
- Category: %s
- Severity: %d
- Source: %s:%d
- Destination: %s:%d
- Protocol: %s
- Timestamp: %s

Please provide:
1. Apa yang dimaksud dengan alert ini (bahasa sederhana)
2. Kenapa aktivitas ini mencurigakan
3. Rekomendasi tindakan untuk user

Jawab singkat, padat, dan actionable. Gunakan format plain text.`,
		alert.Signature,
		alert.Category,
		alert.Severity,
		alert.SrcIP,
		alert.SrcPort,
		alert.DestIP,
		alert.DestPort,
		alert.Proto,
		alert.Timestamp.Format("2006-01-02 15:04:05"),
	)

	response, err := t.aiClient.Complete(prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

// formatAlertMessage creates a formatted alert message for Telegram
func (t *TelegramService) formatAlertMessage(alert *SuricataAlert, explanation string) string {
	var buf bytes.Buffer

	// Header with alert level emoji
	severityEmoji := t.getSeverityEmoji(alert.Severity)
	buf.WriteString(fmt.Sprintf("%s *AmmanGate Security Alert*\n\n", severityEmoji))

	// Alert details
	buf.WriteString("*🚨 DETECTION*\n")
	buf.WriteString(fmt.Sprintf("`%s`\n", alert.Signature))
	buf.WriteString(fmt.Sprintf("_%s_\n\n", alert.Category))

	// Connection details
	buf.WriteString("*📡 KONEKSI*\n")
	buf.WriteString(fmt.Sprintf("• Dari: `%s:%d`\n", alert.SrcIP, alert.SrcPort))
	buf.WriteString(fmt.Sprintf("• Ke: `%s:%d`\n", alert.DestIP, alert.DestPort))
	buf.WriteString(fmt.Sprintf("• Protokol: `%s`\n", alert.Proto))
	buf.WriteString(fmt.Sprintf("• Waktu: `%s`\n\n", alert.Timestamp.Format("2006-01-02 15:04:05")))

	// Severity indicator
	buf.WriteString("*⚠️ SEVERITAS*\n")
	severityBar := t.getSeverityBar(alert.Severity)
	buf.WriteString(fmt.Sprintf("%s (%d/3)\n\n", severityBar, alert.Severity))

	// AI Explanation
	buf.WriteString("*🤖 ANALISIS AI*\n")
	buf.WriteString(explanation)
	buf.WriteString("\n\n")

	// Footer
	buf.WriteString("_Powered by AmmanGate Home Security_")

	return buf.String()
}

// sendMessageToChat sends a message to a specific chat
func (t *TelegramService) sendMessageToChat(chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	message := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendMessage sends a message to Telegram (legacy method)
func (t *TelegramService) sendMessage(text string) error {
	return t.sendMessageToChat(t.chatID, text)
}

// getSeverityEmoji returns an emoji based on severity level
func (t *TelegramService) getSeverityEmoji(severity int) string {
	switch severity {
	case 1:
		return "🔴"
	case 2:
		return "🟠"
	case 3:
		return "🟡"
	default:
		return "⚪"
	}
}

// getSeverityBar returns a visual severity bar
func (t *TelegramService) getSeverityBar(severity int) string {
	switch severity {
	case 1:
		return "███"
	case 2:
		return "█▓▓"
	case 3:
		return "█░░"
	default:
		return "░░░"
	}
}

// SendCustomMessage sends a custom message to Telegram
func (t *TelegramService) SendCustomMessage(text string) error {
	if !t.enabled {
		return nil
	}
	return t.sendMessage(text)
}

// IsEnabled returns whether Telegram notifications are enabled
func (t *TelegramService) IsEnabled() bool {
	return t.enabled
}

// GetStatus returns the status of Telegram service
func (t *TelegramService) GetStatus() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]interface{}{
		"enabled":        t.enabled,
		"configured":     t.botToken != "" && t.chatID != "",
		"chat_id":        t.maskChatID(),
		"active_chats":   len(t.conversations),
		"allowed_users":  t.allowedUser,
	}
}

// GetConversations returns active conversations
func (t *TelegramService) GetConversations() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]interface{})
	for userID, conv := range t.conversations {
		conv.mu.Lock()
		messages := make([]map[string]interface{}, len(conv.Messages))
		for i, msg := range conv.Messages {
			messages[i] = map[string]interface{}{
				"role":     msg.Role,
				"content":  msg.Content,
				"time":     msg.Time.Format(time.RFC3339),
			}
		}
		conv.mu.Unlock()

		result[userID] = map[string]interface{}{
			"message_count":  len(conv.Messages),
			"last_updated":  conv.LastUpdated.Format(time.RFC3339),
			"messages":      messages,
		}
	}

	return result
}

// maskChatID masks the chat ID for security
func (t *TelegramService) maskChatID() string {
	if t.chatID == "" {
		return ""
	}
	if len(t.chatID) <= 4 {
		return "****"
	}
	return t.chatID[:2] + "****" + t.chatID[len(t.chatID)-2:]
}
