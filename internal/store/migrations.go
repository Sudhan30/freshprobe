package store

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS probes (
    id          TEXT PRIMARY KEY,
    url         TEXT NOT NULL,
    verdict     TEXT NOT NULL,
    confidence  REAL NOT NULL,
    result_json TEXT NOT NULL,
    probed_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_probes_url ON probes(url, probed_at);

CREATE TABLE IF NOT EXISTS fingerprints (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    url        TEXT NOT NULL,
    hash       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_fp_url ON fingerprints(url, created_at);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
