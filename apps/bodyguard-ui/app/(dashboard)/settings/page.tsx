"use client";

import UserManagement from './components/UserManagement';
import UserProfileCard from './components/UserProfileCard';

export default function SettingsPage() {
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

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h2 className="text-xl font-semibold text-white mb-4">Security Settings</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-white">Honeypot</p>
                <p className="text-sm text-slate-400">Run decoy services to detect attackers</p>
              </div>
              <button className="btn btn-primary text-sm">Enable</button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-white">Port Scanning Detection</p>
                <p className="text-sm text-slate-400">Detect network scanning behavior</p>
              </div>
              <button className="btn btn-primary text-sm">Enable</button>
            </div>
          </div>
        </div>

        <div className="card">
          <h2 className="text-xl font-semibold text-white mb-4">Scanning</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-white">Vulnerability Scanner</p>
                <p className="text-sm text-slate-400">Scan devices for vulnerabilities</p>
              </div>
              <button className="btn btn-secondary text-sm">Configure</button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-white">Malware Scanning</p>
                <p className="text-sm text-slate-400">Scan traffic with ClamAV (coming soon)</p>
              </div>
              <button className="btn btn-secondary text-sm disabled" disabled>
                Coming Soon
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="card">
        <h2 className="text-xl font-semibold text-white mb-4">System Information</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <p className="text-sm text-slate-400">Version</p>
            <p className="text-white font-medium">0.1.0</p>
          </div>
          <div>
            <p className="text-sm text-slate-400">Backend</p>
            <p className="text-white font-medium">Connected</p>
          </div>
          <div>
            <p className="text-sm text-slate-400">Honeypot Ports</p>
            <p className="text-white font-medium">8 Active</p>
          </div>
        </div>
      </div>
    </div>
  );
}
