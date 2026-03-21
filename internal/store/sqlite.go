package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sudhan30/freshprobe/internal/verdict"
	_ "modernc.org/sqlite"
)

// SQLiteStore persists probe results in a local SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates a SQLite database at the given path.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		dir := filepath.Join(home, ".freshprobe")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
		path = filepath.Join(dir, "data.db")
	}

	// Ensure parent directory exists
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) SaveProbe(ctx context.Context, v *verdict.ProbeVerdict) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal verdict: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO probes (id, url, verdict, confidence, result_json, probed_at) VALUES (?, ?, ?, ?, ?, ?)`,
		v.ProbeMetadata.ProbeID,
		v.Endpoint,
		string(v.Verdict),
		v.Confidence,
		string(data),
		v.ProbeMetadata.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
	)
	return err
}

func (s *SQLiteStore) GetHistory(ctx context.Context, url string, limit int) ([]*verdict.ProbeVerdict, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT result_json FROM probes WHERE url = ? ORDER BY probed_at DESC LIMIT ?`,
		url, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*verdict.ProbeVerdict
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var v verdict.ProbeVerdict
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		results = append(results, &v)
	}
	return results, rows.Err()
}

func (s *SQLiteStore) SaveFingerprint(ctx context.Context, url string, hash string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO fingerprints (url, hash) VALUES (?, ?)`,
		url, hash,
	)
	return err
}

func (s *SQLiteStore) GetFingerprints(ctx context.Context, url string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT hash FROM fingerprints WHERE url = ? ORDER BY created_at DESC LIMIT ?`,
		url, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		hashes = append(hashes, h)
	}
	return hashes, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
