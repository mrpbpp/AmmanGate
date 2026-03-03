import { useState } from "react";
import { Alert } from "@/lib/core";
import { SeverityBadge } from "./SeverityBadge";
import Link from "next/link";
import { useRouter } from "next/navigation";

interface ActiveAlertsProps {
  alerts: Alert[];
  onDismissAlert?: (alertId: string) => void;
}

export function ActiveAlerts({ alerts, onDismissAlert }: ActiveAlertsProps) {
  const [page, setPage] = useState(1);
  const [dismissedIds, setDismissedIds] = useState<Set<string>>(new Set());
  const router = useRouter();

  const itemsPerPage = 5;
  const startIndex = (page - 1) * itemsPerPage;
  const endIndex = startIndex + itemsPerPage;

  // Filter out dismissed alerts
  const activeAlerts = alerts.filter((alert) => !dismissedIds.has(alert.id));
  const totalPages = Math.ceil(activeAlerts.length / itemsPerPage);
  const currentPageAlerts = activeAlerts.slice(startIndex, endIndex);

  const handleDismiss = (alertId: string) => {
    setDismissedIds((prev) => new Set([...prev, alertId]));
    if (onDismissAlert) {
      onDismissAlert(alertId);
    }
  };

  const handleViewDetails = (alertId: string) => {
    router.push(`/alerts?highlight=${alertId}`);
  };

  const handlePrevPage = () => {
    setPage((p) => Math.max(1, p - 1));
  };

  const handleNextPage = () => {
    setPage((p) => Math.min(totalPages, p + 1));
  };

  if (activeAlerts.length === 0) {
    return (
      <div className="card bg-emerald-900/20 border-emerald-700">
        <div className="flex items-center gap-3">
          <span className="text-2xl">✅</span>
          <div>
            <h3 className="font-semibold text-emerald-300">No Active Alerts</h3>
            <p className="text-sm text-emerald-400">Your network is secure</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-semibold text-white">Active Alerts</h2>
          <span className="px-3 py-1 bg-red-900/50 text-red-300 rounded-full text-sm font-medium border border-red-700">
            {activeAlerts.length} active
          </span>
        </div>
        {totalPages > 1 && (
          <div className="flex items-center gap-2">
            <button
              onClick={handlePrevPage}
              disabled={page === 1}
              className="w-8 h-8 flex items-center justify-center rounded-lg bg-slate-700 text-white hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              ‹
            </button>
            <span className="text-sm text-slate-400">
              {page} / {totalPages}
            </span>
            <button
              onClick={handleNextPage}
              disabled={page === totalPages}
              className="w-8 h-8 flex items-center justify-center rounded-lg bg-slate-700 text-white hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              ›
            </button>
          </div>
        )}
      </div>

      <div className="space-y-2">
        {currentPageAlerts.map((alert) => (
          <div
            key={alert.id}
            className="relative bg-slate-700/50 rounded-lg p-4 hover:bg-slate-700 transition-all border border-slate-600"
          >
            {/* Close button */}
            <button
              onClick={() => handleDismiss(alert.id)}
              className="absolute top-2 right-2 w-6 h-6 flex items-center justify-center rounded text-slate-400 hover:text-white hover:bg-slate-600 transition-colors"
              title="Dismiss"
            >
              ✕
            </button>

            <div className="flex items-start justify-between gap-4 pr-6">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-2">
                  <SeverityBadge severity={alert.severity} />
                  <span className="text-xs text-slate-400">
                    {new Date(alert.ts).toLocaleString()}
                  </span>
                </div>
                <h3 className="font-medium text-white">{alert.title}</h3>
                {alert.device_id && (
                  <p className="text-sm text-slate-400 mt-1">
                    Device:{" "}
                    <code className="px-2 py-0.5 bg-slate-800 rounded text-xs">
                      {alert.device_id}
                    </code>
                  </p>
                )}
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleViewDetails(alert.id)}
                  className="btn btn-primary text-sm py-1.5 px-3"
                >
                  View Details
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {activeAlerts.length > itemsPerPage && (
        <div className="text-center pt-2">
          <Link
            href="/alerts"
            className="text-blue-400 hover:text-blue-300 text-sm font-medium"
          >
            View all {activeAlerts.length} alerts →
          </Link>
        </div>
      )}
    </div>
  );
}
