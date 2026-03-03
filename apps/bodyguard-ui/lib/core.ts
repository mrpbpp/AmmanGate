// Core API client for bodyguard-core

const CORE_API_URL = process.env.CORE_API_URL || "http://127.0.0.1:8787";

export interface Device {
  id: string;
  mac: string;
  ip: string;
  hostname: string;
  vendor: string;
  type_guess: string;
  risk_score: number;
  last_seen: string;
}

export interface DeviceDetail extends Device {
  first_seen: string;
  tags: string[];
  notes: string;
  fingerprint?: DeviceFingerprint;
  activity_stats?: DeviceActivity;
}

export interface DeviceFingerprint {
  os_guess: string;
  browser_guess: string;
  open_ports: number[];
  mac_vendor: string;
}

export interface DeviceActivity {
  total_events: number;
  alerts_count: number;
  last_activity: string;
  first_seen: string;
  connection_count: number;
}

export interface Event {
  id: string;
  ts: string;
  device_id: string | null;
  category: string;
  severity: number;
  summary: string;
  raw: Record<string, unknown>;
}

export interface Alert {
  id: string;
  ts: string;
  device_id: string | null;
  severity: number;
  title: string;
  status: string;
  related_event_ids: string[];
}

export interface SystemStatus {
  uptime_sec: number;
  cpu_load: number;
  mem_used_mb: number;
  sensors: Record<string, boolean>;
  last_event_ts: string;
  clamav?: ClamAVStatus;
  suricata?: SuricataStatus;
}

export interface ClamAVStatus {
  enabled: boolean;
  running: boolean;
  version: string;
  db_version: string;
  address: string;
  last_check: string;
}

export interface SuricataAlert {
  timestamp: string;
  alert_id: string;
  signature_id: number;
  signature: string;
  category: string;
  severity: number;
  source_ip: string;
  source_port: number;
  dest_ip: string;
  dest_port: number;
  protocol: string;
  geo_location?: {
    country: string;
    city: string;
    region: string;
    isp: string;
    is_risky: boolean;
    formatted: string;
  };
}

export interface SuricataStatus {
  enabled: boolean;
  running: boolean;
  version: string;
  eve_log: string;
  eve_log_accessible: boolean;
  eve_log_error?: string;
  stats: {
    total_alerts: number;
    high_severity: number;
    medium_severity: number;
    low_severity: number;
    last_alert: string;
    packets_seen: number;
    bytes_seen: number;
    start_time: string;
  };
  alerts_count: number;
}

export interface ApprovalChallenge {
  action_id: string;
  approval_id: string;
  expires_at: string;
  message: string;
}

export interface ActionResult {
  action_id: string;
  status: string;
  detail: string;
}

export interface FilterRule {
  id: string;
  name: string;
  type: string;
  pattern: string;
  enabled: boolean;
  created_at: string;
}

export interface DNSQueryLog {
  id: string;
  ts: string;
  device_id: string;
  domain: string;
  blocked: boolean;
  rule_id: string;
}

// API Client class
export class CoreClient {
  private baseUrl: string;

  constructor(baseUrl?: string) {
    this.baseUrl = baseUrl || CORE_API_URL;
  }

  // Public fetch method for custom requests
  async fetch(path: string, options?: RequestInit): Promise<Response> {
    const url = `${this.baseUrl}${path}`;
    return fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    });
  }

  private async fetchTyped<T>(path: string, options?: RequestInit): Promise<T> {
    const url = `${this.baseUrl}/v1${path}`;
    const response = await fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`API error ${response.status}: ${text}`);
    }

    return response.json() as Promise<T>;
  }

  // Health check
  async health(): Promise<{ ok: boolean; ts: string; version: string }> {
    return this.fetchTyped("/health");
  }

  // System status
  async getSystemStatus(): Promise<SystemStatus> {
    return this.fetchTyped("/system/status");
  }

  // Network info
  async getNetworkInfo(): Promise<{
    hostname: string;
    ips: string[];
    dns_port: number;
    web_port: number;
    primary_ip: string;
    dns_running: boolean;
  }> {
    return this.fetchTyped("/system/network");
  }

  // Devices
  async getDevices(params?: { q?: string; limit?: number }): Promise<{ items: Device[] }> {
    const searchParams = new URLSearchParams();
    if (params?.q) searchParams.set("q", params.q);
    if (params?.limit) searchParams.set("limit", params.limit.toString());
    const query = searchParams.toString();
    return this.fetchTyped(`/devices${query ? `?${query}` : ""}`);
  }

  async getDevice(id: string): Promise<DeviceDetail> {
    return this.fetchTyped(`/devices/${id}`);
  }

  // Events
  async getEvents(params?: {
    since?: string;
    min_severity?: number;
    device_id?: string;
    limit?: number;
  }): Promise<{ items: Event[] }> {
    const searchParams = new URLSearchParams();
    if (params?.since) searchParams.set("since", params.since);
    if (params?.min_severity) searchParams.set("min_severity", params.min_severity.toString());
    if (params?.device_id) searchParams.set("device_id", params.device_id);
    if (params?.limit) searchParams.set("limit", params.limit.toString());
    const query = searchParams.toString();
    return this.fetchTyped(`/events${query ? `?${query}` : ""}`);
  }

  // Alerts
  async getActiveAlerts(): Promise<{ items: Alert[] }> {
    return this.fetchTyped("/alerts/active");
  }

  // Actions
  async requestApproval(data: {
    action_type: string;
    target: Record<string, unknown>;
    ttl_sec?: number;
    requested_by: string;
  }): Promise<ApprovalChallenge> {
    return this.fetchTyped("/actions/request-approval", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async approveAction(approvalId: string, pin: string): Promise<ActionResult> {
    return this.fetchTyped("/actions/approve", {
      method: "POST",
      body: JSON.stringify({ approval_id: approvalId, pin }),
    });
  }

  async getPendingActions(): Promise<{ items: unknown[] }> {
    return this.fetchTyped("/actions/pending");
  }

  // Parental Control
  async getFilters(): Promise<{ items: FilterRule[] }> {
    return this.fetchTyped("/filters");
  }

  async addFilter(data: {
    name: string;
    type: string;
    pattern: string;
  }): Promise<FilterRule> {
    return this.fetchTyped("/filters", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async deleteFilter(id: string): Promise<{ success: boolean }> {
    return this.fetchTyped(`/filters/${id}`, {
      method: "DELETE",
    });
  }

  async toggleFilter(id: string, enabled: boolean): Promise<{ success: boolean; enabled: boolean }> {
    return this.fetchTyped(`/filters/${id}/toggle`, {
      method: "PUT",
      body: JSON.stringify({ enabled }),
    });
  }

  async getDeviceProfile(deviceId: string): Promise<{
    device_id: string;
    filter_level: string;
  }> {
    return this.fetchTyped(`/devices/${deviceId}/profile`);
  }

  async setDeviceProfile(deviceId: string, filterLevel: string): Promise<{
    success: boolean;
    device_id: string;
    filter_level: string;
  }> {
    return this.fetchTyped(`/devices/${deviceId}/profile`, {
      method: "PUT",
      body: JSON.stringify({ filter_level: filterLevel }),
    });
  }

  async getDNSLogs(limit?: number): Promise<{ items: DNSQueryLog[] }> {
    return this.fetchTyped(`/dns-logs?limit=${limit || 50}`);
  }

  // ClamAV
  async getClamAVStatus(): Promise<ClamAVStatus> {
    return this.fetchTyped("/clamav/status");
  }

  async refreshClamAV(): Promise<{ success: boolean; message: string; status: ClamAVStatus }> {
    return this.fetchTyped("/clamav/refresh", {
      method: "POST",
    });
  }

  // Suricata
  async getSuricataStatus(): Promise<SuricataStatus> {
    return this.fetchTyped("/suricata/status");
  }

  async getSuricataAlerts(limit?: number): Promise<{ items: SuricataAlert[] }> {
    return this.fetchTyped(`/suricata/alerts?limit=${limit || 50}`);
  }

  // WebSocket connection
  connectWebSocket(): WebSocket {
    const wsUrl = this.baseUrl.replace("http://", "ws://").replace("https://", "wss://");
    return new WebSocket(`${wsUrl}/v1/ws`);
  }
}

// Singleton instance
let coreClient: CoreClient | null = null;

export function getCoreClient(): CoreClient {
  if (!coreClient) {
    coreClient = new CoreClient();
  }
  return coreClient;
}
