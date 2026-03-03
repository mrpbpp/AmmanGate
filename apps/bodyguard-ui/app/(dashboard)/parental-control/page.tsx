"use client";

import { useState, useEffect } from "react";
import { getCoreClient, FilterRule, DNSQueryLog, ClamAVStatus } from "@/lib/core";

export default function ParentalControlPage() {
  const [filters, setFilters] = useState<FilterRule[]>([]);
  const [dnsLogs, setDnsLogs] = useState<DNSQueryLog[]>([]);
  const [networkInfo, setNetworkInfo] = useState<{
    hostname: string;
    ips: string[];
    dns_port: number;
    web_port: number;
    primary_ip: string;
    dns_running: boolean;
  } | null>(null);
  const [clamavStatus, setClamavStatus] = useState<ClamAVStatus | null>(null);
  const [dnsServerRunning, setDnsServerRunning] = useState(false);
  const [loading, setLoading] = useState(true);
  const [showAddRule, setShowAddRule] = useState(false);
  const [newRule, setNewRule] = useState({ name: "", type: "domain", pattern: "" });

  useEffect(() => {
    loadFilters();
    loadDNSLogs();
    loadNetworkInfo();
    loadClamAVStatus();
  }, []);

  const loadFilters = async () => {
    try {
      const client = getCoreClient();
      const data = await client.getFilters();
      setFilters(data.items || []);
    } catch (error) {
      console.error("Failed to load filters:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadNetworkInfo = async () => {
    try {
      const client = getCoreClient();
      const data = await client.getNetworkInfo();
      setNetworkInfo(data);
      setDnsServerRunning(data.dns_running || false);
    } catch (error) {
      console.error("Failed to load network info:", error);
      // Set default values on error
      setNetworkInfo({
        hostname: "Unknown",
        ips: ["127.0.0.1"],
        dns_port: 53,
        web_port: 8787,
        primary_ip: "127.0.0.1",
        dns_running: false,
      });
      setDnsServerRunning(false);
    }
  };

  const loadDNSLogs = async () => {
    try {
      const client = getCoreClient();
      const data = await client.getDNSLogs(50);
      setDnsLogs(data.items || []);
    } catch (error) {
      console.error("Failed to load DNS logs:", error);
    }
  };

  const loadClamAVStatus = async () => {
    try {
      const client = getCoreClient();
      const data = await client.getClamAVStatus();
      setClamavStatus(data);
    } catch (error) {
      console.error("Failed to load ClamAV status:", error);
      setClamavStatus({
        enabled: false,
        running: false,
        version: "Error",
        db_version: "N/A",
        address: "localhost:3310",
        last_check: new Date().toISOString(),
      });
    }
  };

  const addRule = async () => {
    if (!newRule.name || !newRule.pattern) return;

    try {
      const client = getCoreClient();
      await client.addFilter(newRule);
      setNewRule({ name: "", type: "domain", pattern: "" });
      setShowAddRule(false);
      loadFilters();
    } catch (error) {
      console.error("Failed to add rule:", error);
    }
  };

  const toggleRule = async (id: string, enabled: boolean) => {
    try {
      const client = getCoreClient();
      await client.toggleFilter(id, enabled);
      loadFilters();
    } catch (error) {
      console.error("Failed to toggle rule:", error);
    }
  };

  const deleteRule = async (id: string) => {
    if (!confirm("Are you sure you want to delete this rule?")) return;

    try {
      const client = getCoreClient();
      await client.deleteFilter(id);
      loadFilters();
    } catch (error) {
      console.error("Failed to delete rule:", error);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-white">Parental Control</h1>
        <p className="text-slate-400 mt-1">
          Manage content filtering and protect your family from inappropriate content
        </p>
      </div>

      {/* Router Setup Notice - CRITICAL */}
      <div className="card bg-gradient-to-r from-blue-900/50 to-purple-900/50 border-l-4 border-blue-500">
        <div className="flex items-start gap-4">
          <div className="text-4xl">⚠️</div>
          <div className="flex-1">
            <h2 className="text-xl font-bold text-white mb-2">
              Router Configuration Required
            </h2>
            <p className="text-slate-300 mb-4">
              For Parental Control to work, you must configure your home router to use AmmanGate as your DNS server.
            </p>

            {/* DNS Server Status */}
            <div className="bg-slate-800/50 rounded-lg p-4 mb-4">
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-semibold text-white">DNS Server Status</h3>
                <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium ${
                  dnsServerRunning
                    ? "bg-green-500/20 text-green-400"
                    : "bg-red-500/20 text-red-400"
                }`}>
                  <span className={`w-2 h-2 rounded-full ${
                    dnsServerRunning ? "bg-green-400 animate-pulse" : "bg-red-400"
                  }`}></span>
                  {dnsServerRunning ? "Running" : "Stopped"}
                </div>
              </div>

              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div>
                  <p className="text-slate-400">Hostname</p>
                  <p className="text-white font-medium">{networkInfo?.hostname || "Loading..."}</p>
                </div>
                <div>
                  <p className="text-slate-400">Primary IP</p>
                  <p className="text-white font-medium">{networkInfo?.primary_ip || "Loading..."}</p>
                </div>
                <div>
                  <p className="text-slate-400">DNS Port</p>
                  <p className="text-white font-medium">{networkInfo?.dns_port || 53}</p>
                </div>
                <div>
                  <p className="text-slate-400">Web Port</p>
                  <p className="text-white font-medium">{networkInfo?.web_port || 8787}</p>
                </div>
              </div>

              {networkInfo && networkInfo.ips.length > 1 && (
                <details className="mt-3">
                  <summary className="cursor-pointer text-blue-400 hover:text-blue-300 text-sm">
                    View all IP addresses
                  </summary>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {networkInfo.ips.map((ip, idx) => (
                      <code key={idx} className="bg-slate-700 px-2 py-1 rounded text-white text-sm">
                        {ip}
                      </code>
                    ))}
                  </div>
                </details>
              )}
            </div>

            <div className="bg-slate-800/50 rounded-lg p-4 space-y-3">
              <h3 className="font-semibold text-white">Setup Instructions:</h3>
              <ol className="list-decimal list-inside space-y-2 text-slate-300 text-sm">
                <li>Access your router's admin panel (usually 192.168.1.1 or 192.168.0.1)</li>
                <li>Find <strong className="text-white">DNS Settings</strong> or <strong className="text-white">DHCP Settings</strong></li>
                <li>Change the DNS server to your AmmanGate IP: <code className="bg-blue-600/30 text-blue-300 px-2 py-1 rounded">{networkInfo?.primary_ip || "192.168.1.X"}</code></li>
                <li>Save and restart your router</li>
              </ol>

              <details className="mt-3">
                <summary className="cursor-pointer text-blue-400 hover:text-blue-300">
                  View Router-Specific Instructions
                </summary>
                <div className="mt-3 space-y-2 text-sm">
                  <div className="bg-slate-700/50 rounded p-3">
                    <strong className="text-white">TP-Link:</strong> Advanced → DHCP → DHCP Settings → Primary DNS
                  </div>
                  <div className="bg-slate-700/50 rounded p-3">
                    <strong className="text-white">Asus:</strong> LAN → DHCP Server → DNS Server 1
                  </div>
                  <div className="bg-slate-700/50 rounded p-3">
                    <strong className="text-white">Netgear:</strong> Advanced → Setup → DNS Settings
                  </div>
                  <div className="bg-slate-700/50 rounded p-3">
                    <strong className="text-white">D-Link:</strong> Setup → Network Settings → DNS
                  </div>
                </div>
              </details>
            </div>
          </div>
        </div>
      </div>

      {/* ClamAV Status */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-white">ClamAV Antivirus</h2>
          <div className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium ${
            clamavStatus?.running
              ? "bg-green-500/20 text-green-400"
              : clamavStatus?.enabled
              ? "bg-yellow-500/20 text-yellow-400"
              : "bg-red-500/20 text-red-400"
          }`}>
            <span className={`w-2 h-2 rounded-full ${
              clamavStatus?.running
                ? "bg-green-400 animate-pulse"
                : clamavStatus?.enabled
                ? "bg-yellow-400"
                : "bg-red-400"
            }`}></span>
            {clamavStatus?.running
              ? "Running"
              : clamavStatus?.enabled
              ? "Enabled - Not Connected"
              : "Disabled"}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-slate-800/50 rounded-lg p-4">
            <p className="text-slate-400 text-sm">Status</p>
            <p className={`text-lg font-semibold ${
              clamavStatus?.running ? "text-green-400" : "text-yellow-400"
            }`}>
              {clamavStatus?.running ? "🟢 Active" : "🟡 Inactive"}
            </p>
          </div>
          <div className="bg-slate-800/50 rounded-lg p-4">
            <p className="text-slate-400 text-sm">ClamAV Version</p>
            <p className="text-white font-medium">{clamavStatus?.version || "N/A"}</p>
          </div>
          <div className="bg-slate-800/50 rounded-lg p-4">
            <p className="text-slate-400 text-sm">Database Version</p>
            <p className="text-white font-medium">{clamavStatus?.db_version || "N/A"}</p>
          </div>
        </div>

        {!clamavStatus?.running && (
          <div className="mt-4 bg-blue-900/30 border border-blue-500/30 rounded-lg p-4">
            <h3 className="font-semibold text-blue-400 mb-2">📋 ClamAV Setup Required</h3>
            <p className="text-slate-300 text-sm mb-3">
              ClamAV is configured but the daemon is not running. Install and start ClamAV to enable malware scanning.
            </p>
            <ol className="list-decimal list-inside space-y-1 text-slate-300 text-sm">
              <li>Download ClamAV for Windows from <a href="https://www.clamav.net/downloads" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:underline">clamav.net</a></li>
              <li>Install ClamAV and start the clamd service</li>
              <li>Ensure clamd is listening on port 3310 (or update CLAMAV_ADDRESS in .env)</li>
            </ol>
          </div>
        )}
      </div>

      {/* Filter Rules */}
      <div className="card">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-white">Filter Rules</h2>
          <button
            onClick={() => setShowAddRule(!showAddRule)}
            className="btn btn-primary"
          >
            + Add Rule
          </button>
        </div>

        {showAddRule && (
          <div className="bg-slate-800 rounded-lg p-4 mb-4 space-y-4">
            <h3 className="font-semibold text-white">Add New Filter Rule</h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm text-slate-400 mb-1">Rule Name</label>
                <input
                  type="text"
                  value={newRule.name}
                  onChange={(e) => setNewRule({ ...newRule, name: e.target.value })}
                  placeholder="e.g., Adult Content"
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-400 mb-1">Type</label>
                <select
                  value={newRule.type}
                  onChange={(e) => setNewRule({ ...newRule, type: e.target.value })}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-blue-500"
                >
                  <option value="domain">Domain</option>
                  <option value="category">Category</option>
                </select>
              </div>
              <div>
                <label className="block text-sm text-slate-400 mb-1">Pattern</label>
                <input
                  type="text"
                  value={newRule.pattern}
                  onChange={(e) => setNewRule({ ...newRule, pattern: e.target.value })}
                  placeholder={newRule.type === "domain" ? "example.com" : "adult,porn,xxx"}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>
            <div className="flex gap-2">
              <button onClick={addRule} className="btn btn-primary">
                Add Rule
              </button>
              <button
                onClick={() => setShowAddRule(false)}
                className="btn btn-secondary"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8 text-slate-400">Loading filters...</div>
        ) : filters.length === 0 ? (
          <div className="text-center py-8 text-slate-400">
            No filter rules configured. Add a rule to get started.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Name</th>
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Type</th>
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Pattern</th>
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Status</th>
                  <th className="text-right py-3 px-4 text-slate-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filters.map((filter) => (
                  <tr key={filter.id} className="border-b border-slate-700/50">
                    <td className="py-3 px-4 text-white">{filter.name}</td>
                    <td className="py-3 px-4">
                      <span className={`px-2 py-1 rounded text-xs ${
                        filter.type === "domain"
                          ? "bg-blue-500/20 text-blue-400"
                          : "bg-purple-500/20 text-purple-400"
                      }`}>
                        {filter.type}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-slate-300 font-mono text-sm">
                      {filter.pattern}
                    </td>
                    <td className="py-3 px-4">
                      <button
                        onClick={() => toggleRule(filter.id, !filter.enabled)}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                          filter.enabled ? "bg-blue-600" : "bg-slate-600"
                        }`}
                      >
                        <span
                          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                            filter.enabled ? "translate-x-6" : "translate-x-1"
                          }`}
                        />
                      </button>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <button
                        onClick={() => deleteRule(filter.id)}
                        className="text-red-400 hover:text-red-300 text-sm"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* DNS Query Logs */}
      <div className="card">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-white">Recent DNS Queries</h2>
          <button onClick={loadDNSLogs} className="btn btn-secondary text-sm">
            Refresh
          </button>
        </div>

        {dnsLogs.length === 0 ? (
          <div className="text-center py-8 text-slate-400">
            No DNS queries logged yet. Make sure your router is configured to use AmmanGate DNS.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Domain</th>
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Status</th>
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Time</th>
                </tr>
              </thead>
              <tbody>
                {dnsLogs.map((log) => (
                  <tr key={log.id} className="border-b border-slate-700/50">
                    <td className="py-3 px-4 text-white font-mono text-sm">{log.domain}</td>
                    <td className="py-3 px-4">
                      {log.blocked ? (
                        <span className="px-2 py-1 rounded text-xs bg-red-500/20 text-red-400">
                          Blocked
                        </span>
                      ) : (
                        <span className="px-2 py-1 rounded text-xs bg-green-500/20 text-green-400">
                          Allowed
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-slate-400 text-sm">
                      {new Date(log.ts).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* AI Commands Info */}
      <div className="card bg-gradient-to-r from-green-900/30 to-blue-900/30">
        <h2 className="text-xl font-semibold text-white mb-4">🤖 AI Commands</h2>
        <p className="text-slate-300 mb-3">
          You can also manage Parental Control using AI chat commands:
        </p>
        <ul className="space-y-2 text-slate-400 text-sm">
          <li className="flex items-center gap-2">
            <span className="text-blue-400">•</span>
            <code className="bg-slate-800 px-2 py-1 rounded">"Set parental control to strict for my kids' devices"</code>
          </li>
          <li className="flex items-center gap-2">
            <span className="text-blue-400">•</span>
            <code className="bg-slate-800 px-2 py-1 rounded">"Block gambling websites"</code>
          </li>
          <li className="flex items-center gap-2">
            <span className="text-blue-400">•</span>
            <code className="bg-slate-800 px-2 py-1 rounded">"Block domain example.com"</code>
          </li>
          <li className="flex items-center gap-2">
            <span className="text-blue-400">•</span>
            <code className="bg-slate-800 px-2 py-1 rounded">"Show my parental control rules"</code>
          </li>
        </ul>
      </div>
    </div>
  );
}
