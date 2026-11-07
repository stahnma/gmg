// Package store persists grill temperature history in SQLite. The file name
// and schema match the original Node server, so existing history carries over.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/stahnma/gmg/internal/protocol"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

// FileName is the database file inside the configured directory.
const FileName = "grill_data.db"

// cleanupEvery is how many inserts happen between database size checks.
const cleanupEvery = 100

// Row is one history entry. JSON names match the original API.
type Row struct {
	ID        int64  `json:"temperature_log_id"`
	Timestamp int64  `json:"timestamp"`
	Grill     int64  `json:"grill_temperature"`
	Food      *int64 `json:"food_temperature"`
}

// Store is the temperature history database.
type Store struct {
	db       *sql.DB
	path     string
	maxBytes int64
	mu       sync.Mutex
	inserts  int
}

// Open opens (creating if needed) dir/grill_data.db. maxSizeMB bounds the
// file size; when exceeded, the oldest 10% of rows are deleted.
func Open(dir string, maxSizeMB int) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, FileName)
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // one writer; keeps VACUUM and inserts serialized
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS temperature_log (
			temperature_log_id integer PRIMARY KEY,
			timestamp integer UNIQUE,
			grill_temperature integer(2) NOT NULL,
			food_temperature integer(2) NULL
		)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return &Store{db: db, path: path, maxBytes: int64(maxSizeMB) * 1024 * 1024}, nil
}

// Path returns the database file path.
func (s *Store) Path() string { return s.path }

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Record stores a status sample if the grill is on. A second sample in the
// same second is ignored (timestamp is UNIQUE at one-second resolution).
func (s *Store) Record(ctx context.Context, st protocol.Status) error {
	if !st.IsOn {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO temperature_log (timestamp, grill_temperature, food_temperature)
		VALUES (strftime('%s','now'), ?, ?)`, st.CurrentGrillTemp, st.CurrentFoodTemp); err != nil {
		return fmt.Errorf("record status: %w", err)
	}
	s.mu.Lock()
	s.inserts++
	check := s.inserts%cleanupEvery == 0
	s.mu.Unlock()
	if check {
		if _, err := s.Cleanup(ctx); err != nil {
			return fmt.Errorf("cleanup: %w", err)
		}
	}
	return nil
}

// Cleanup deletes the oldest 10% of rows and vacuums if the file is over the
// size limit. It returns the number of rows deleted.
func (s *Store) Cleanup(ctx context.Context) (int64, error) {
	fi, err := os.Stat(s.path)
	if err != nil || fi.Size() <= s.maxBytes {
		return 0, nil
	}
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM temperature_log`).Scan(&count); err != nil {
		return 0, err
	}
	n := int64(math.Ceil(float64(count) * 0.1))
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM temperature_log WHERE temperature_log_id IN (
			SELECT temperature_log_id FROM temperature_log ORDER BY timestamp ASC LIMIT ?
		)`, n); err != nil {
		return 0, err
	}
	if _, err := s.db.ExecContext(ctx, `VACUUM`); err != nil {
		return 0, err
	}
	return n, nil
}

// History returns rows with timestamp >= since (Unix seconds), oldest first.
func (s *Store) History(ctx context.Context, since int64) ([]Row, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT temperature_log_id, timestamp, grill_temperature, food_temperature
		FROM temperature_log WHERE timestamp >= ? ORDER BY timestamp`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Row{}
	for rows.Next() {
		var r Row
		var food sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Timestamp, &r.Grill, &food); err != nil {
			return nil, err
		}
		if food.Valid {
			r.Food = &food.Int64
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
