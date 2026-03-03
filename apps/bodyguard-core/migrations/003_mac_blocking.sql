-- MAC-based Device Blocking for Parental Control
-- Allows parents to block/unblock devices by MAC address via Telegram bot

CREATE TABLE IF NOT EXISTS blocked_devices (
    id TEXT PRIMARY KEY,
    mac_address TEXT UNIQUE NOT NULL,
    device_name TEXT,
    blocked BOOLEAN DEFAULT 1,
    block_reason TEXT,
    blocked_at TEXT NOT NULL DEFAULT (datetime('now')),
    blocked_by TEXT,  -- 'telegram:userid' or 'web:username'
    unblocked_at TEXT,
    notes TEXT
);

-- Index for faster lookups by MAC address
CREATE INDEX IF NOT EXISTS idx_blocked_devices_mac ON blocked_devices(mac_address);

-- Index for active blocked devices
CREATE INDEX IF NOT EXISTS idx_blocked_devices_active ON blocked_devices(blocked) WHERE blocked = 1;

-- Audit log for MAC blocking actions
CREATE TABLE IF NOT EXISTS mac_block_audit (
    id TEXT PRIMARY KEY,
    mac_address TEXT NOT NULL,
    device_name TEXT,
    action TEXT NOT NULL CHECK(action IN ('blocked', 'unblocked', 'toggled')),
    performed_by TEXT NOT NULL,
    performed_at TEXT NOT NULL DEFAULT (datetime('now')),
    reason TEXT,
    source TEXT NOT NULL DEFAULT 'telegram' CHECK(source IN ('telegram', 'web', 'api'))
);
