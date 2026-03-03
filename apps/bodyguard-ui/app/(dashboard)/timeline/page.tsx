"use client";

import { useEffect, useState } from "react";
import { getCoreClient, type Event } from "@/lib/core";
import { SeverityBadge } from "@/app/components/SeverityBadge";

type FilterCategory = "all" | "critical" | "honeypot" | "dns" | "suricata" | "malware" | "parental" | "device";
type TimeRange = "1h" | "6h" | "24h" | "7d" | "30d" | "all";

interface TimelineStats {
  total: number;
  critical: number;
  honeypot_hits: number;
  dns_blocked: number;
  suricata_alerts: number;
  malware_detected: number;
}

export default function TimelinePage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [stats, setStats] = useState<TimelineStats>({
    total: 0,
    critical: 0,
    honeypot_hits: 0,
    dns_blocked: 0,
    suricata_alerts: 0,
    malware_detected: 0,
  });
  const [loading, setLoading] = useState(true);
  const [filterCategory, setFilterCategory] = useState<FilterCategory>("all");
  const [timeRange, setTimeRange] = useState<TimeRange>("24h");
  const [searchQuery, setSearchQuery] = useState("");

  const client = getCoreClient();

  useEffect(() => {
    loadEvents();
    const interval = setInterval(loadEvents, 10000);
    return () => clearInterval(interval);
  }, [timeRange]);

  const loadEvents = async () => {
    setLoading(true);
    try {
      // Calculate time range
      const now = Date.now();
      let since = new Date(0);
      switch (timeRange) {
        case "1h": since = new Date(now - 1 * 60 * 60 * 1000); break;
        case "6h": since = new Date(now - 6 * 60 * 60 * 1000); break;
        case "24h": since = new Date(now - 24 * 60 * 60 * 1000); break;
        case "7d": since = new Date(now - 7 * 24 * 60 * 60 * 1000); break;
        case "30d": since = new Date(now - 30 * 24 * 60 * 60 * 1000); break;
        case "all": since = new Date(0); break;
      }

      const data = await client.getEvents({
        since: since.toISOString(),
        limit: 200,
      });

      const filteredEvents = (data.items ?? []).filter((event) => {
        // Apply category filter
        if (filterCategory !== "all" && filterCategory !== "critical") {
          if (filterCategory === "honeypot" && event.category !== "honeypot") return false;
          if (filterCategory === "dns" && !event.summary.toLowerCase().includes("dns") && !event.summary.toLowerCase().includes("blocked")) return false;
          if (filterCategory === "suricata" && event.category !== "suricata") return false;
          if (filterCategory === "malware" && !event.summary.toLowerCase().includes("malware") && !event.summary.toLowerCase().includes("virus")) return false;
          if (filterCategory === "parental" && !event.summary.toLowerCase().includes("parental") && !event.summary.toLowerCase().includes("filter")) return false;
          if (filterCategory === "device" && event.category !== "network" && event.category !== "arp") return false;
        }
        if (filterCategory === "critical" && event.severity < 70) return false;

        // Apply search filter
        if (searchQuery && !event.summary.toLowerCase().includes(searchQuery.toLowerCase())) {
          return false;
        }

        return true;
      });

      setEvents(filteredEvents);

      // Calculate stats
      const items = data.items ?? [];
      setStats({
        total: items.length,
        critical: items.filter(e => e.severity >= 70).length,
        honeypot_hits: items.filter(e => e.category === "honeypot").length,
        dns_blocked: items.filter(e => e.summary.toLowerCase().includes("dns") || e.summary.toLowerCase().includes("blocked")).length,
        suricata_alerts: items.filter(e => e.category === "suricata").length,
        malware_detected: items.filter(e => e.summary.toLowerCase().includes("malware") || e.summary.toLowerCase().includes("virus")).length,
      });
    } catch (error) {
      console.error("Failed to load events:", error);
    } finally {
      setLoading(false);
    }
  };

  const getEventIcon = (category: string, severity: number) => {
    if (severity >= 70) return "🔴";
    if (category === "honeypot") return "🎯";
    if (category === "suricata") return "🛡️";
    if (category === "malware") return "🦠";
    if (category === "dns") return "🌐";
    if (category === "parental") return "👨‍👩‍👧‍👦";
    if (category === "network" || category === "arp") return "📱";
    return "📋";
  };

  const getCategoryColor = (category: string) => {
    const colors: Record<string, string> = {
      honeypot: "bg-purple-500/20 text-purple-400 border-purple-500/30",
      suricata: "bg-red-500/20 text-red-400 border-red-500/30",
      malware: "bg-orange-500/20 text-orange-400 border-orange-500/30",
      dns: "bg-blue-500/20 text-blue-400 border-blue-500/30",
      parental: "bg-green-500/20 text-green-400 border-green-500/30",
      network: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
      arp: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
    };
    return colors[category] || "bg-slate-500/20 text-slate-400 border-slate-500/30";
  };

  const formatDateTime = (ts: string) => {
    const date = new Date(ts);
    return date.toLocaleString("id-ID", {
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  };

  const formatTimeAgo = (ts: string) => {
    const date = new Date(ts);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return "Baru saja";
    if (diffMins < 60) return `${diffMins} menit yang lalu`;
    if (diffHours < 24) return `${diffHours} jam yang lalu`;
    return `${diffDays} hari yang lalu`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-white">Timeline Keamanan</h1>
        <p className="text-slate-400 mt-1">
          Riwayat lengkap aktivitas keamanan jaringan Anda
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
        <div className="card bg-slate-800/50">
          <p className="text-slate-400 text-xs">Total Event</p>
          <p className="text-2xl font-bold text-white">{stats.total}</p>
        </div>
        <div className="card bg-red-900/30 border-red-700/30">
          <p className="text-red-300 text-xs">Kritis</p>
          <p className="text-2xl font-bold text-red-400">{stats.critical}</p>
        </div>
        <div className="card bg-purple-900/30 border-purple-700/30">
          <p className="text-purple-300 text-xs">Honeypot</p>
          <p className="text-2xl font-bold text-purple-400">{stats.honeypot_hits}</p>
        </div>
        <div className="card bg-blue-900/30 border-blue-700/30">
          <p className="text-blue-300 text-xs">DNS Diblokir</p>
          <p className="text-2xl font-bold text-blue-400">{stats.dns_blocked}</p>
        </div>
        <div className="card bg-orange-900/30 border-orange-700/30">
          <p className="text-orange-300 text-xs">Malware</p>
          <p className="text-2xl font-bold text-orange-400">{stats.malware_detected}</p>
        </div>
        <div className="card bg-red-900/30 border-red-700/30">
          <p className="text-red-300 text-xs">IDS/IPS</p>
          <p className="text-2xl font-bold text-red-400">{stats.suricata_alerts}</p>
        </div>
      </div>

      {/* Filters */}
      <div className="card">
        <div className="flex flex-col md:flex-row gap-4">
          {/* Search */}
          <div className="flex-1">
            <input
              type="text"
              placeholder="🔍 Cari event..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:border-blue-500"
            />
          </div>

          {/* Category Filter */}
          <select
            value={filterCategory}
            onChange={(e) => setFilterCategory(e.target.value as FilterCategory)}
            className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-blue-500"
          >
            <option value="all">Semua Kategori</option>
            <option value="critical">⚠️ Kritis Saja</option>
            <option value="honeypot">🎯 Honeypot</option>
            <option value="dns">🌐 DNS Filter</option>
            <option value="suricata">🛡️ IDS/IPS</option>
            <option value="malware">🦠 Malware</option>
            <option value="parental">👨‍👩‍👧‍👦 Parental</option>
            <option value="device">📱 Perangkat</option>
          </select>

          {/* Time Range */}
          <select
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value as TimeRange)}
            className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-blue-500"
          >
            <option value="1h">1 Jam Terakhir</option>
            <option value="6h">6 Jam Terakhir</option>
            <option value="24h">24 Jam Terakhir</option>
            <option value="7d">7 Hari Terakhir</option>
            <option value="30d">30 Hari Terakhir</option>
            <option value="all">Semua Waktu</option>
          </select>
        </div>
      </div>

      {/* Events List */}
      {loading ? (
        <div className="card flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      ) : events.length === 0 ? (
        <div className="card">
          <div className="text-center py-12">
            <div className="text-6xl mb-4">📭</div>
            <h3 className="text-xl font-semibold text-white mb-2">Tidak Ada Event</h3>
            <p className="text-slate-400">
              {filterCategory === "critical"
                ? "Tidak ada event kritis dalam periode ini"
                : "Belum ada event keamanan yang tercatat"}
            </p>
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          {events.map((event, index) => (
            <div
              key={event.id}
              className="card bg-slate-800/50 border-l-4 hover:bg-slate-800 transition-colors"
              style={{
                borderLeftColor: event.severity >= 70 ? "#ef4444" :
                                 event.severity >= 40 ? "#eab308" :
                                 "#3b82f6"
              }}
            >
              <div className="flex items-start gap-4">
                {/* Icon */}
                <div className="text-3xl shrink-0">
                  {getEventIcon(event.category, event.severity)}
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  {/* Summary */}
                  <p className="text-white font-medium mb-2">{event.summary}</p>

                  {/* Tags */}
                  <div className="flex flex-wrap items-center gap-2 mb-2">
                    <span className={`text-xs px-2 py-0.5 rounded border ${getCategoryColor(event.category)}`}>
                      {event.category}
                    </span>
                    <SeverityBadge severity={event.severity} />
                    {event.raw && typeof event.raw === "object" && (
                      <>
                        {"geo_location" in event.raw && (
                          <span className="text-xs px-2 py-0.5 rounded bg-slate-700 text-slate-300">
                            🌍 {(event.raw.geo_location as any)?.country || "Unknown"}
                          </span>
                        )}
                        {"signature" in event.raw && (
                          <span className="text-xs px-2 py-0.5 rounded bg-slate-700 text-slate-300">
                            🔍 {(event.raw as any).signature}
                          </span>
                        )}
                      </>
                    )}
                  </div>

                  {/* Time */}
                  <div className="flex items-center gap-2 text-xs text-slate-400">
                    <span>📅 {formatDateTime(event.ts)}</span>
                    <span>•</span>
                    <span>{formatTimeAgo(event.ts)}</span>
                  </div>
                </div>

                {/* Severity Indicator */}
                <div className={`w-3 h-3 rounded-full shrink-0 ${
                  event.severity >= 70 ? "bg-red-500" :
                  event.severity >= 40 ? "bg-yellow-500" :
                  "bg-blue-500"
                }`} />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Event Count */}
      {events.length > 0 && (
        <div className="text-center text-sm text-slate-400">
          Menampilkan {events.length} event dari periode yang dipilih
        </div>
      )}
    </div>
  );
}
