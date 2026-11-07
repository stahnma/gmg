package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

var ctx = context.Background()

func open(t *testing.T, maxMB int) *Store {
	t.Helper()
	s, err := Open(t.TempDir(), maxMB)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func on(grill, food int) protocol.Status {
	return protocol.Status{IsOn: true, CurrentGrillTemp: grill, CurrentFoodTemp: food}
}

func TestRecordAndHistory(t *testing.T) {
	s := open(t, 50)
	before := time.Now().Unix()
	if err := s.Record(ctx, on(225, 150)); err != nil {
		t.Fatal(err)
	}
	rows, err := s.History(ctx, before)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows %+v, %v", rows, err)
	}
	r := rows[0]
	if r.Grill != 225 || r.Food == nil || *r.Food != 150 || r.Timestamp < before {
		t.Fatalf("row %+v", r)
	}
	if rows, _ := s.History(ctx, time.Now().Unix()+60); len(rows) != 0 {
		t.Fatalf("future since should return nothing, got %d", len(rows))
	}
}

func TestSkipsWhenGrillOff(t *testing.T) {
	s := open(t, 50)
	if err := s.Record(ctx, protocol.Status{IsOn: false, CurrentGrillTemp: 90}); err != nil {
		t.Fatal(err)
	}
	if rows, _ := s.History(ctx, 0); len(rows) != 0 || s.inserts != 0 {
		t.Fatalf("expected nothing recorded, got %d rows", len(rows))
	}
}

func TestSameSecondIsIgnored(t *testing.T) {
	s := open(t, 50)
	for i := 0; i < 3; i++ { // all within one second in practice
		if err := s.Record(ctx, on(200+i, 100)); err != nil {
			t.Fatalf("duplicate timestamp must not error: %v", err)
		}
	}
	if rows, _ := s.History(ctx, 0); len(rows) < 1 || len(rows) > 2 {
		t.Fatalf("expected one row per second, got %d", len(rows))
	}
}

func TestHistoryIsEmptySliceNotNil(t *testing.T) {
	rows, err := open(t, 50).History(ctx, 0)
	if err != nil || rows == nil {
		t.Fatalf("got %v, %v (must be [] so the API returns a JSON array)", rows, err)
	}
}

func TestCleanupDeletesOldestTenPercent(t *testing.T) {
	s := open(t, 0) // any non-empty file is over the limit
	for i := 0; i < 20; i++ {
		if _, err := s.db.Exec(`INSERT INTO temperature_log (timestamp, grill_temperature) VALUES (?, ?)`, 1000+i, 200); err != nil {
			t.Fatal(err)
		}
	}
	n, err := s.Cleanup(ctx)
	if err != nil || n != 2 {
		t.Fatalf("deleted %d, %v; want 2", n, err)
	}
	rows, _ := s.History(ctx, 0)
	if len(rows) != 18 || rows[0].Timestamp != 1002 {
		t.Fatalf("want 18 rows starting at 1002, got %d starting at %d", len(rows), rows[0].Timestamp)
	}
}

func TestCleanupUnderLimitDoesNothing(t *testing.T) {
	s := open(t, 50)
	s.Record(ctx, on(200, 100))
	if n, err := s.Cleanup(ctx); n != 0 || err != nil {
		t.Fatalf("got %d, %v", n, err)
	}
}

func TestCleanupRunsEveryHundredInserts(t *testing.T) {
	s := open(t, 0)
	s.db.Exec(`INSERT INTO temperature_log (timestamp, grill_temperature) VALUES (1, 1), (2, 2), (3, 3)`)
	s.inserts = cleanupEvery - 1
	if err := s.Record(ctx, on(200, 100)); err != nil {
		t.Fatal(err)
	}
	if rows, _ := s.History(ctx, 0); len(rows) != 3 { // 4 rows, ceil(10%) = 1 deleted
		t.Fatalf("want 3 rows after cleanup, got %d", len(rows))
	}
}

func TestRecordErrorIsReturned(t *testing.T) {
	s := open(t, 50)
	s.Close()
	if err := s.Record(ctx, on(200, 100)); err == nil {
		t.Fatal("expected an error from a closed database")
	}
}

func TestReadsDatabaseCreatedByNodeServer(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, 50) // same CREATE TABLE as the Node server
	if err != nil {
		t.Fatal(err)
	}
	s.db.Exec(`INSERT INTO temperature_log (timestamp, grill_temperature, food_temperature) VALUES (100, 225, NULL)`)
	s.Close()
	if _, err := os.Stat(filepath.Join(dir, "grill_data.db")); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir, 50)
	defer s2.Close()
	rows, _ := s2.History(ctx, 0)
	if len(rows) != 1 || rows[0].Food != nil {
		t.Fatalf("got %+v", rows)
	}
}
