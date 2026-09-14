// Package cache is finctl's on-disk store: a per-provider response cache with
// TTLs, plus a small time-series table for trended metrics (e.g. gpu-rent).
package cache

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver, no cgo
)

type Store struct {
	db *sql.DB
}

// Open creates (or opens) the SQLite store at path, creating parent dirs.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS cache(
			provider TEXT NOT NULL, key TEXT NOT NULL,
			body BLOB NOT NULL, fetched_at INTEGER NOT NULL,
			PRIMARY KEY(provider, key));
		CREATE TABLE IF NOT EXISTS history(
			series TEXT NOT NULL, ts INTEGER NOT NULL, value REAL NOT NULL,
			PRIMARY KEY(series, ts));`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Get returns a cached body when it is younger than ttl. The bool is false on a
// miss or a stale entry, so the caller fetches.
func (s *Store) Get(ctx context.Context, provider, key string, ttl time.Duration) ([]byte, bool) {
	var body []byte
	var fetched int64
	err := s.db.QueryRowContext(ctx, `SELECT body, fetched_at FROM cache WHERE provider=? AND key=?`, provider, key).Scan(&body, &fetched)
	if err != nil {
		return nil, false
	}
	if ttl > 0 && time.Since(time.Unix(fetched, 0)) > ttl {
		return nil, false
	}
	return body, true
}

// Put stores (or replaces) a cached body with the current timestamp.
func (s *Store) Put(ctx context.Context, provider, key string, body []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cache(provider,key,body,fetched_at) VALUES(?,?,?,?)
		 ON CONFLICT(provider,key) DO UPDATE SET body=excluded.body, fetched_at=excluded.fetched_at`,
		provider, key, body, time.Now().Unix())
	return err
}

// Clear wipes the cache; a provider of "" clears everything.
func (s *Store) Clear(ctx context.Context, provider string) error {
	if provider == "" {
		_, err := s.db.ExecContext(ctx, `DELETE FROM cache`)
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM cache WHERE provider=?`, provider)
	return err
}

// Append records a point in a trended series (idempotent per second).
func (s *Store) Append(ctx context.Context, series string, ts int64, value float64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO history(series,ts,value) VALUES(?,?,?)
		 ON CONFLICT(series,ts) DO UPDATE SET value=excluded.value`,
		series, ts, value)
	return err
}

// Series returns a trended series ordered by time.
func (s *Store) Series(ctx context.Context, series string) ([]float64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT value FROM history WHERE series=? ORDER BY ts`, series)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) Close() error { return s.db.Close() }
