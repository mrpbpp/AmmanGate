"use client";

import { useEffect, useState } from "react";
import UserProfileCard from './components/UserProfileCard';
import UserManagement from './components/UserManagement';

interface ServiceStatus {
  enabled: boolean;
  running?: boolean;
  version?: string;
  [key: string]: any;
}

interface Service {
  id: string;
  name: string;
  description: string;
  icon: string;
  endpoint: string;
  status: ServiceStatus | null;
  loading: boolean;
}

const SERVICES: Omit<Service, "status" | "loading">[] = [
  {
    id: "dns",
    name: "DNS Server",
    description: "DNS filtering for parental control and threat blocking",
    icon: "🌐",
    endpoint: "/dns",
  },
  {
    id: "clamav",
    name: "ClamAV Antivirus",
    description: "Real-time malware and virus scanning",
    icon: "🦠",
    endpoint: "/clamav",
  },
  {
    id: "suricata",
    name: "Suricata IDS",
    description: "Intrusion Detection System for network threats",
    icon: "🛡️",
    endpoint: "/suricata",
  },
  {
    id: "ai",
    name: "AI Analysis",
    description: "AI-powered security analysis and threat detection",
    icon: "🤖",
    endpoint: "/ai",
  },
];

export default function SettingsPage() {
  const [services, setServices] = useState<Service[]>(
    SERVICES.map((s) => ({ ...s, status: null, loading: true }))
  );

  const API_BASE = process.env.NEXT_PUBLIC_CORE_API || "http://127.0.0.1:8787/v1";

  // Fetch service status
  const fetchServiceStatus = async (serviceId: string) => {
    setServices((prev) =>
      prev.map((s) =>
        s.id === serviceId ? { ...s, loading: true } : s
      )
    );

    try {
      const response = await fetch(`${API_BASE}${services.find((s) => s.id === serviceId)!.endpoint}/status`);
      if (response.ok) {
        const data = await response.json();
        setServices((prev) =>
          prev.map((s) =>
            s.id === serviceId ? { ...s, status: data, loading: false } : s
          )
        );
      } else {
        setServices((prev) =>
          prev.map((s) =>
            s.id === serviceId ? { ...s, status: { enabled: false }, loading: false } : s
          )
        );
      }
    } catch (error) {
      console.error(`Error fetching ${serviceId} status:`, error);
      setServices((prev) =>
        prev.map((s) =>
          s.id === serviceId ? { ...s, status: { enabled: false }, loading: false } : s
        )
      );
    }
  };

  // Toggle service
  const toggleService = async (serviceId: string, enabled: boolean) => {
    const service = services.find((s) => s.id === serviceId);
    if (!service) return;

    setServices((prev) =>
      prev.map((s) =>
        s.id === serviceId ? { ...s, loading: true } : s
      )
    );

    try {
      const response = await fetch(`${API_BASE}${service.endpoint}/toggle`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ enabled }),
      });

      if (response.ok) {
        const data = await response.json();
        setServices((prev) =>
          prev.map((s) =>
            s.id === serviceId
              ? { ...s, status: { ...s.status, enabled: data.enabled, running: data.running }, loading: false }
              : s
          )
        );
      } else {
        // Revert on error
        await fetchServiceStatus(serviceId);
      }
    } catch (error) {
      console.error(`Error toggling ${serviceId}:`, error);
      await fetchServiceStatus(serviceId);
    }
  };

  // Fetch all service statuses on mount
  useEffect(() => {
    services.forEach((s) => fetchServiceStatus(s.id));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-white">Settings</h1>
        <p className="text-slate-400 mt-1">Configure your AmmanGate system</p>
      </div>

      {/* User Profile */}
      <UserProfileCard />

      {/* User Management (Admin only) */}
      <UserManagement />

      {/* Services Configuration */}
      <div>
        <h2 className="text-xl font-semibold text-white mb-4">Services</h2>
        <p className="text-slate-400 text-sm mb-4">Manage security modules and services</p>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {services.map((service) => {
            const isEnabled = service.status?.enabled ?? false;
            const isRunning = service.status?.running ?? false;

            return (
              <div key={service.id} className="card">
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3">
                    <div className="text-3xl">{service.icon}</div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="text-base font-semibold text-white">{service.name}</h3>
                        {service.loading && (
                          <div className="w-3 h-3 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                        )}
                      </div>
                      <p className="text-xs text-slate-400 mt-1">{service.description}</p>

                      {/* Status indicators */}
                      <div className="flex items-center gap-3 mt-2">
                        <div className="flex items-center gap-1.5">
                          <div className={`w-1.5 h-1.5 rounded-full ${isEnabled ? "bg-green-500" : "bg-slate-500"}`} />
                          <span className="text-xs text-slate-400">
                            {isEnabled ? "Enabled" : "Disabled"}
                          </span>
                        </div>
                        {service.status?.running !== undefined && (
                          <div className="flex items-center gap-1.5">
                            <div className={`w-1.5 h-1.5 rounded-full ${isRunning ? "bg-blue-500" : "bg-slate-600"}`} />
                            <span className="text-xs text-slate-400">
                              {isRunning ? "Running" : "Stopped"}
                            </span>
                          </div>
                        )}
                      </div>

                      {/* Version info if available */}
                      {service.status?.version && (
                        <p className="text-xs text-slate-500 mt-1">
                          v{service.status.version}
                        </p>
                      )}
                    </div>
                  </div>

                  {/* Toggle Switch */}
                  <button
                    onClick={() => toggleService(service.id, !isEnabled)}
                    disabled={service.loading}
                    className={`
                      relative inline-flex h-10 w-16 items-center rounded-full transition-colors duration-200
                      ${isEnabled ? "bg-blue-600" : "bg-slate-700"}
                      ${service.loading ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}
                    `}
                  >
                    <span
                      className={`
                        inline-block h-8 w-8 transform rounded-full bg-white shadow-lg transition-transform duration-200
                        ${isEnabled ? "translate-x-7" : "translate-x-0.5"}
                      `}
                    />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Service Description Card */}
      <div className="card">
        <h2 className="text-lg font-semibold text-white mb-3">About Services</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-slate-400">
          <div>
            <span className="text-white font-medium">🌐 DNS Server:</span> Filters DNS queries to block malicious domains and enforce parental control rules.
          </div>
          <div>
            <span className="text-white font-medium">🦠 ClamAV Antivirus:</span> Scans files and traffic for malware, viruses, and other threats.
          </div>
          <div>
            <span className="text-white font-medium">🛡️ Suricata IDS:</span> Monitors network traffic for intrusion attempts, exploits, and suspicious patterns.
          </div>
          <div>
            <span className="text-white font-medium">🤖 AI Analysis:</span> Uses artificial intelligence to analyze security events and provide insights.
          </div>
        </div>
      </div>
    </div>
  );
}
