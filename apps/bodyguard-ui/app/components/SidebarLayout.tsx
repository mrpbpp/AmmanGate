"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { getSession, clearSession } from "@/lib/auth";
import { getCoreClient } from "@/lib/core";

interface NavItem {
  name: string;
  href: string;
  icon: string;
  badge?: number;
}

interface Notification {
  id: string;
  type: "honeypot" | "alert" | "info";
  title: string;
  message: string;
  timestamp: Date;
}

const navItems: NavItem[] = [
  { name: "Dashboard", href: "/", icon: "🏠" },
  { name: "Devices", href: "/devices", icon: "📱" },
  { name: "Parental Control", href: "/parental-control", icon: "👨‍👩‍👧‍👦" },
  { name: "Alerts", href: "/alerts", icon: "🚨" },
  { name: "Timeline", href: "/timeline", icon: "📊" },
  { name: "Settings", href: "/settings", icon: "⚙️" },
];

export function SidebarLayout({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [username, setUsername] = useState<string>("");
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const pathname = usePathname();

  useEffect(() => {
    // Get current user session
    getSession().then((session) => {
      if (session?.username) {
        setUsername(session.username);
      }
    });

    // Connect to WebSocket for real-time updates
    connectWebSocket();

    // Remove old notifications (auto-dismiss after 10 seconds)
    const interval = setInterval(() => {
      setNotifications((prev) =>
        prev.filter((n) => Date.now() - n.timestamp.getTime() < 10000)
      );
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const connectWebSocket = () => {
    try {
      const client = getCoreClient();
      const ws = client.connectWebSocket();

      ws.onopen = () => {
        console.log("Notification WebSocket connected");
      };

      ws.onmessage = (event) => {
        try {
          const wsMessage = JSON.parse(event.data);
          console.log("WS notification:", wsMessage);

          // Handle honeypot hit notification
          if (wsMessage.type === "honeypot_hit" && wsMessage.data) {
            const data = wsMessage.data;
            let message = `Connection from ${data.remote_ip || "unknown"} to port ${data.port || "?"} (${data.service || "unknown"})`;

            // Add geolocation info if available
            if (data.geo_location) {
              const geo = data.geo_location;
              message += ` - ${geo.formatted || "Unknown Location"}`;
              if (geo.is_risky) {
                message += " ⚠️ VPN/Proxy/Tor";
              }
            }

            addNotification({
              id: `notif-${Date.now()}`,
              type: "honeypot",
              title: "🎯 Honeypot Alert!",
              message: message,
              timestamp: new Date(),
            });
          }
        } catch (e) {
          console.error("Failed to parse WS notification:", e);
        }
      };
    } catch (error) {
      console.error("Failed to connect notification WebSocket:", error);
    }
  };

  const addNotification = (notification: Notification) => {
    setNotifications((prev) => [notification, ...prev].slice(0, 5));

    // Also show browser notification if permission granted
    if ("Notification" in window && Notification.permission === "granted") {
      new Notification(notification.title, {
        body: notification.message,
        icon: "/logo.png",
      });
    }
  };

  const handleLogout = async () => {
    await clearSession();
    window.location.href = "/login";
  };

  // Request notification permission on first user interaction
  useEffect(() => {
    const handleUserInteraction = () => {
      if ("Notification" in window && Notification.permission === "default") {
        Notification.requestPermission();
      }
      document.removeEventListener("click", handleUserInteraction);
    };

    document.addEventListener("click", handleUserInteraction);
    return () => document.removeEventListener("click", handleUserInteraction);
  }, []);

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Notifications */}
      <div className="fixed top-4 right-4 z-50 space-y-2 max-w-sm">
        {notifications.map((notif) => (
          <div
            key={notif.id}
            className={`${
              notif.type === "honeypot"
                ? "bg-red-900/90 border-red-500"
                : notif.type === "alert"
                ? "bg-yellow-900/90 border-yellow-500"
                : "bg-blue-900/90 border-blue-500"
            } backdrop-blur-lg border-l-4 rounded-lg shadow-2xl p-4 animate-pulse-glow`}
          >
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0">
                {notif.type === "honeypot" && "🎯"}
                {notif.type === "alert" && "🚨"}
                {notif.type === "info" && "ℹ️"}
              </div>
              <div className="flex-1">
                <p className="font-semibold text-white text-sm">{notif.title}</p>
                <p className="text-slate-200 text-xs mt-1">{notif.message}</p>
              </div>
              <button
                onClick={() =>
                  setNotifications((prev) => prev.filter((n) => n.id !== notif.id))
                }
                className="text-slate-400 hover:text-white flex-shrink-0"
              >
                ✕
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* Mobile sidebar backdrop */}
      {mobileSidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
          onClick={() => setMobileSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`
          fixed top-0 left-0 z-50 h-full transition-transform duration-300 ease-in-out
          ${sidebarOpen ? "w-64" : "w-20"}
          ${mobileSidebarOpen ? "translate-x-0" : "-translate-x-full"}
          lg:translate-x-0
          bg-gradient-to-b from-slate-800 to-slate-900
          border-r border-slate-700
          flex flex-col
        `}
      >
        {/* Logo */}
        <div className="flex items-center justify-between h-16 px-4 border-b border-slate-700">
          <Link href="/" className="flex items-center gap-3">
            <div className="w-10 h-10 flex items-center justify-center flex-shrink-0">
              <img src="/logo.png" alt="AmmanGate" className="w-10 h-10 object-contain" />
            </div>
            {sidebarOpen && (
              <div className="overflow-hidden">
                <h1 className="text-lg font-bold text-white">AmmanGate</h1>
                <p className="text-xs text-slate-400">Security Console</p>
              </div>
            )}
          </Link>
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="hidden lg:flex p-2 rounded-lg hover:bg-slate-700 text-slate-400 hover:text-white transition-colors"
          >
            <svg
              className={`w-5 h-5 transition-transform ${sidebarOpen ? "rotate-180" : ""}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
            </svg>
          </button>
          <button
            onClick={() => setMobileSidebarOpen(false)}
            className="lg:hidden p-2 rounded-lg hover:bg-slate-700 text-slate-400 hover:text-white"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
          {navItems.map((item) => {
            const isActive = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`
                  flex items-center gap-3 px-3 py-3 rounded-lg transition-all duration-200
                  ${isActive
                    ? "bg-blue-600 text-white shadow-lg shadow-blue-600/30"
                    : "text-slate-300 hover:bg-slate-800 hover:text-white"
                  }
                `}
              >
                <span className="text-xl flex-shrink-0">{item.icon}</span>
                {sidebarOpen && (
                  <span className="font-medium truncate">{item.name}</span>
                )}
                {item.badge && sidebarOpen && (
                  <span className="ml-auto bg-red-500 text-white text-xs font-bold px-2 py-0.5 rounded-full animate-pulse">
                    {item.badge}
                  </span>
                )}
              </Link>
            );
          })}
        </nav>

        {/* User section */}
        <div className="p-4 border-t border-slate-700">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white font-bold flex-shrink-0">
              {username.charAt(0).toUpperCase()}
            </div>
            {sidebarOpen && (
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-white truncate">{username}</p>
                <p className="text-xs text-slate-400">Administrator</p>
              </div>
            )}
            <button
              onClick={handleLogout}
              className="p-2 rounded-lg hover:bg-slate-700 text-slate-400 hover:text-red-400 transition-colors flex-shrink-0"
              title="Logout"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
            </button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div
        className={`
          transition-all duration-300 ease-in-out
          ${sidebarOpen ? "lg:ml-64" : "lg:ml-20"}
        `}
      >
        {/* Mobile header */}
        <header className="lg:hidden flex items-center justify-between h-16 px-4 bg-slate-800 border-b border-slate-700">
          <Link href="/" className="flex items-center gap-3">
            <div className="w-8 h-8 flex items-center justify-center">
              <img src="/logo.png" alt="AmmanGate" className="w-8 h-8 object-contain" />
            </div>
            <span className="text-lg font-bold text-white">AmmanGate</span>
          </Link>
          <button
            onClick={() => setMobileSidebarOpen(true)}
            className="p-2 rounded-lg hover:bg-slate-700 text-slate-400 hover:text-white"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        </header>

        {/* Page content */}
        <main className="p-4 lg:p-6">{children}</main>
      </div>
    </div>
  );
}
