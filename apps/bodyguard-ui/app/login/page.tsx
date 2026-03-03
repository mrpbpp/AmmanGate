"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";

// Force dynamic rendering
export const dynamic = 'force-dynamic';

export default function LoginPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      console.log("[LOGIN PAGE] Attempting login for:", username);
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      console.log("[LOGIN PAGE] Response status:", response.status);
      const data = await response.json();
      console.log("[LOGIN PAGE] Response data:", data);

      if (!response.ok) {
        setError(data.error || "Login failed");
        console.error("[LOGIN PAGE] Login failed:", data.error);
        return;
      }

      console.log("[LOGIN PAGE] Login successful, redirecting...");
      const redirect = searchParams.get("redirect") || "/";
      router.push(redirect);
      router.refresh();
    } catch (err) {
      console.error("[LOGIN PAGE] Error:", err);
      setError("Connection error. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-slate-800 to-blue-900 px-4">
      {/* Background effects */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-blue-500/20 rounded-full blur-3xl"></div>
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-purple-500/20 rounded-full blur-3xl"></div>
      </div>

      <div className="max-w-md w-full relative z-10">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-24 h-24 bg-slate-800/80 backdrop-blur rounded-2xl shadow-2xl p-3 border border-slate-700">
            <img src="/logo.png" alt="AmmanGate Logo" className="w-full h-full object-contain" />
          </div>
          <h1 className="mt-6 text-3xl font-bold text-white">AmmanGate</h1>
          <p className="text-slate-400">AI Home Cyber Bodyguard</p>
        </div>

        {/* Login Form */}
        <div className="bg-slate-800/80 backdrop-blur-xl rounded-2xl shadow-2xl border border-slate-700 p-8">
          <h2 className="text-xl font-semibold text-white mb-6">Sign In</h2>

          {error && (
            <div className="mb-4 p-3 bg-red-900/50 border border-red-700 rounded-lg">
              <p className="text-sm text-red-300">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="username" className="block text-sm font-medium text-slate-300 mb-1">
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="input"
                placeholder="admin"
                required
                autoComplete="username"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-1">
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="input"
                placeholder="••••••••"
                required
                autoComplete="current-password"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? "Signing in..." : "Sign In"}
            </button>
          </form>

          {/* Default credentials hint */}
          <div className="mt-6 p-4 bg-slate-700/50 rounded-lg border border-slate-600">
            <p className="text-xs text-slate-400">
              <strong className="text-slate-300">Default credentials:</strong> admin / admin123
              <br />
              <span className="text-yellow-400">⚠️ Please change after first login</span>
            </p>
          </div>
        </div>

        <p className="text-center text-sm text-slate-500 mt-6">
          Protected by AmmanGate Security Engine
        </p>
      </div>
    </div>
  );
}
