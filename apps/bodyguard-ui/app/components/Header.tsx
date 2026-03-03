"use client";

import Link from "next/link";
import { use } from "react";
import { getSession, clearSession } from "@/lib/auth";

export function Header() {
  const handleLogout = async () => {
    await clearSession();
    window.location.href = "/login";
  };

  return (
    <header className="bg-white border-b border-gray-200 sticky top-0 z-50">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link href="/" className="flex items-center gap-3">
            <div className="w-10 h-10 flex items-center justify-center">
              <img src="/logo.png" alt="AmmanGate Logo" className="w-10 h-10 object-contain" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-gray-900">AmmanGate</h1>
              <p className="text-xs text-gray-500">AI Home Cyber Bodyguard</p>
            </div>
          </Link>

          {/* Navigation */}
          <nav className="flex items-center gap-6">
            <Link
              href="/"
              className="text-gray-600 hover:text-gray-900 font-medium transition-colors"
            >
              Dashboard
            </Link>
            <Link
              href="/devices"
              className="text-gray-600 hover:text-gray-900 font-medium transition-colors"
            >
              Devices
            </Link>
            <Link
              href="/alerts"
              className="text-gray-600 hover:text-gray-900 font-medium transition-colors"
            >
              Alerts Activities
            </Link>
            <Link
              href="/timeline"
              className="text-gray-600 hover:text-gray-900 font-medium transition-colors"
            >
              Timeline
            </Link>
            <Link
              href="/settings"
              className="text-gray-600 hover:text-gray-900 font-medium transition-colors"
            >
              Settings
            </Link>

            <div className="w-px h-6 bg-gray-300" />

            <button
              onClick={handleLogout}
              className="text-gray-600 hover:text-danger-600 font-medium transition-colors"
            >
              Logout
            </button>
          </nav>
        </div>
      </div>
    </header>
  );
}
