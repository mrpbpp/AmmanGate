"use client";

import { useEffect, useState } from "react";
import { getCoreClient } from "@/lib/core";

interface SuricataAlert {
  alert_id: string;
  signature: string;
  category: string;
  severity: number;
  source_ip: string;
  source_port: number;
  dest_ip: string;
  dest_port: number;
  timestamp: string;
  geo_location?: {
    country: string;
    is_risky: boolean;
  };
}

interface SuricataStats {
  total_alerts: number;
  high_severity: number;
  medium_severity: number;
  low_severity: number;
}

interface SuricataStatus {
  enabled: boolean;
  running: boolean;
  stats?: SuricataStats;
}

export function SuricataStatus() {
  const [status, setStatus] = useState<SuricataStatus | null>(null);
  const [alerts, setAlerts] = useState<SuricataAlert[]>([]);
  const [loading, setLoading] = useState(true);

  const client = getCoreClient();

  const loadData = async () => {
    try {
      const statusRes = await fetch("http://127.0.0.1:8787/v1/suricata/status");
      const statusData = await statusRes.json();
      setStatus(statusData);

      const alertsRes = await fetch("http://127.0.0.1:8787/v1/suricata/alerts?limit=10");
      const alertsData = await alertsRes.json();
      setAlerts(alertsData.items || []);
    } catch (error) {
      console.error("Failed to load Suricata data:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 10000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold text-white mb-4">Suricata IDS</h2>
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      </div>
    );
  }

  const isRunning = status?.running ?? false;
  const isEnabled = status?.enabled ?? false;
  const stats = status?.stats;

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold text-white">Suricata IDS</h2>
        <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium ${
          isRunning
            ? "bg-green-500/20 text-green-400"
            : isEnabled
            ? "bg-yellow-500/20 text-yellow-400"
            : "bg-slate-500/20 text-slate-400"
        }`}>
          <span className={`w-2 h-2 rounded-full ${
            isRunning ? "bg-green-400 animate-pulse" : isEnabled ? "bg-yellow-400" : "bg-slate-400"
          }`}></span>
          {isRunning ? "Running" : isEnabled ? "Enabled - Stopped" : "Disabled"}
        </div>
      </div>

      {!isRunning && (
        <div className="bg-blue-900/30 border border-blue-500/30 rounded-lg p-4 mb-4">
          <h3 className="font-semibold text-blue-400 mb-2">📋 Suricata Setup Required</h3>
          <p className="text-slate-300 text-sm mb-2">
            Suricata is {isEnabled ? "enabled but not running" : "disabled"}. Install and configure Suricata to enable IDS/IPS protection.
          </p>
          <ol className="list-decimal list-inside space-y-1 text-slate-300 text-sm">
            <li>Download Suricata for Windows</li>
            <li>Configure EVE JSON logging to: C:\Suricata\log\eve.json</li>
            <li>Set SURICATA_ENABLED=true in .env</li>
            <li>Restart bodyguard-core</li>
          </ol>
        </div>
      )}

      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
          <div className="bg-slate-800/50 rounded-lg p-3">
            <p className="text-slate-400 text-xs">Total Alerts</p>
            <p className="text-xl font-bold text-white">{stats.total_alerts}</p>
          </div>
          <div className="bg-slate-800/50 rounded-lg p-3">
            <p className="text-slate-400 text-xs">High Severity</p>
            <p className="text-xl font-bold text-red-400">{stats.high_severity}</p>
          </div>
          <div className="bg-slate-800/50 rounded-lg p-3">
            <p className="text-slate-400 text-xs">Medium</p>
            <p className="text-xl font-bold text-yellow-400">{stats.medium_severity}</p>
          </div>
          <div className="bg-slate-800/50 rounded-lg p-3">
            <p className="text-slate-400 text-xs">Low</p>
            <p className="text-xl font-bold text-blue-400">{stats.low_severity}</p>
          </div>
        </div>
      )}

      <div>
        <h3 className="font-semibold text-white mb-3">
          Recent Alerts {alerts.length > 0 && `(${alerts.length})`}
        </h3>
        {alerts.length === 0 ? (
          <div className="bg-slate-800/30 rounded-lg p-6 text-center">
            <p className="text-slate-400">
              {isRunning ? "No recent alerts - system is clean!" : "Enable Suricata to see alerts"}
            </p>
          </div>
        ) : (
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {alerts.map((alert) => (
              <div
                key={alert.alert_id}
                className="bg-slate-800/50 rounded-lg p-3 border-l-4"
                style={{
                  borderLeftColor: alert.severity >= 3 ? "#ef4444" :
                                   alert.severity === 2 ? "#eab308" :
                                   "#3b82f6"
                }}
              >
                <div className="flex items-start justify-between mb-1">
                  <div className="flex-1">
                    <p className="text-white font-medium text-sm">{alert.signature}</p>
                    <p className="text-slate-400 text-xs">
                      {alert.source_ip}:{alert.source_port} → {alert.dest_ip}:{alert.dest_port}
                    </p>
                  </div>
                  <span className={`text-xs px-2 py-0.5 rounded ${
                    alert.severity >= 3 ? "bg-red-500/20 text-red-400" :
                    alert.severity === 2 ? "bg-yellow-500/20 text-yellow-400" :
                    "bg-blue-500/20 text-blue-400"
                  }`}>
                    Sev {alert.severity}
                  </span>
                </div>
                <div className="flex items-center gap-3 text-xs text-slate-500">
                  <span>{alert.category}</span>
                  <span>•</span>
                  <span>{new Date(alert.timestamp).toLocaleTimeString()}</span>
                  {alert.geo_location && (
                    <>
                      <span>•</span>
                      <span>{alert.geo_location.country}</span>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
