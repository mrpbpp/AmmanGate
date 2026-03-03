#!/usr/bin/env node

/**
 * AmmanGate Claw Gateway
 *
 * This is the Clawbot gateway that handles ChatOps commands.
 * It acts as a secure bridge between Clawbot and the bodyguard-core API.
 *
 * Architecture:
 * Clawbot -> Command Parser -> API Caller -> Approval Flow -> Response
 *
 * Installation:
 * 1. Install Clawbot: curl -fsSL https://openclaw.ai/install.sh | bash
 * 2. Configure Clawbot to use this gateway's webhook URL
 */

import express from "express";
import fetch from "node-fetch";
import dotenv from "dotenv";

dotenv.config();

const app = express();
app.use(express.json());

// Configuration
const CORE_API_URL = process.env.CORE_API_URL || "http://127.0.0.1:8787";
const PORT = process.env.CLAW_PORT || 3001;
const ACTION_PIN = process.env.ACTION_PIN || "1234";

// Clawbot configuration
const CLAWBOT_API_URL = process.env.CLAWBOT_API_URL || "http://localhost:8080";
const CLAWBOT_API_KEY = process.env.CLAWBOT_API_KEY || "";
const CLAWBOT_WEBHOOK_SECRET = process.env.CLAWBOT_WEBHOOK_SECRET || "";

// Telegram Bot configuration
const TELEGRAM_BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN || "";
const TELEGRAM_ALLOWED_USER_IDS = (process.env.TELEGRAM_ALLOWED_USER_IDS || "").split(",").map(id => id.trim()).filter(id => id);
const USE_TELEGRAM_DIRECT = TELEGRAM_BOT_TOKEN !== "";

// LM Studio configuration
const LM_STUDIO_URL = process.env.BG_LM_STUDIO_URL || "http://localhost:1234/v1";
const LM_STUDIO_MODEL = process.env.BG_LM_STUDIO_MODEL || "local-model";
const LM_STUDIO_TOKEN = process.env.BG_LM_STUDIO_TOKEN || "";

// Quick action shortcuts (user-defined)
const QUICK_ACTIONS = new Map();

// Store pending approvals per chat session
const pendingApprovals = new Map();

/**
 * Clawbot Client
 * Handles communication with Clawbot API
 */
class ClawbotClient {
  constructor(apiUrl, apiKey) {
    this.apiUrl = apiUrl;
    this.apiKey = apiKey;
  }

  getHeaders() {
    const headers = {
      "Content-Type": "application/json",
    };
    if (this.apiKey) {
      headers["Authorization"] = `Bearer ${this.apiKey}`;
    }
    return headers;
  }

  async sendMessage(chatId, text, options = {}) {
    const payload = {
      chat_id: chatId,
      text: text,
      parse_mode: "Markdown",
      ...options,
    };

    const response = await fetch(`${this.apiUrl}/api/v1/messages`, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error(`Clawbot API error: ${response.status}`);
    }

    return response.json();
  }

  async sendPhoto(chatId, photoUrl, caption) {
    const payload = {
      chat_id: chatId,
      photo: photoUrl,
      caption: caption,
    };

    const response = await fetch(`${this.apiUrl}/api/v1/photos`, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error(`Clawbot API error: ${response.status}`);
    }

    return response.json();
  }

  async getChatInfo(chatId) {
    const response = await fetch(
      `${this.apiUrl}/api/v1/chats/${encodeURIComponent(chatId)}`,
      {
        headers: this.getHeaders(),
      }
    );

    if (!response.ok) {
      throw new Error(`Clawbot API error: ${response.status}`);
    }

    return response.json();
  }

  async registerWebhook(webhookUrl) {
    const payload = {
      webhook_url: webhookUrl,
      secret: CLAWBOT_WEBHOOK_SECRET,
      events: ["message", "callback_query"],
    };

    const response = await fetch(`${this.apiUrl}/api/v1/webhook`, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error(`Clawbot webhook registration failed: ${response.status}`);
    }

    return response.json();
  }
}

const clawbot = new ClawbotClient(CLAWBOT_API_URL, CLAWBOT_API_KEY);

/**
 * Telegram Bot Client
 * Handles direct communication with Telegram Bot API
 */
class TelegramBotClient {
  constructor(botToken) {
    this.botToken = botToken;
    this.apiUrl = `https://api.telegram.org/bot${botToken}`;
  }

  async sendMessage(chatId, text, options = {}) {
    const payload = {
      chat_id: chatId,
      text: text,
      parse_mode: "Markdown",
      ...options,
    };

    // Remove buttons for basic implementation
    delete payload.buttons;

    const response = await fetch(`${this.apiUrl}/sendMessage`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(`Telegram API error: ${JSON.stringify(error)}`);
    }

    return response.json();
  }

  async setWebhook(webhookUrl, secret = "") {
    const payload = {
      url: webhookUrl,
    };

    if (secret) {
      payload.secret_token = secret;
    }

    const response = await fetch(`${this.apiUrl}/setWebhook`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error(`Telegram webhook setup failed: ${response.status}`);
    }

    return response.json();
  }

  async getMe() {
    const response = await fetch(`${this.apiUrl}/getMe`);
    if (!response.ok) {
      throw new Error(`Failed to get bot info: ${response.status}`);
    }
    return response.json();
  }

  async getUpdates(offset = 0, timeout = 30) {
    const params = new URLSearchParams({
      offset: offset.toString(),
      timeout: timeout.toString(),
    });
    const response = await fetch(`${this.apiUrl}/getUpdates?${params}`);
    if (!response.ok) {
      throw new Error(`Failed to get updates: ${response.status}`);
    }
    return response.json();
  }

  async deleteWebhook() {
    const response = await fetch(`${this.apiUrl}/deleteWebhook`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    });
    if (!response.ok) {
      throw new Error(`Failed to delete webhook: ${response.status}`);
    }
    return response.json();
  }

  isAllowedUser(userId) {
    if (TELEGRAM_ALLOWED_USER_IDS.length === 0) {
      return true; // No restrictions configured
    }
    return TELEGRAM_ALLOWED_USER_IDS.includes(String(userId));
  }
}

const telegram = USE_TELEGRAM_DIRECT ? new TelegramBotClient(TELEGRAM_BOT_TOKEN) : null;

/**
 * LM Studio Client
 * Handles AI chat responses using local LLM
 */
class LMStudioClient {
  constructor(baseUrl, model, token) {
    this.baseUrl = baseUrl;
    this.model = model;
    this.token = token;
    this.conversationHistory = new Map(); // chatId -> history array
  }

  async chat(message, chatId = "default", systemContext = "") {
    const headers = {
      "Content-Type": "application/json",
    };
    if (this.token) {
      headers["Authorization"] = `Bearer ${this.token}`;
    }

    // Get or create conversation history for this chat
    if (!this.conversationHistory.has(chatId)) {
      this.conversationHistory.set(chatId, []);
    }
    const history = this.conversationHistory.get(chatId);

    // Build system prompt with AmmanGate context
    const systemPrompt = `You are AmmanGate Assistant, a friendly and intelligent AI security assistant for the AmmanGate network security system.

🛡️ ABOUT AMMANGATE:
AmmanGate is a comprehensive network security monitoring and parental control system that runs locally.

📊 KEY INFORMATION YOU CAN ACCESS:
- System status (uptime, memory, CPU, sensors)
- Network devices (all connected devices with risk scores)
- Security alerts (active threats and warnings)
- Recent events (network activity, detections)
- Parental controls (domain filters, DNS settings)

🔧 AVAILABLE ACTIONS (when appropriate):
- Block/unblock domains
- View parental control filters
- Check device details
- Analyze security events

💬 YOUR PERSONALITY:
- Be conversational and friendly, like a helpful security expert
- Provide analysis and insights, not just commands
- Explain technical concepts in simple terms
- Be proactive - suggest relevant actions
- Use Indonesian or English based on user's language
- Keep responses concise (under 300 words usually)
- Use emojis to make responses more engaging

🎯 RESPONSE GUIDELINES:
1. Answer questions thoughtfully with real analysis
2. If the user asks for status/devices/alerts, actually fetch and show the data
3. For complex queries, break down your answer
4. If you need more information, ask clarifying questions
5. When suggesting actions, explain why they matter
6. Be conversational - don't sound robotic

${systemContext}`;

    // Build messages array
    const messages = [
      { role: "system", content: systemPrompt },
    ];

    // Add recent conversation history (last 10 messages)
    const recentHistory = history.slice(-10);
    for (const msg of recentHistory) {
      messages.push(msg);
    }

    // Add current user message
    messages.push({ role: "user", content: message });

    try {
      const response = await fetch(`${this.baseUrl}/chat/completions`, {
        method: "POST",
        headers,
        body: JSON.stringify({
          model: this.model,
          messages,
          temperature: 0.8,
          max_tokens: 800,
        }),
      });

      if (!response.ok) {
        throw new Error(`LM Studio error: ${response.status}`);
      }

      const data = await response.json();
      const aiResponse = data.choices?.[0]?.message?.content || "Maaf, saya tidak dapat memberikan respons.";

      // Save to history
      history.push({ role: "user", content: message });
      history.push({ role: "assistant", content: aiResponse });

      // Keep history manageable (max 20 messages)
      if (history.length > 20) {
        history.splice(0, history.length - 20);
      }

      return aiResponse;
    } catch (error) {
      console.error("LM Studio error:", error.message);
      throw error;
    }
  }

  clearHistory(chatId) {
    this.conversationHistory.delete(chatId);
  }
}

const lmStudio = new LMStudioClient(LM_STUDIO_URL, LM_STUDIO_MODEL, LM_STUDIO_TOKEN);

/**
 * Core API client
 */
class CoreClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
  }

  async get(path) {
    const response = await fetch(`${this.baseUrl}/v1${path}`);
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
  }

  async post(path, data) {
    const response = await fetch(`${this.baseUrl}/v1${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
  }

  async put(path, data) {
    const response = await fetch(`${this.baseUrl}/v1${path}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
  }

  async delete(path) {
    const response = await fetch(`${this.baseUrl}/v1${path}`, {
      method: "DELETE",
    });
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
  }

  async getSystemStatus() {
    return this.get("/system/status");
  }

  async getDevices(params = {}) {
    const query = new URLSearchParams(params);
    return this.get(`/devices?${query}`);
  }

  async getDevice(id) {
    return this.get(`/devices/${id}`);
  }

  async getActiveAlerts() {
    return this.get("/alerts/active");
  }

  async getEvents(params = {}) {
    const query = new URLSearchParams(params);
    return this.get(`/events?${query}`);
  }

  async requestApproval(actionData) {
    return this.post("/actions/request-approval", actionData);
  }

  async approveAction(approvalId, pin) {
    return this.post("/actions/approve", { approval_id: approvalId, pin });
  }

  async getPendingActions() {
    return this.get("/actions/pending");
  }

  async explainEvents(data) {
    return this.post("/explain", data);
  }
}

const core = new CoreClient(CORE_API_URL);

/**
 * Command parser with enhanced features
 */
class CommandParser {
  constructor() {
    this.commands = {
      // Status commands
      "status": this.handleStatus.bind(this),
      "health": this.handleHealth.bind(this),
      "version": this.handleVersion.bind(this),

      // Device commands
      "devices": this.handleDevices.bind(this),
      "device": this.handleDeviceDetail.bind(this),
      "search": this.handleSearchDevices.bind(this),
      "find": this.handleSearchDevices.bind(this),

      // Alert commands
      "alerts": this.handleAlerts.bind(this),
      "alert": this.handleAlertDetail.bind(this),
      "dismiss": this.handleDismissAlert.bind(this),
      "why": this.handleWhy.bind(this),

      // Event commands
      "events": this.handleEvents.bind(this),
      "history": this.handleEvents.bind(this),
      "recent": this.handleRecentEvents.bind(this),

      // Action commands
      "quarantine": this.handleQuarantine.bind(this),
      "unquarantine": this.handleUnquarantine.bind(this),
      "block": this.handleBlock.bind(this),
      "unblock": this.handleUnblock.bind(this),
      "pin": this.handlePin.bind(this),

      // Quick actions
      "scan": this.handleScan.bind(this),
      "emergency": this.handleEmergency.bind(this),
      "safe": this.handleSafeMode.bind(this),

      // Parental Control commands
      "filters": this.handleFilters.bind(this),
      "filter": this.handleFilterDetail.bind(this),
      "block": this.handleBlockDomain.bind(this),
      "unblock": this.handleUnblockDomain.bind(this),
      "parental": this.handleParental.bind(this),

      // MAC-based Device Blocking for Parental Control
      "blockmac": this.handleBlockMAC.bind(this),
      "unblockmac": this.handleUnblockMAC.bind(this),
      "blocked": this.handleListBlocked.bind(this),
      "child": this.handleChild.bind(this),
      "togglemac": this.handleToggleMAC.bind(this),

      // Management commands
      "pending": this.handlePending.bind(this),
      "cancel": this.handleCancel.bind(this),
      "approve": this.handleApprovePending.bind(this),

      // Utility commands
      "help": this.handleHelp.bind(this),
      "commands": this.handleHelp.bind(this),
      "about": this.handleAbout.bind(this),
    };
  }

  async parse(message, chatId, userId) {
    const text = message.toLowerCase().trim();
    const parts = text.split(/\s+/);

    const command = parts[0];
    const args = parts.slice(1);

    const handler = this.commands[command];
    if (!handler) {
      return await this.unknownCommand(message, chatId, userId);
    }

    return await handler(args, chatId, userId);
  }

  // ============================================================================
  // STATUS COMMANDS
  // ============================================================================

  async handleStatus(args, chatId, userId) {
    try {
      const status = await core.getSystemStatus();
      const alerts = await core.getActiveAlerts();

      // Provide analysis, not just data
      let analysis = "Sistem berjalan normal. ";

      if (alerts.items?.length > 0) {
        analysis = `⚠️ Perhatian: Terdapat ${alerts.items.length} alert aktif yang perlu diperiksa. `;
      }

      if (status.uptime_sec < 300) {
        analysis += "Sistem baru saja dimulai. ";
      } else if (status.uptime_sec > 86400) {
        analysis += "Sistem telah berjalan lebih dari 24 jam - pertimbangkan untuk restart. ";
      }

      if (status.mem_used_mb > 500) {
        analysis += "Penggunaan memori cukup tinggi. ";
      }

      const sensorStatus = Object.entries(status.sensors || {})
        .map(([name, active]) => `${active ? '🟢' : '🔴'} ${name}`)
        .join(' | ');

      return {
        text: `📊 *Status AmmanGate*\n\n` +
              `📈 Uptime: ${formatUptime(status.uptime_sec)}\n` +
              `💾 Memory: ${status.mem_used_mb} MB / 1024 MB\n` +
              `🖥️  CPU: ${(status.cpu_load * 100).toFixed(1)}%\n` +
              `🚨 Alerts: ${alerts.items?.length || 0} aktif\n` +
              `📡 Sensors: ${sensorStatus}\n\n` +
              `💡 ${analysis}\n\n` +
              `_Perlu detail? Ketik: alerts_`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleHealth(args, chatId, userId) {
    try {
      const health = await core.get("/health");
      return {
        text: `✅ *System Healthy*\n\n` +
              `Version: ${health.version}\n` +
              `Status: ${health.ok ? 'OK' : 'Degraded'}\n` +
              `Timestamp: ${health.ts}`,
      };
    } catch (error) {
      return { text: `❌ System unhealthy: ${error.message}` };
    }
  }

  async handleVersion(args, chatId, userId) {
    return {
      text: `📋 *AmmanGate Versions*\n\n` +
            `Core: ${await this.getComponentVersion('core')}\n` +
            `Gateway: 1.0.0\n` +
            `Clawbot: Integrated`,
    };
  }

  async getComponentVersion(component) {
    try {
      const health = await core.get("/health");
      return health.version || "unknown";
    } catch {
      return "unknown";
    }
  }

  // ============================================================================
  // DEVICE COMMANDS
  // ============================================================================

  async handleDevices(args, chatId, userId) {
    try {
      const limit = parseInt(args[0]) || 10;
      const result = await core.getDevices({ limit });

      if (result.items.length === 0) {
        return { text: "📱 No devices found" };
      }

      let text = `📱 *Devices (${result.items.length})*\n\n`;
      for (const device of result.items.slice(0, 10)) {
        const riskEmoji = getRiskEmoji(device.risk_score);
        text += `${riskEmoji} *${device.hostname || device.vendor || "Unknown"}*\n`;
        text += `   IP: ${device.ip || "No IP"}\n`;
        text += `   MAC: ${device.mac}\n`;
        text += `   Risk: ${device.risk_score}/100\n`;
        text += `   Last seen: ${formatDate(device.last_seen)}\n\n`;
      }

      if (result.items.length > 10) {
        text += `_...and ${result.items.length - 10} more devices_`;
      }

      return { text };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleDeviceDetail(args, chatId, userId) {
    if (args.length === 0) {
      return { text: "Usage: `device <device_id>`" };
    }

    try {
      const deviceId = args[0];
      const device = await core.getDevice(deviceId);

      if (!device) {
        return { text: `❌ Device not found: ${deviceId}` };
      }

      return {
        text: `📱 *Device Details*\n\n` +
              `Hostname: ${device.hostname || "Unknown"}\n` +
              `IP: ${device.ip || "None"}\n` +
              `MAC: ${device.mac}\n` +
              `Vendor: ${device.vendor || "Unknown"}\n` +
              `Type: ${device.type_guess || "Unknown"}\n` +
              `Risk Score: ${device.risk_score}/100 ${getRiskEmoji(device.risk_score)}\n` +
              `First Seen: ${formatDate(device.first_seen)}\n` +
              `Last Seen: ${formatDate(device.last_seen)}\n` +
              `Tags: ${device.tags?.join(', ') || 'None'}\n` +
              `Notes: ${device.notes || 'None'}`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleSearchDevices(args, chatId, userId) {
    if (args.length === 0) {
      return { text: "Usage: `search <query>` - Search by hostname, IP, or MAC" };
    }

    try {
      const query = args.join(' ');
      const result = await core.getDevices({ q: query, limit: 20 });

      if (result.items.length === 0) {
        return { text: `🔍 No devices found matching "${query}"` };
      }

      let text = `🔍 *Search Results (${result.items.length})*\n\n`;
      for (const device of result.items) {
        text += `${getRiskEmoji(device.risk_score)} ${device.hostname || device.vendor || device.ip}\n`;
      }

      return { text };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  // ============================================================================
  // ALERT COMMANDS
  // ============================================================================

  async handleAlerts(args, chatId, userId) {
    try {
      const result = await core.getActiveAlerts();

      if (!result.items || result.items.length === 0) {
        return {
          text: `✅ *No Active Alerts*\n\n` +
                `Your network is secure! No threats detected.`,
        };
      }

      let text = `🚨 *Active Alerts (${result.items.length})*\n\n`;
      for (const alert of result.items.slice(0, 10)) {
        const severityEmoji = getSeverityEmoji(alert.severity);
        text += `${severityEmoji} *${alert.title}*\n`;
        text += `   ID: \`${alert.id}\`\n`;
        text += `   Severity: ${alert.severity}/10\n`;
        text += `   Status: ${alert.status}\n`;
        text += `   Time: ${formatDate(alert.ts)}\n\n`;
      }

      return {
        text: text + `\n_Reply with \`dismiss <alert_id>\` to dismiss an alert_`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleAlertDetail(args, chatId, userId) {
    // This would be implemented when API supports individual alert retrieval
    return { text: "📋 Alert detail: Use `alerts` to list all alerts" };
  }

  async handleDismissAlert(args, chatId, userId) {
    if (args.length === 0) {
      return { text: "Usage: `dismiss <alert_id>`" };
    }
    // This would be implemented when API supports alert dismissal
    return { text: "⚠️ Alert dismissal requires dashboard access (coming soon)" };
  }

  async handleWhy(args, chatId, userId) {
    if (args.length === 0) {
      return { text: "Usage: `why <alert_id>` - Get AI explanation for an alert" };
    }

    const alertId = args[0];
    try {
      const explanation = await core.explainEvents({
        alert_id: alertId,
      });

      return {
        text: `📋 *Analysis for Alert ${alertId}*\n\n` +
              `🔍 *Suspected Cause:*\n${explanation.suspected_cause}\n\n` +
              `📝 *Narrative:*\n${explanation.narrative}\n\n` +
              `✅ *Recommended Actions:*\n${explanation.recommended_actions?.map(a => `• ${a}`).join('\n') || 'None'}\n\n` +
              `🎯 Confidence: ${(explanation.confidence * 100).toFixed(0)}%`,
      };
    } catch (error) {
      return {
        text: `📋 *Analysis for Alert ${alertId}*\n\n` +
              `_AI explanation will be available in v1.0 with local LLM integration._\n\n` +
              `For now, check the dashboard for detailed event information.`,
      };
    }
  }

  // ============================================================================
  // EVENT COMMANDS
  // ============================================================================

  async handleEvents(args, chatId, userId) {
    try {
      const limit = parseInt(args[0]) || 20;
      const result = await core.getEvents({ limit });

      if (!result.items || result.items.length === 0) {
        return { text: "📭 No events found" };
      }

      let text = `📊 *Recent Events (${result.items.length})*\n\n`;
      for (const event of result.items.slice(0, 15)) {
        const severityEmoji = getSeverityEmoji(event.severity);
        text += `${severityEmoji} *${event.summary || event.category}*\n`;
        text += `   ${formatDate(event.ts)}\n\n`;
      }

      return { text };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleRecentEvents(args, chatId, userId) {
    return await this.handleEvents(['10'], chatId, userId);
  }

  // ============================================================================
  // ACTION COMMANDS
  // ============================================================================

  async handleQuarantine(args, chatId, userId) {
    if (args.length === 0) {
      return {
        text: "🚫 *Quarantine Device*\n\n" +
              "Usage: `quarantine <device_ip_or_id>`\n\n" +
              "This will isolate the device from the network.",
      };
    }

    const target = args[0];
    return await this.requestAction(
      "quarantine_device",
      { device: target },
      chatId,
      userId,
      `🚫 *Quarantine Device: ${target}*\n\nThis will block all network traffic from this device.`
    );
  }

  async handleUnquarantine(args, chatId, userId) {
    if (args.length === 0) {
      return { text: "Usage: `unquarantine <device_ip_or_id>`" };
    }

    const target = args[0];
    return await this.requestAction(
      "unquarantine_device",
      { device: target },
      chatId,
      userId,
      `✅ *Restore Device: ${target}*`
    );
  }

  async handleBlock(args, chatId, userId) {
    if (args.length < 2) {
      return {
        text: "🚫 *Block Traffic*\n\n" +
              "Usage: `block <ip|domain> <value>`\n\n" +
              "Examples:\n" +
              "• `block ip 192.168.1.100`\n" +
              "• `block domain malicious.com`",
      };
    }

    const type = args[0];
    const value = args[1];

    if (type === "ip") {
      return await this.requestAction("block_ip", { ip: value }, chatId, userId,
        `🚫 *Block IP: ${value}*`);
    } else if (type === "domain") {
      // For domain blocking, use parental control filter
      return await this.handleBlockDomain([value], chatId, userId);
    }

    return { text: "Usage: `block <ip|domain> <value>`" };
  }

  // ============================================================================
  // PARENTAL CONTROL COMMANDS
  // ============================================================================

  async handleFilters(args, chatId, userId) {
    try {
      const result = await core.get("/filters");
      const filters = result.items || [];

      if (filters.length === 0) {
        return {
          text: `🔒 *Parental Control Filters*\n\nNo filters configured yet.\n\nUse:\n• \`filter add adult\` to block adult content\n• \`filter add gambling\` to block gambling`,
        };
      }

      let text = `🔒 *Parental Control Filters (${filters.length})*\n\n`;
      for (const filter of filters.slice(0, 10)) {
        const status = filter.enabled ? "✅" : "❌";
        const typeEmoji = filter.type === "domain" ? "🌐" : "📁";
        text += `${status} ${typeEmoji} *${filter.name}*\n`;
        text += `   Pattern: \`${filter.pattern}\`\n\n`;
      }

      if (filters.length > 10) {
        text += `...and ${filters.length - 10} more filters\n\n`;
      }

      text += `Use \`filter <id>\` for details`;
      return { text };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleFilterDetail(args, chatId, userId) {
    if (args.length === 0) {
      return {
        text: "Usage: `filter <filter_id>` - Get filter details\n\n" +
              "Use \`filters\` to list all filter IDs",
      };
    }

    try {
      // Since we don't have individual filter endpoint, show from list
      const result = await core.get("/filters");
      const filter = result.items?.find(f => f.id.startsWith(args[0]));

      if (!filter) {
        return { text: `❌ Filter not found: ${args[0]}` };
      }

      return {
        text: `🔒 *Filter Details*\n\n` +
              `ID: \`${filter.id}\`\n` +
              `Name: *${filter.name}*\n` +
              `Type: ${filter.type}\n` +
              `Pattern: \`${filter.pattern}\`\n` +
              `Status: ${filter.enabled ? "Enabled ✅" : "Disabled ❌"}`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleBlockDomain(args, chatId, userId) {
    if (args.length === 0) {
      return {
        text: "🚫 *Block Domain*\n\n" +
              "Usage: `block <domain>`\n\n" +
              "Examples:\n" +
              "• `block example.com`\n" +
              "• `filter add gambling` (block category)",
      };
    }

    const domain = args[0];

    try {
      await core.post("/filters", {
        name: `Blocked: ${domain}`,
        type: "domain",
        pattern: domain,
      });

      return {
        text: `✅ *Domain Blocked*\n\n` +
              `Domain: ${domain}\n` +
              `Type: Domain Filter\n\n` +
              `Use \`filters\` to view all filters`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleUnblockDomain(args, chatId, userId) {
    if (args.length === 0) {
      return {
        text: "🔓 *Unblock Domain*\n\n" +
              "Usage: `unblock <domain>`\n\n" +
              "Example:\n" +
              "• `unblock example.com`",
      };
    }

    const domain = args[0];

    try {
      // Find and remove the filter
      const result = await core.get("/filters");
      const filter = result.items?.find(f => f.pattern === domain);

      if (!filter) {
        return { text: `❌ Domain not found in filters: ${domain}` };
      }

      await core.fetch(`/filters/${filter.id}`, { method: "DELETE" });

      return {
        text: `✅ *Domain Unblocked*\n\n` +
              `Domain: ${domain}\n\n` +
              `The domain is now allowed.`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleParental(args, chatId, userId) {
    const subCommand = args[0];

    if (!subCommand || subCommand === "status") {
      return await this.handleParentalStatus(chatId, userId);
    }

    switch (subCommand) {
      case "status":
        return await this.handleParentalStatus(chatId, userId);
      case "enable":
        return await this.handleParentalEnable(args.slice(1), chatId, userId);
      case "disable":
        return await this.handleParentalDisable(args.slice(1), chatId, userId);
      default:
        return {
          text: `👨‍👩‍👧‍👦 *Parental Control*\n\n` +
                `Subcommands:\n` +
                `• status - Show parental control status\n` +
                `• enable <device_id> - Enable for device\n` +
                `• disable <device_id> - Disable for device`,
        };
    }
  }

  async handleParentalStatus(chatId, userId) {
    try {
      const statusResult = await core.get("/system/network");
      const filtersResult = await core.get("/filters");

      const enabledCount = filtersResult.items?.filter(f => f.enabled).length || 0;
      const dnsRunning = statusResult.dns_running || false;

      return {
        text: `👨‍👩‍👧‍👦 *Parental Control Status*\n\n` +
              `DNS Server: ${dnsRunning ? "✅ Running" : "❌ Stopped"}\n` +
              `Active Filters: ${enabledCount}\n` +
              `Total Filters: ${filtersResult.items?.length || 0}\n\n` +
              `Server IP: ${statusResult.primary_ip}\n` +
              `DNS Port: ${statusResult.dns_port}\n\n` +
              `${dnsRunning ? "✅ Parental Control is active" : "⚠️ DNS server is stopped"}`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleParentalEnable(args, chatId, userId) {
    if (args.length === 0) {
      return { text: "Usage: `parental enable <device_id>`" };
    }

    return {
      text: `✅ *Parental Control Enabled*\n\n` +
            `Device: ${args[0]}\n\n` +
            `Note: Make sure your router is configured to use AmmanGate DNS server (${env("BG_ADDR", "127.0.0.1")}:${env("BG_DNS_ADDR", ":53")})`,
    };
  }

  async handleParentalDisable(args, chatId, userId) {
    if (args.length === 0) {
      return { text: "Usage: `parental disable <device_id>`" };
    }

    return {
      text: `⚠️ *Parental Control Disabled*\n\n` +
            `Device: ${args[0]}\n\n` +
            `DNS filtering will be disabled for this device.`,
    };
  }

  async handleUnblock(args, chatId, userId) {
    if (args.length < 2) {
      return { text: "Usage: `unblock <ip|domain> <value>`" };
    }

    const type = args[0];
    const value = args[1];

    if (type === "ip") {
      return await this.requestAction("unblock_ip", { ip: value }, chatId, userId);
    } else if (type === "domain") {
      return await this.requestAction("unblock_domain", { domain: value }, chatId, userId);
    }

    return { text: "Usage: `unblock <ip|domain> <value>`" };
  }

  async handlePin(args, chatId, userId) {
    const pending = pendingApprovals.get(chatId);
    if (!pending) {
      return { text: "❌ No pending action. Send a command first." };
    }

    if (new Date() > pending.expiresAt) {
      pendingApprovals.delete(chatId);
      return { text: "❌ Approval expired. Please try again." };
    }

    const providedPin = args.join(''); // Handle PIN with or without spaces
    if (providedPin !== ACTION_PIN) {
      return { text: `❌ Invalid PIN. Use: \`PIN ${ACTION_PIN}\`` };
    }

    try {
      const result = await core.approveAction(pending.approvalId, ACTION_PIN);
      pendingApprovals.delete(chatId);

      return {
        text: `✅ *Action Executed*\n\n` +
              `${result.detail}\n` +
              `Action ID: \`${result.action_id}\``,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  // ============================================================================
  // QUICK ACTIONS
  // ============================================================================

  async handleScan(args, chatId, userId) {
    return {
      text: `🔍 *Network Scan*\n\n` +
            `Triggering network scan...\n\n` +
            `_This may take a few minutes. Results will appear in the device list._`,
    };
  }

  async handleEmergency(args, chatId, userId) {
    return {
      text: `🚨 *Emergency Mode*\n\n` +
            `⚠️ This would enable enhanced monitoring.\n\n` +
            `_Feature coming in v0.2_`,
    };
  }

  async handleSafeMode(args, chatId, userId) {
    return this.requestAction(
      "unblock_ip",
      { ip: "0.0.0.0/0" },
      chatId,
      userId,
      `🛡️ *Safe Mode*\n\nThis will restore all network rules to default.`
    );
  }

  // ============================================================================
  // MANAGEMENT COMMANDS
  // ============================================================================

  async handlePending(args, chatId, userId) {
    try {
      const result = await core.getPendingActions();

      if (!result.items || result.items.length === 0) {
        return { text: "✅ No pending actions" };
      }

      let text = `⏳ *Pending Actions (${result.items.length})*\n\n`;
      for (const action of result.items) {
        text += `• \`${action.id}\`\n`;
        text += `  ${action.action_type} on ${action.target}\n`;
        text += `  Requested by: ${action.requested_by}\n\n`;
      }

      return { text };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleCancel(args, chatId, userId) {
    const pending = pendingApprovals.get(chatId);
    if (!pending) {
      return { text: "❌ No pending action to cancel." };
    }

    pendingApprovals.delete(chatId);
    return { text: "✅ Pending action cancelled." };
  }

  async handleApprovePending(args, chatId, userId) {
    return await this.handlePin([ACTION_PIN], chatId, userId);
  }

  // ============================================================================
  // UTILITY COMMANDS
  // ============================================================================

  handleHelp(args, chatId, userId) {
    return {
      text: `🛡️ *AmmanGate Commands*\n\n` +
            `*📊 Status & Info:*\n` +
            `status - Show system status\n` +
            `health - System health check\n` +
            `version - Version information\n\n` +
            `*📱 Devices:*\n` +
            `devices [n] - List devices (max n)\n` +
            `device <id> - Device details\n` +
            `search <q> - Search devices\n` +
            `find <q> - Find devices\n\n` +
            `*🚨 Alerts:*\n` +
            `alerts - Show active alerts\n` +
            `why <id> - Explain alert\n` +
            `dismiss <id> - Dismiss alert\n\n` +
            `*📊 Events:*\n` +
            `events [n] - Show events\n` +
            `history [n] - Event history\n` +
            `recent - Recent events\n\n` +
            `*🔒 Security Actions (require PIN):*\n` +
            `quarantine <device> - Isolate device\n` +
            `unquarantine <device> - Restore device\n` +
            `block ip <ip> - Block IP\n` +
            `block domain <dom> - Block domain\n` +
            `unblock ip <ip> - Unblock IP\n` +
            `unblock domain <dom> - Unblock domain\n\n` +
            `*⚡ Quick Actions:*\n` +
            `scan - Trigger network scan\n` +
            `pending - Show pending actions\n` +
            `cancel - Cancel pending action\n` +
            `PIN <code> - Approve action\n\n` +
            `*❓ Help:*\n` +
            `help - Show this message\n` +
            `about - About AmmanGate`,
    };
  }

  // ============================================================================
  // MAC-BASED DEVICE BLOCKING FOR PARENTAL CONTROL
  // ============================================================================

  async handleBlockMAC(args, chatId, userId) {
    // Usage: blockmac <mac_address> [device_name] [reason]
    if (args.length < 1) {
      return {
        text: `📱 *Block Device by MAC*\n\n` +
              `Usage: \`blockmac <mac_address> [device_name] [reason]\`\n\n` +
              `Examples:\n` +
              `• blockmac AA:BB:CC:DD:EE:FF\n` +
              `• blockmac AA:BB:CC:DD:EE:FF "iPhone Anak"\n` +
              `• blockmac AA:BB:CC:DD:EE:FF "Laptop" "Belum selesai PR"\n\n` +
              `MAC format: XX:XX:XX:XX:XX:XX or XX-XX-XX-XX-XX-XX`,
      };
    }

    const mac = args[0];
    const deviceName = args[1] || "Unknown Device";
    const reason = args.slice(2).join(" ") || "Blocked by parent";

    try {
      const response = await core.post("/block-device", {
        mac_address: mac,
        device_name: deviceName,
        block_reason: reason,
        blocked_by: `telegram:${userId}`,
      });

      return {
        text: `🚫 *Device Blocked*\n\n` +
              `MAC: ${response.mac_address}\n` +
              `Name: ${response.device_name}\n` +
              `Reason: ${response.block_reason}\n\n` +
              `✅ Device has been added to firewall blocklist.\n` +
              `Use \`unblockmac ${mac}\` to unblock.`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleUnblockMAC(args, chatId, userId) {
    // Usage: unblockmac <mac_address>
    if (args.length < 1) {
      return {
        text: `📱 *Unblock Device by MAC*\n\n` +
              `Usage: \`unblockmac <mac_address>\`\n\n` +
              `Example: unblockmac AA:BB:CC:DD:EE:FF`,
      };
    }

    const mac = args[0];

    try {
      await core.delete(`/block-device/${encodeURIComponent(mac)}`);

      return {
        text: `✅ *Device Unblocked*\n\n` +
              `MAC: ${mac}\n\n` +
              `Device has been removed from firewall blocklist.\n` +
              `Internet access is now restored.`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleListBlocked(args, chatId, userId) {
    try {
      const response = await core.get("/blocked-devices");
      const devices = response.items || [];

      if (devices.length === 0) {
        return {
          text: `📱 *Blocked Devices*\n\n` +
                `No devices are currently blocked.\n\n` +
                `Use \`blockmac <mac>\` to block a device.`,
        };
      }

      let text = `📱 *Blocked Devices* (${devices.length})\n\n`;

      devices.forEach((device, index) => {
        const status = device.blocked ? "🔴 BLOCKED" : "🟢 UNBLOCKED";
        text += `${index + 1}. ${status}\n`;
        text += `   MAC: ${device.mac_address}\n`;
        if (device.device_name) {
          text += `   Name: ${device.device_name}\n`;
        }
        if (device.block_reason) {
          text += `   Reason: ${device.block_reason}\n`;
        }
        text += `   Since: ${new Date(device.blocked_at).toLocaleString('id-ID')}\n\n`;
      });

      text += `Use \`unblockmac <mac>\` to unblock a device.`;

      return { text };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleToggleMAC(args, chatId, userId) {
    // Usage: togglemac <mac_address>
    if (args.length < 1) {
      return {
        text: `📱 *Toggle Device Block*\n\n` +
              `Usage: \`togglemac <mac_address>\`\n\n` +
              `Toggles the blocked status of a device.\n` +
              `Useful for temporarily allowing/blocking internet access.`,
      };
    }

    const mac = args[0];

    try {
      const response = await core.put(`/blocked-device/${encodeURIComponent(mac)}/toggle`);

      const action = response.blocked ? "blocked" : "unblocked";

      return {
        text: `🔄 *Device ${action === 'blocked' ? 'Blocked' : 'Unblocked'}*\n\n` +
              `MAC: ${response.mac_address}\n` +
              `Name: ${response.device_name || 'Unknown'}\n` +
              `Status: ${response.blocked ? '🔴 BLOCKED' : '🟢 UNBLOCKED'}\n\n` +
              `Use \`togglemac ${mac}\` again to reverse.`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleChild(args, chatId, userId) {
    // Usage: child <subcommand> ...
    const subCommand = args[0];

    if (!subCommand) {
      return {
        text: `👨‍👩‍👧‍👦 *Child Device Control*\n\n` +
              `Manage your children's internet access easily.\n\n` +
              `Commands:\n` +
              `• child add <mac> <name> - Add child's device\n` +
              `• child list - List all child devices\n` +
              `• child enable <mac> - Allow internet access\n` +
              `• child disable <mac> - Block internet access\n\n` +
              `Examples:\n` +
              `• child add AA:BB:CC:DD:EE:FF "Ahmad HP"\n` +
              `• child disable AA:BB:CC:DD:EE:FF\n` +
              `• child enable AA:BB:CC:DD:EE:FF`,
      };
    }

    switch (subCommand) {
      case "add":
        return await this.handleChildAdd(args.slice(1), chatId, userId);
      case "list":
        return await this.handleChildList(chatId, userId);
      case "enable":
        return await this.handleChildEnable(args.slice(1), chatId, userId);
      case "disable":
        return await this.handleChildDisable(args.slice(1), chatId, userId);
      default:
        return {
          text: `❓ Unknown child command.\n\n` +
                `Available: add, list, enable, disable\n\n` +
                `Use \`child\` without arguments for help.`,
        };
    }
  }

  async handleChildAdd(args, chatId, userId) {
    if (args.length < 2) {
      return {
        text: `❌ Usage: \`child add <mac_address> <child_name>\`\n\n` +
              `Example: child add AA:BB:CC:DD:EE:FF "Ahmad HP"`,
      };
    }

    const mac = args[0];
    const name = args[1];

    try {
      const response = await core.post("/block-device", {
        mac_address: mac,
        device_name: name,
        block_reason: `Child device: ${name}`,
        blocked_by: `telegram:${userId}`,
      });

      return {
        text: `👶 *Child Device Added*\n\n` +
              `Name: ${name}\n` +
              `MAC: ${response.mac_address}\n\n` +
              `✅ Device is registered.\n\n` +
              `Use these commands to control access:\n` +
              `• child disable ${mac} - Block internet\n` +
              `• child enable ${mac} - Allow internet`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleChildList(chatId, userId) {
    try {
      const response = await core.get("/blocked-devices");
      const devices = response.items || [];

      const childDevices = devices.filter(d =>
        d.block_reason && d.block_reason.includes("Child device:")
      );

      if (childDevices.length === 0) {
        return {
          text: `👶 *Child Devices*\n\n` +
                `No child devices registered.\n\n` +
                `Use \`child add <mac> <name>\` to register a device.`,
        };
      }

      let text = `👶 *Child Devices* (${childDevices.length})\n\n`;

      childDevices.forEach((device, index) => {
        const status = device.blocked ? "🔴 No Internet" : "🟢 Internet ON";
        text += `${index + 1}. ${status}\n`;
        text += `   Name: ${device.device_name}\n`;
        text += `   MAC: ${device.mac_address}\n\n`;
      });

      text += `Use \`child enable/disable <mac>\` to control access.`;

      return { text };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleChildEnable(args, chatId, userId) {
    if (args.length < 1) {
      return {
        text: `❌ Usage: \`child enable <mac_address>\`\n\n` +
              `Allows internet access for the child's device.`,
      };
    }

    const mac = args[0];

    try {
      await core.delete(`/block-device/${encodeURIComponent(mac)}`);

      return {
        text: `🟢 *Internet Enabled*\n\n` +
              `MAC: ${mac}\n\n` +
              `✅ Your child can now access the internet.\n\n` +
              `Use \`child disable ${mac}\` to block again.`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleChildDisable(args, chatId, userId) {
    if (args.length < 1) {
      return {
        text: `❌ Usage: \`child disable <mac_address>\`\n\n` +
              `Blocks internet access for the child's device.`,
      };
    }

    const mac = args[0];

    try {
      const response = await core.post("/block-device", {
        mac_address: mac,
        block_reason: "Parental control - internet disabled",
        blocked_by: `telegram:${userId}`,
      });

      return {
        text: `🔴 *Internet Disabled*\n\n` +
              `MAC: ${mac}\n\n` +
              `✅ Your child's internet access has been blocked.\n\n` +
              `Use \`child enable ${mac}\` to allow again.`,
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  handleAbout(args, chatId, userId) {
    return {
      text: `🛡️ *AmmanGate - AI Home Cyber Bodyguard*\n\n` +
            `Local-only network security guardian for your home and small business.\n\n` +
            `*Key Features:*\n` +
            `• Device discovery & fingerprinting\n` +
            `• Real-time threat detection\n` +
            `• Firewall integration\n` +
            `• AI-powered chat control\n` +
            `• Privacy-first (all data local)\n\n` +
            `*Version:* 0.1.0 MVP\n` +
            `*License:* MIT\n\n` +
            `https://github.com/ammangate`,
    };
  }

  async unknownCommand(message, chatId, userId) {
    // Try to match partial commands first
    const cmd = message.toLowerCase().trim();
    const matches = Object.keys(this.commands).filter(k => k.startsWith(cmd));
    if (matches.length > 0 && matches.length <= 3) {
      return {
        text: `❓ Maksud Anda: ${matches.map(m => '`' + m + '`').join(', ')}?\n\nKetik salah satu perintah di atas untuk melanjutkan.`,
      };
    }

    // Use LM Studio for natural language understanding with actual system context
    try {
      // Build system context with actual data
      let contextData = "";

      // Determine what context to fetch based on keywords
      const lowerMsg = message.toLowerCase();

      if (lowerMsg.includes('status') || lowerMsg.includes(' sistem') || lowerMsg.includes(' kesehatan')) {
        try {
          const status = await core.getSystemStatus();
          const alerts = await core.getActiveAlerts();
          contextData = `Current System Status:
- Uptime: ${Math.floor(status.uptime_sec / 60)} minutes
- Memory: ${status.mem_used_mb} MB used
- CPU Load: ${(status.cpu_load * 100).toFixed(1)}%
- Active Alerts: ${alerts.items?.length || 0}
- Last Event: ${status.last_event_ts || "N/A"}`;
        } catch (e) {}
      }

      if (lowerMsg.includes('device') || lowerMsg.includes(' perangkat')) {
        try {
          const devices = await core.getDevices({ limit: 10 });
          const deviceCount = devices.items?.length || 0;
          contextData = `Network Devices: ${deviceCount} devices connected
- High risk devices: ${devices.items?.filter(d => d.risk_score > 50).length || 0}
- Medium risk devices: ${devices.items?.filter(d => d.risk_score > 20 && d.risk_score <= 50).length || 0}`;
        } catch (e) {}
      }

      if (lowerMsg.includes('alert') || lowerMsg.includes(' ancaman') || lowerMsg.includes(' bahaya')) {
        try {
          const alerts = await core.getActiveAlerts();
          const alertCount = alerts.items?.length || 0;
          contextData = `Security Alerts: ${alertCount} active alerts`;
          if (alertCount > 0) {
            contextData += `\nRecent alert: ${alerts.items?.[0]?.title || "None"}`;
          }
        } catch (e) {}
      }

      const fullContext = `${contextData}

User asked: "${message}"

Provide a helpful, conversational response in Indonesian or English (matching user's language). If they asked for status/devices/alerts, include the actual data above. Be concise but thorough. If action is needed, suggest specific commands.`;

      const aiResponse = await lmStudio.chat(message, chatId, fullContext);

      return {
        text: aiResponse,
      };
    } catch (error) {
      console.error("AI chat error:", error.message);

      // Fallback with helpful suggestions
      const suggestions = this.getSuggestions(message);
      return {
        text: `🤖 Maaf, saya sedang memproses permintaan Anda. ${suggestions}

_Gunakan \`help\` untuk semua perintah yang tersedia._`,
      };
    }
  }

  // Helper to get relevant suggestions based on message content
  getSuggestions(message) {
    const lower = message.toLowerCase();
    let suggestions = [];

    if (lower.includes('status') || lower.includes('kondisi')) {
      suggestions.push("\n\n💡 Ketik: `status` - untuk melihat status sistem");
    }
    if (lower.includes('device') || lower.includes('perangkat')) {
      suggestions.push("\n\n💡 Ketik: `devices` - untuk melihat daftar perangkat");
    }
    if (lower.includes('alert') || lower.includes('ancaman')) {
      suggestions.push("\n\n💡 Ketik: `alerts` - untuk melihat alert keamanan");
    }
    if (lower.includes('blokir') || lower.includes('block')) {
      suggestions.push("\n\n💡 Ketik: `block domain <nama>` - untuk blokir domain");
    }
    if (lower.includes('filter') || lower.includes('parental')) {
      suggestions.push("\n\n💡 Ketik: `filters` - untuk melihat filter parental control");
    }

    if (suggestions.length === 0) {
      suggestions = "\n\n💡 Ketik: `help` - untuk melihat semua perintah";
    }

    return suggestions.join('');
  }

  // ============================================================================
  // ACTION HELPERS
  // ============================================================================

  async requestAction(actionType, target, chatId, userId, description = null) {
    try {
      const result = await core.requestApproval({
        action_type: actionType,
        target: target,
        ttl_sec: 1800,
        requested_by: `clawbot:${userId}`,
      });

      // Store pending approval
      pendingApprovals.set(chatId, {
        approvalId: result.approval_id,
        actionId: result.action_id,
        expiresAt: new Date(result.expires_at),
        userId: userId,
      });

      const desc = description || `⚠️ *Action Pending Approval*\n\n${result.message}`;

      return {
        text: `${desc}\n\n` +
              `Reply with: \`PIN ${ACTION_PIN}\` to confirm.`,
        expectsPin: true,
        buttons: [
          { text: `✅ Approve`, callback_data: `approve:${result.approval_id}:${ACTION_PIN}` },
          { text: "❌ Cancel", callback_data: `cancel:${result.approval_id}` },
        ],
      };
    } catch (error) {
      return { text: `❌ Error: ${error.message}` };
    }
  }

  async handleCallback(callbackData, chatId, userId) {
    const [action, approvalId, ...rest] = callbackData.split(":");

    if (action === "approve") {
      const pin = rest[0];
      return await this.handlePin([pin], chatId, userId);
    } else if (action === "cancel") {
      pendingApprovals.delete(chatId);
      return { text: "✅ Action cancelled." };
    }

    return { text: "❓ Unknown action" };
  }
}

const parser = new CommandParser();

// Special handler for PIN responses
parser.commands["pin"] = parser.handlePin.bind(parser);

/**
 * Utility functions
 */
function formatUptime(seconds) {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);

  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${mins}m`;
  return `${mins}m`;
}

function formatDate(dateStr) {
  if (!dateStr) return "Unknown";
  const date = new Date(dateStr);
  const now = new Date();
  const diff = (now - date) / 1000; // seconds

  if (diff < 60) return "Just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return date.toLocaleDateString();
}

function getRiskEmoji(score) {
  if (score >= 80) return "🔴";
  if (score >= 60) return "🟠";
  if (score >= 40) return "🟡";
  return "🟢";
}

function getSeverityEmoji(severity) {
  if (severity >= 9) return "🔴";
  if (severity >= 7) return "🟠";
  if (severity >= 5) return "🟡";
  return "🔵";
}

/**
 * Webhook signature verification
 */
function verifyWebhookSignature(payload, signature) {
  if (!CLAWBOT_WEBHOOK_SECRET) {
    return true; // Skip verification if no secret configured
  }

  const crypto = require("crypto");
  const hmac = crypto.createHmac("sha256", CLAWBOT_WEBHOOK_SECRET);
  const digest = hmac.update(payload).digest("hex");

  return signature === digest;
}

/**
 * HTTP endpoints (for Clawbot integration)
 */

// Health check
app.get("/health", (req, res) => {
  res.json({
    ok: true,
    service: "claw-gateway",
    version: "1.0.0",
    integration: USE_TELEGRAM_DIRECT ? "telegram" : "clawbot",
    telegram_enabled: USE_TELEGRAM_DIRECT,
    allowed_users: TELEGRAM_ALLOWED_USER_IDS,
  });
});

// Register webhook with Clawbot
app.post("/webhook/register", async (req, res) => {
  try {
    const webhookUrl = req.body.webhook_url || `http://${req.headers.host}/webhook/clawbot`;
    const result = await clawbot.registerWebhook(webhookUrl);

    console.log("✅ Webhook registered with Clawbot:", result);
    res.json({ success: true, result });
  } catch (error) {
    console.error("Failed to register webhook:", error);
    res.status(500).json({ error: error.message });
  }
});

// Receive message from Clawbot
app.post("/webhook/clawbot", async (req, res) => {
  try {
    // Verify webhook signature if secret is configured
    const signature = req.headers["x-clawbot-signature"];
    if (CLAWBOT_WEBHOOK_SECRET && signature) {
      const payload = JSON.stringify(req.body);
      if (!verifyWebhookSignature(payload, signature)) {
        return res.status(401).json({ error: "Invalid signature" });
      }
    }

    const { event, data } = req.body;

    if (event === "message") {
      const { chat_id, message_id, from, text } = data;

      // Parse and handle command
      const response = await parser.parse(text, chat_id, from);

      // Send response back via Clawbot
      if (response.text) {
        await clawbot.sendMessage(chat_id, response.text, {
          reply_to_message_id: message_id,
          buttons: response.buttons,
        });
      }
    } else if (event === "callback_query") {
      const { chat_id, from, callback_id, data: callbackData } = data;

      // Handle callback (button press)
      const response = await parser.handleCallback(callbackData, chat_id, from);

      // Send response
      if (response.text) {
        await clawbot.sendMessage(chat_id, response.text);
      }

      // Answer callback query
      res.json({ callback_id, status: "answered" });
      return;
    }

    res.json({ success: true });
  } catch (error) {
    console.error("Error handling webhook:", error);
    res.status(500).json({ error: error.message });
  }
});

// ============================================================================
// TELEGRAM BOT WEBHOOK (Direct Telegram Integration)
// ============================================================================

// Telegram webhook endpoint
app.post("/webhook/telegram", async (req, res) => {
  try {
    if (!USE_TELEGRAM_DIRECT || !telegram) {
      return res.status(503).json({ error: "Telegram bot not configured" });
    }

    const update = req.body;
    console.log("Received Telegram update:", JSON.stringify(update, null, 2));

    // Handle message
    if (update.message) {
      const chat = update.message.chat;
      const from = update.message.from;
      const text = update.message.text;
      const messageId = update.message.message_id;

      // Check if user is allowed
      if (!telegram.isAllowedUser(from.id)) {
        console.log(`Unauthorized user ${from.id} (${from.username}) attempted to use bot`);
        return res.json({ ok: true }); // Still return 200 to prevent retries
      }

      // Parse and handle command
      const response = await parser.parse(text, String(chat.id), String(from.id));

      // Send response back via Telegram
      if (response && response.text) {
        await telegram.sendMessage(chat.id, response.text, {
          reply_to_message_id: messageId,
        });
      }
    }

    // Handle callback query (button presses)
    if (update.callback_query) {
      const callbackQuery = update.callback_query;
      const chat = callbackQuery.message.chat;
      const from = callbackQuery.from;
      const data = callbackQuery.data;
      const queryId = callbackQuery.id;

      // Check if user is allowed
      if (!telegram.isAllowedUser(from.id)) {
        // Answer callback query to remove loading state
        await fetch(`${telegram.apiUrl}/answerCallbackQuery`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            callback_query_id: queryId,
            text: "⚠️ You are not authorized to use this bot",
            show_alert: true,
          }),
        });
        return res.json({ ok: true });
      }

      // Handle callback
      const response = await parser.handleCallback(data, String(chat.id), String(from.id));

      // Answer callback query first
      await fetch(`${telegram.apiUrl}/answerCallbackQuery`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          callback_query_id: queryId,
        }),
      });

      // Send response
      if (response && response.text) {
        await telegram.sendMessage(chat.id, response.text);
      }
    }

    res.json({ ok: true });
  } catch (error) {
    console.error("Error handling Telegram webhook:", error);
    res.status(500).json({ error: error.message });
  }
});

// Setup Telegram webhook
app.post("/webhook/telegram/setup", async (req, res) => {
  try {
    if (!USE_TELEGRAM_DIRECT || !telegram) {
      return res.status(503).json({ error: "Telegram bot not configured" });
    }

    const externalUrl = req.body.external_url || req.headers["x-forwarded-host"] ||
                        req.headers.host || `localhost:${PORT}`;
    const protocol = req.headers["x-forwarded-proto"] || "http";
    const webhookUrl = `${protocol}://${externalUrl}/webhook/telegram`;

    console.log(`Setting Telegram webhook to: ${webhookUrl}`);

    const result = await telegram.setWebhook(webhookUrl);

    console.log("✅ Telegram webhook set:", result);

    res.json({
      success: true,
      webhook_url: webhookUrl,
      result: result,
    });
  } catch (error) {
    console.error("Failed to set Telegram webhook:", error);
    res.status(500).json({ error: error.message });
  }
});

// Get bot info
app.get("/webhook/telegram/info", async (req, res) => {
  try {
    if (!USE_TELEGRAM_DIRECT || !telegram) {
      return res.status(503).json({ error: "Telegram bot not configured" });
    }

    const botInfo = await telegram.getMe();

    res.json({
      configured: true,
      bot: botInfo.result,
      allowed_users: TELEGRAM_ALLOWED_USER_IDS,
    });
  } catch (error) {
    console.error("Failed to get bot info:", error);
    res.status(500).json({ error: error.message });
  }
});

// Test endpoint for development
app.post("/test", async (req, res) => {
  const { message, chat_id = "test-chat", user_id = "test-user" } = req.body;
  const response = await parser.parse(message || "status", chat_id, user_id);

  // If Clawbot is configured, try to send the message
  if (CLAWBOT_API_URL && response.text) {
    try {
      await clawbot.sendMessage(chat_id, response.text, {
        buttons: response.buttons,
      });
    } catch (error) {
      console.error("Failed to send via Clawbot:", error.message);
    }
  }

  res.json(response);
});

// Interactive test endpoint
app.get("/test", (req, res) => {
  res.send(`
    <html>
    <head><title>AmmanGate Claw Gateway Test</title></head>
    <body style="font-family: monospace; padding: 20px;">
      <h1>🛡️ AmmanGate Claw Gateway Test</h1>
      <form method="POST" action="/test">
        <input type="text" name="message" placeholder="Enter command..." style="width: 300px; padding: 10px;" />
        <button type="submit" style="padding: 10px;">Send</button>
      </form>
      <p>Try: <code>status</code>, <code>devices</code>, <code>help</code></p>
    </body>
    </html>
  `);
});

/**
 * Start server
 */
const server = app.listen(PORT, () => {
  console.log(`🛡️ AmmanGate Claw Gateway listening on port ${PORT}`);
  console.log(`   Core API: ${CORE_API_URL}`);
  console.log(`   Action PIN: ${ACTION_PIN}`);
  console.log("");

  // Telegram Bot Mode
  if (USE_TELEGRAM_DIRECT && telegram) {
    console.log(`📱 Telegram Bot Mode: ENABLED`);
    console.log(`   Allowed User IDs: ${TELEGRAM_ALLOWED_USER_IDS.join(", ") || "All users"}`);
    console.log("");
  } else {
    console.log(`   Clawbot API: ${CLAWBOT_API_URL}`);
    console.log("");
    console.log("Clawbot Integration:");
    console.log(`   Webhook URL: http://localhost:${PORT}/webhook/clawbot`);
    console.log(`   Register: curl -X POST http://localhost:${PORT}/webhook/register`);
  }

  console.log("");
  console.log("Available commands:");
  console.log("  status, devices, alerts, events");
  console.log("  search, find, why, help");
  console.log("  quarantine, block, unblock");
  console.log("");
  console.log(`Test UI: http://localhost:${PORT}/test`);
  console.log(`Test: curl -X POST http://localhost:${PORT}/test -H "Content-Type: application/json" -d '{"message":"help"}'`);
});

/**
 * Telegram Polling Mode (Alternative to Webhook)
 * Uses long polling to receive updates from Telegram
 */
let pollingOffset = 0;
let pollingActive = false;

async function startTelegramPolling() {
  if (!USE_TELEGRAM_DIRECT || !telegram) return;
  if (pollingActive) return;

  pollingActive = true;

  // Clear any existing webhook first
  try {
    await telegram.deleteWebhook();
    console.log("📱 Telegram Polling Mode: STARTED");
    console.log("   Bot will actively check for new messages");
  } catch (error) {
    console.log("⚠️  Warning: Could not delete webhook:", error.message);
  }

  // Start polling loop
  pollTelegram();
}

async function pollTelegram() {
  if (!pollingActive) return;

  try {
    // Use shorter timeout for faster response
    const result = await telegram.getUpdates(pollingOffset, 10);

    if (result.ok && result.result.length > 0) {
      for (const update of result.result) {
        // Update offset IMMEDIATELY after receiving update
        pollingOffset = update.update_id + 1;

        // Process the update
        await processTelegramUpdate(update);
      }
    }
  } catch (error) {
    // Silently ignore polling errors
    // console.error("Polling error:", error.message);
  }

  // Immediately poll again (no delay needed as getUpdates waits for updates)
  if (pollingActive) {
    setImmediate(pollTelegram);
  }
}

/**
 * Process Telegram update (from webhook or polling)
 */
const processedUpdates = new Set();

async function processTelegramUpdate(update) {
  if (!update.message) return;

  // Prevent duplicate processing
  const updateId = update.update_id;
  if (processedUpdates.has(updateId)) {
    return;
  }
  processedUpdates.add(updateId);

  // Clean up old processed IDs (keep last 1000)
  if (processedUpdates.size > 1000) {
    const firstItem = processedUpdates.values().next().value;
    processedUpdates.delete(firstItem);
  }

  const msg = update.message;
  const chatId = msg.chat.id;
  const userId = msg.from?.id;
  const text = msg.text || "";

  // Check authorization
  if (!telegram.isAllowedUser(userId)) {
    await telegram.sendMessage(chatId, "⛔ You are not authorized to use this bot.");
    return;
  }

  // Handle commands
  const parser = new CommandParser();
  const response = await parser.parse(text, chatId, userId);

  // Send response (only once)
  try {
    await telegram.sendMessage(chatId, response.text, {
      reply_to_message_id: msg.message_id,
    });
  } catch (error) {
    console.error("Failed to send message:", error.message);
  }
}

// Start polling if Telegram is enabled
if (USE_TELEGRAM_DIRECT && telegram) {
  startTelegramPolling();
}

export default app;
