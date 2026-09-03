CREATE TABLE IF NOT EXISTS stations (
    id INTEGER PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS doors (
    id INTEGER PRIMARY KEY,
    station_id INTEGER NOT NULL REFERENCES stations(id) ON DELETE RESTRICT,
    platform TEXT NOT NULL,
    door_number INTEGER NOT NULL,
    sap_tag TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_station_platform_door UNIQUE (station_id, platform, door_number)
);

CREATE INDEX IF NOT EXISTS idx_doors_station ON doors(station_id);
CREATE INDEX IF NOT EXISTS idx_doors_sap_tag ON doors(sap_tag);
