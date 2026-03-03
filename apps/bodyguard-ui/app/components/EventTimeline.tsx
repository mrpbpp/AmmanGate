"use client";

import { useEffect, useState } from "react";
import { getCoreClient, type Event } from "@/lib/core";
import Link from "next/link";

export function EventTimeline() {
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(true);

  const client = getCoreClient();

  useEffect(() => {
    loadEvents();
    // Refresh every 10 seconds
    const interval = setInterval(loadEvents, 10000);
    return () => clearInterval(interval);
  }, []);

  const loadEvents = async () => {
    try {
      const data = await client.getEvents({
        since: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
        min_severity: 1,
        limit: 5,
      });
      setEvents(data.items ?? []);
    } catch (error) {
      console.error("Failed to load events:", error);
    } finally {
      setLoading(false);
    }
  };

  const getEventIcon = (category: string, severity: number) => {
    if (severity >= 70) return "🔴";
    if (severity >= 40) return "🟡";
    if (category === "honeypot") return "🎯";
    if (category === "arp") return "🔍";
    if (category === "network") return "🌐";
    return "📋";
  };

  const getEventStyle = (severity: number) => {
    if (severity >= 70) return "bg-red-900/30 border-red-700";
    if (severity >= 40) return "bg-yellow-900/30 border-yellow-700";
    return "bg-slate-700/50 border-slate-600";
  };

  const formatTime = (ts: string) => {
    const date = new Date(ts);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Event Timeline</h2>
        <Link
          href="/timeline"
          className="text-blue-400 hover:text-blue-300 text-sm font-medium"
        >
          View All →
        </Link>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
        </div>
      ) : events.length === 0 ? (
        <div className="relative">
          <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-slate-700" />
          <div className="space-y-4">
            <div className="relative pl-10">
              <div className="absolute left-2 w-4 h-4 bg-emerald-500 rounded-full border-2 border-slate-800" />
              <div className="p-3 bg-emerald-900/30 rounded-lg border border-emerald-700">
                <p className="text-sm font-medium text-emerald-300">System Ready</p>
                <p className="text-xs text-emerald-400 mt-1">Waiting for security events...</p>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="relative">
          {/* Timeline line */}
          <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-slate-700" />

          <div className="space-y-4">
            {events.map((event, index) => (
              <div key={event.id} className="relative pl-10">
                <div
                  className={`absolute left-2 w-4 h-4 rounded-full border-2 border-slate-800 ${
                    event.severity >= 70
                      ? "bg-red-500"
                      : event.severity >= 40
                      ? "bg-yellow-500"
                      : "bg-blue-500"
                  }`}
                />
                <div className={`p-3 rounded-lg border ${getEventStyle(event.severity)}`}>
                  <div className="flex items-start gap-2">
                    <span>{getEventIcon(event.category, event.severity)}</span>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-white">{event.summary}</p>
                      <div className="flex items-center gap-2 mt-1">
                        <span className="text-xs px-2 py-0.5 rounded bg-slate-700 text-slate-300">
                          {event.category}
                        </span>
                        <p className="text-xs text-slate-400">{formatTime(event.ts)}</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
