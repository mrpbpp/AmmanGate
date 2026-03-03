"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getCoreClient, type SystemStatus, type Alert, type Device, type SuricataStatus as SuricataStatusType } from "@/lib/core";
import { StatusCard } from "../components/StatusCard";
import { ActiveAlerts } from "../components/ActiveAlerts";
import { RecentDevices } from "../components/RecentDevices";
import { EventTimeline } from "../components/EventTimeline";
import { AIAnalysis } from "../components/AIAnalysis";
import { SuricataStatus } from "../components/SuricataStatus";

export default function HomePage() {
  const router = useRouter();
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [suricata, setSuricata] = useState<SuricataStatusType | null>(null);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [wsConnected, setWsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const client = getCoreClient();

  useEffect(() => {
    loadInitialData();
    connectWebSocket();

    // Refresh data every 30 seconds
    const interval = setInterval(loadInitialData, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadInitialData = async () => {
    try {
      const [statusData, alertsData, devicesData, suricataData] = await Promise.all([
        client.getSystemStatus(),
        client.getActiveAlerts(),
        client.getDevices({ limit: 10 }),
        client.getSuricataStatus(),
      ]);

      setStatus(statusData);
      setSuricata(suricataData);
      // Handle null items from API
      setAlerts(alertsData.items ?? []);
      setDevices(devicesData.items ?? []);
      setError(null);
    } catch (error) {
      console.error("Failed to load data:", error);
      setError("Failed to load dashboard data");
      // Set empty arrays to prevent null reference errors
      setAlerts([]);
      setDevices([]);
    } finally {
      setLoading(false);
    }
  };

  const connectWebSocket = () => {
    try {
      const ws = client.connectWebSocket();

      ws.onopen = () => {
        setWsConnected(true);
        console.log("WebSocket connected");
      };

      ws.onclose = () => {
        setWsConnected(false);
        console.log("WebSocket disconnected, reconnecting in 5s...");
        setTimeout(connectWebSocket, 5000);
      };

      ws.onerror = (error) => {
        console.error("WebSocket error:", error);
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log("WS message:", message);

          // Refresh data on important events
          if (message.type === "action_pending" ||
              message.type === "action_executed" ||
              message.type === "honeypot_hit" ||
              message.type === "new_event" ||
              message.type === "new_alert" ||
              message.type === "suricata_alert") {
            loadInitialData();
          }
        } catch (e) {
          console.error("Failed to parse WS message:", e);
        }
      };
    } catch (error) {
      console.error("Failed to connect WebSocket:", error);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
        <p className="text-slate-400">Loading dashboard...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
        <div className="text-red-400 text-4xl">⚠️</div>
        <p className="text-red-400">{error}</p>
        <button
          onClick={loadInitialData}
          className="btn btn-primary"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Status Banner */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white">Dashboard</h1>
          <p className="text-slate-400 mt-1">AI Home Cyber Bodyguard</p>
        </div>
        <div className="flex items-center gap-2">
          <div className={`w-3 h-3 rounded-full ${wsConnected ? "bg-emerald-500 animate-pulse" : "bg-slate-500"}`} />
          <span className="text-sm text-slate-400">
            {wsConnected ? "Live" : "Reconnecting..."}
          </span>
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatusCard
          title="Active Alerts"
          value={alerts.length}
          color={alerts.length > 0 ? "danger" : "success"}
          icon="🚨"
          onClick={() => router.push("/alerts")}
        />
        <StatusCard
          title="Known Devices"
          value={devices.length}
          color="primary"
          icon="📱"
          onClick={() => router.push("/devices")}
        />
        <StatusCard
          title="Uptime"
          value={status ? formatUptime(status.uptime_sec) : "-"}
          color="info"
          icon="⏱️"
        />
        <StatusCard
          title="Memory Usage"
          value={status ? `${status.mem_used_mb} MB` : "-"}
          color="info"
          icon="💾"
        />
      </div>

      {/* Security Modules Status */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {status?.clamav && (
          <div className="card bg-slate-800/50">
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-semibold text-slate-300">🦠 ClamAV</h3>
              <div className={`w-2 h-2 rounded-full ${status.clamav.running ? "bg-green-500" : status.clamav.enabled ? "bg-yellow-500" : "bg-slate-500"}`} />
            </div>
            <p className="text-lg font-bold text-white">
              {status.clamav.running ? "Active" : status.clamav.enabled ? "Not Connected" : "Disabled"}
            </p>
            {status.clamav.running && (
              <p className="text-xs text-slate-400 mt-1">v{status.clamav.version} | DB: {status.clamav.db_version}</p>
            )}
          </div>
        )}
        {suricata && (
          <div className="card bg-slate-800/50">
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-semibold text-slate-300">🛡️ Suricata IDS</h3>
              <div className={`w-2 h-2 rounded-full ${suricata.running ? "bg-green-500" : suricata.enabled ? "bg-yellow-500" : "bg-slate-500"}`} />
            </div>
            <p className="text-lg font-bold text-white">
              {suricata.running ? "Active" : suricata.enabled ? "Not Connected" : "Disabled"}
            </p>
            {suricata.running && (
              <p className="text-xs text-slate-400 mt-1">v{suricata.version} | Alerts: {suricata.alerts_count}</p>
            )}
          </div>
        )}
        {status?.sensors && (
          <div className="card bg-slate-800/50">
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-semibold text-slate-300">🌐 DNS Filter</h3>
              <div className={`w-2 h-2 rounded-full ${status.sensors.dns ? "bg-green-500" : "bg-slate-500"}`} />
            </div>
            <p className="text-lg font-bold text-white">{status.sensors.dns ? "Running" : "Stopped"}</p>
          </div>
        )}
      </div>

      {/* Active Alerts */}
      <ActiveAlerts alerts={alerts} />

      {/* Suricata IDS Status */}
      <SuricataStatus />

      {/* AI Security Analysis */}
      <AIAnalysis deviceCount={devices.length} alertCount={alerts.length} />

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <RecentDevices devices={devices} />
        <EventTimeline />
      </div>
    </div>
  );
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);

  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${mins}m`;
  return `${mins}m`;
}
