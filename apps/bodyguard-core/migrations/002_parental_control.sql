-- Parental Control Tables
-- Filter rules for domains and categories
CREATE TABLE IF NOT EXISTS filter_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('domain', 'category')),
    pattern TEXT NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- DNS query logs for monitoring and reporting
CREATE TABLE IF NOT EXISTS dns_queries (
    id TEXT PRIMARY KEY,
    ts TEXT NOT NULL DEFAULT (datetime('now')),
    device_id TEXT,
    domain TEXT NOT NULL,
    query_type TEXT,
    blocked BOOLEAN DEFAULT 0,
    rule_id TEXT,
    FOREIGN KEY(device_id) REFERENCES devices(id),
    FOREIGN KEY(rule_id) REFERENCES filter_rules(id)
);

-- Device parental control profiles
CREATE TABLE IF NOT EXISTS device_profiles (
    device_id TEXT PRIMARY KEY,
    filter_level TEXT DEFAULT 'off' CHECK(filter_level IN ('off', 'light', 'moderate', 'strict')),
    schedule TEXT, -- JSON: {"allowed_days":[1,2,3,4,5,6,7],"start":"00:00","end":"23:59"}
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY(device_id) REFERENCES devices(id)
);

-- Preset filter categories
INSERT OR IGNORE INTO filter_rules (id, name, type, pattern) VALUES
('cat-adult', 'Adult Content', 'category', 'adult,porn,xxx,sex'),
('cat-gambling', 'Gambling & Betting', 'category', 'gambling,casino,betting,poker,lottery'),
('cat-social', 'Social Media', 'category', 'facebook,twitter,instagram,tiktok,snapchat,linkedin'),
('cat-violence', 'Violence & Gore', 'category', 'violence,gore,weapons,drugs'),
('cat-streaming', 'Streaming Entertainment', 'category', 'youtube,netflix,hulu,disney+,hbo,max'),
('cat-games', 'Online Gaming', 'category', 'steam,ea,ubisoft,blizzard,xbox,playstation'),
('cat-forums', 'Forums & Chat', 'category', 'reddit,4chan,discord,telegram');
