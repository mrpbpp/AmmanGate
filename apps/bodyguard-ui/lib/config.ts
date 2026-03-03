// AmmanGate Configuration
// Reads from environment variables with sensible defaults

export const config = {
  // Backend API URL
  // In Docker: use container name
  // In development: use localhost
  backendApi: process.env.NEXT_PUBLIC_BACKENED_API || 'http://127.0.0.1:8787/v1',

  // WebSocket URL
  wsUrl: process.env.NEXT_PUBLIC_WS_URL || 'ws://127.0.0.1:8787/v1/ws',

  // App info
  appName: 'AmmanGate',
  version: process.env.NEXT_PUBLIC_APP_VERSION || '0.1.0',

  // Feature flags
  features: {
    telegram: process.env.NEXT_PUBLIC_TELEGRAM_ENABLED === 'true',
    websocket: true,
    suricata: true,
    clamav: true,
  },

  // Timeouts
  timeouts: {
    api: 30000, // 30 seconds
    ws: 60000,  // 1 minute
  },
} as const;

// Type for the config
export type Config = typeof config;
