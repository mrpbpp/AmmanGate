"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Device } from "@/lib/core";
import { RiskScoreBadge } from "./RiskScoreBadge";

interface RecentDevicesProps {
  devices: Device[];
}

export function RecentDevices({ devices }: RecentDevicesProps) {
  const router = useRouter();
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);

  const handleViewDetails = (device: Device) => {
    setSelectedDevice(device);
  };

  const getDeviceIcon = (type: string) => {
    switch (type?.toLowerCase()) {
      case "mobile": return "📱";
      case "laptop": return "💻";
      case "desktop": return "🖥️";
      case "router": return "📡";
      case "iot": return "🔌";
      case "tablet": return "📟";
      default: return "📡";
    }
  };

  return (
    <>
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Recent Devices</h2>
          <button
            onClick={() => router.push("/devices")}
            className="text-blue-400 hover:text-blue-300 text-sm font-medium"
          >
            View All →
          </button>
        </div>

        <div className="space-y-3">
          {devices.length === 0 ? (
            <p className="text-slate-500 text-center py-8">No devices detected yet</p>
          ) : (
            devices.slice(0, 5).map((device) => (
              <div
                key={device.id}
                className="flex items-center justify-between p-3 bg-slate-700/50 rounded-lg hover:bg-slate-700 transition-colors cursor-pointer"
                onClick={() => router.push(`/devices/${device.id}`)}
              >
                <div className="flex items-center gap-3 flex-1">
                  <span className="text-2xl">{getDeviceIcon(device.type_guess)}</span>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <h3 className="font-medium text-white">
                        {device.hostname || device.vendor || "Unknown"}
                      </h3>
                      <RiskScoreBadge score={device.risk_score} />
                    </div>
                    <p className="text-sm text-slate-400 mt-1">
                      {device.ip || "No IP"} • <span className="font-mono text-xs">{device.mac}</span>
                    </p>
                    <p className="text-xs text-slate-500 mt-1">
                      Last seen: {device.last_seen ? new Date(device.last_seen).toLocaleString() : "Never"}
                    </p>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Device Detail Modal */}
      {selectedDevice && (
        <div
          className="fixed inset-0 bg-black/70 flex items-center justify-center z-50"
          onClick={() => setSelectedDevice(null)}
        >
          <div
            className="bg-slate-800 rounded-xl shadow-2xl max-w-lg w-full mx-4 border border-slate-700"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-3">
                  <span className="text-4xl">{getDeviceIcon(selectedDevice.type_guess)}</span>
                  <div>
                    <h2 className="text-2xl font-bold text-white">{selectedDevice.hostname || "Unknown Device"}</h2>
                    <p className="text-slate-400">{selectedDevice.vendor || "Unknown Vendor"}</p>
                  </div>
                </div>
                <button
                  onClick={() => setSelectedDevice(null)}
                  className="text-slate-400 hover:text-white text-2xl"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="bg-slate-700/50 p-3 rounded-lg">
                    <label className="text-xs text-slate-400 uppercase">IP Address</label>
                    <p className="font-mono font-medium text-white">{selectedDevice.ip || "N/A"}</p>
                  </div>
                  <div className="bg-slate-700/50 p-3 rounded-lg">
                    <label className="text-xs text-slate-400 uppercase">MAC Address</label>
                    <p className="font-mono text-sm text-white">{selectedDevice.mac}</p>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="bg-slate-700/50 p-3 rounded-lg">
                    <label className="text-xs text-slate-400 uppercase">Device Type</label>
                    <p className="font-medium capitalize text-white">{selectedDevice.type_guess || "Unknown"}</p>
                  </div>
                  <div className="bg-slate-700/50 p-3 rounded-lg">
                    <label className="text-xs text-slate-400 uppercase">Risk Score</label>
                    <p className="font-medium">
                      <span className={`px-2 py-1 rounded text-sm ${
                        selectedDevice.risk_score >= 70 ? "bg-red-900/50 text-red-300" :
                        selectedDevice.risk_score >= 40 ? "bg-yellow-900/50 text-yellow-300" :
                        "bg-emerald-900/50 text-emerald-300"
                      }`}>
                        {selectedDevice.risk_score}/100
                      </span>
                    </p>
                  </div>
                </div>

                <div className="bg-slate-700/50 p-3 rounded-lg">
                  <label className="text-xs text-slate-400 uppercase">Last Seen</label>
                  <p className="font-medium text-white">
                    {selectedDevice.last_seen ? new Date(selectedDevice.last_seen).toLocaleString() : "Never"}
                  </p>
                </div>

                <div className="pt-4 border-t border-slate-700 flex gap-2">
                  <button
                    onClick={() => {
                      router.push(`/devices/${selectedDevice.id}`);
                      setSelectedDevice(null);
                    }}
                    className="flex-1 btn btn-primary"
                  >
                    Full Details
                  </button>
                  <button
                    onClick={() => setSelectedDevice(null)}
                    className="btn btn-secondary"
                  >
                    Close
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
