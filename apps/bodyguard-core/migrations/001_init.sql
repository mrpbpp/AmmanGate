PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY,
  mac TEXT NOT NULL,
  ip TEXT,
  hostname TEXT,
  vendor TEXT,
  type_guess TEXT,
  risk_score INTEGER NOT NULL DEFAULT 0,
  first_seen TEXT NOT NULL,
  last_seen TEXT NOT NULL,
  tags TEXT NOT NULL DEFAULT '[]',
  notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_devices_mac ON devices(mac);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);

CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  ts TEXT NOT NULL,
  device_id TEXT,
  category TEXT NOT NULL,
  severity INTEGER NOT NULL,
  summary TEXT NOT NULL,
  raw TEXT NOT NULL DEFAULT '{}',
  FOREIGN KEY(device_id) REFERENCES devices(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE INDEX IF NOT EXISTS idx_events_device ON events(device_id);
CREATE INDEX IF NOT EXISTS idx_events_sev ON events(severity);

CREATE TABLE IF NOT EXISTS alerts (
  id TEXT PRIMARY KEY,
  ts TEXT NOT NULL,
  device_id TEXT,
  severity INTEGER NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  related_event_ids TEXT NOT NULL DEFAULT '[]',
  FOREIGN KEY(device_id) REFERENCES devices(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);

CREATE TABLE IF NOT EXISTS actions (
  id TEXT PRIMARY KEY,
  ts TEXT NOT NULL,
  action_type TEXT NOT NULL,
  target TEXT NOT NULL,
  ttl_sec INTEGER NOT NULL DEFAULT 1800,
  requested_by TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  approved_by TEXT,
  executed_ts TEXT,
  audit TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_actions_status ON actions(status);

CREATE TABLE IF NOT EXISTS approvals (
  id TEXT PRIMARY KEY,
  ts TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  action_id TEXT NOT NULL,
  method TEXT NOT NULL DEFAULT 'pin',
  nonce TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'waiting',
  FOREIGN KEY(action_id) REFERENCES actions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_approvals_action ON approvals(action_id);
CREATE INDEX IF NOT EXISTS idx_approvals_expires ON approvals(expires_at);
