"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getCoreClient, type Device } from "@/lib/core";

export default function DevicesPage() {
  const router = useRouter();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");

  const client = getCoreClient();

  useEffect(() => {
    loadDevices();
  }, []);

  const loadDevices = async () => {
    setLoading(true);
    try {
      const data = await client.getDevices({ limit: 100 });
      setDevices(data.items ?? []);
    } catch (error) {
      console.error("Failed to load devices:", error);
    } finally {
      setLoading(false);
    }
  };

  const filteredDevices = devices.filter(
    (d) =>
      d.hostname.toLowerCase().includes(searchQuery.toLowerCase()) ||
      d.ip.includes(searchQuery) ||
      d.mac.toLowerCase().includes(searchQuery.toLowerCase()) ||
      d.vendor.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const getRiskColor = (score: number) => {
    if (score >= 70) return "text-red-400";
    if (score >= 40) return "text-yellow-400";
    if (score >= 20) return "text-orange-400";
    return "text-emerald-400";
  };

  const getRiskLabel = (score: number) => {
    if (score >= 70) return "Critical";
    if (score >= 40) return "High";
    if (score >= 20) return "Medium";
    return "Low";
  };

  const getDeviceIcon = (type: string) => {
    const icons: Record<string, string> = {
      mobile: "📱",
      laptop: "💻",
      desktop: "🖥️",
      tablet: "📟",
      iot: "🔌",
      router: "📡",
      unknown: "📦",
    };
    return icons[type] || "📦";
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white">Devices</h1>
          <p className="text-slate-400 mt-1">All devices on your network</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-slate-400">Total Devices</p>
              <p className="text-2xl font-bold text-white">{devices.length}</p>
            </div>
            <div className="text-3xl">📱</div>
          </div>
        </div>
        <div className="card border-l-4 border-red-500">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-slate-400">High Risk</p>
              <p className="text-2xl font-bold text-red-400">
                {devices.filter((d) => d.risk_score >= 40).length}
              </p>
            </div>
            <div className="text-3xl">⚠️</div>
          </div>
        </div>
        <div className="card border-l-4 border-emerald-500">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-slate-400">Safe</p>
              <p className="text-2xl font-bold text-emerald-400">
                {devices.filter((d) => d.risk_score < 40).length}
              </p>
            </div>
            <div className="text-3xl">✅</div>
          </div>
        </div>
      </div>

      {/* Search */}
      <div className="card">
        <input
          type="text"
          placeholder="Search by hostname, IP, MAC, or vendor..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="input w-full"
        />
      </div>

      {/* Device List */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : filteredDevices.length === 0 ? (
        <div className="card text-center py-12">
          <div className="text-6xl mb-4">📭</div>
          <p className="text-slate-400">No devices found</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredDevices.map((device) => (
            <div
              key={device.id}
              onClick={() => router.push(`/devices/${device.id}`)}
              className="card hover:border-blue-500 transition-all cursor-pointer"
            >
              <div className="flex items-start gap-4">
                <div className="text-4xl">{getDeviceIcon(device.type_guess)}</div>
                <div className="flex-1 min-w-0">
                  <h3 className="text-white font-semibold truncate">{device.hostname || "Unknown"}</h3>
                  <p className="text-sm text-slate-400 truncate">{device.vendor}</p>

                  <div className="mt-3 space-y-1">
                    <p className="text-xs text-slate-400">
                      <span className="font-mono">{device.ip}</span>
                    </p>
                    <p className="text-xs text-slate-500 font-mono">
                      {device.mac}
                    </p>
                  </div>

                  <div className="mt-3 flex items-center justify-between">
                    <span className={`text-sm font-medium ${getRiskColor(device.risk_score)}`}>
                      {getRiskLabel(device.risk_score)}
                    </span>
                    <span className="text-xs text-slate-400">
                      Risk: {device.risk_score}/100
                    </span>
                  </div>

                  <div className="mt-3 w-full bg-slate-700 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full ${
                        device.risk_score >= 70
                          ? "bg-red-500"
                          : device.risk_score >= 40
                          ? "bg-yellow-500"
                          : "bg-emerald-500"
                      }`}
                      style={{ width: `${device.risk_score}%` }}
                    ></div>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
