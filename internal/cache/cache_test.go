package cache

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestGetPutTTL(t *testing.T) {
	s := open(t)
	if _, ok := s.Get(context.Background(), "p", "k", time.Minute); ok {
		t.Fatal("miss expected")
	}
	if err := s.Put(context.Background(), "p", "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	if b, ok := s.Get(context.Background(), "p", "k", time.Minute); !ok || string(b) != "v" {
		t.Fatalf("hit expected: %q %v", b, ok)
	}
	// A zero/negative TTL means "any age is stale" only when >0; ttl<=0 = never expire here.
	if _, ok := s.Get(context.Background(), "p", "k", -time.Second); !ok {
		t.Fatal("ttl<=0 should not expire")
	}
	// Force staleness via a tiny ttl.
	time.Sleep(2 * time.Millisecond)
	if _, ok := s.Get(context.Background(), "p", "k", time.Nanosecond); ok {
		t.Fatal("stale expected")
	}
}

func TestClearAndProviderScope(t *testing.T) {
	s := open(t)
	_ = s.Put(context.Background(), "a", "1", []byte("x"))
	_ = s.Put(context.Background(), "b", "1", []byte("y"))
	if err := s.Clear(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(context.Background(), "a", "1", time.Minute); ok {
		t.Fatal("a cleared")
	}
	if _, ok := s.Get(context.Background(), "b", "1", time.Minute); !ok {
		t.Fatal("b kept")
	}
	_ = s.Clear(context.Background(), "")
	if _, ok := s.Get(context.Background(), "b", "1", time.Minute); ok {
		t.Fatal("all cleared")
	}
}

func TestHistorySeries(t *testing.T) {
	s := open(t)
	_ = s.Append(context.Background(), "gpu:h100", 100, 2.5)
	_ = s.Append(context.Background(), "gpu:h100", 200, 2.1)
	_ = s.Append(context.Background(), "gpu:h100", 200, 2.2) // upsert
	vals, err := s.Series(context.Background(), "gpu:h100")
	if err != nil || len(vals) != 2 || vals[0] != 2.5 || vals[1] != 2.2 {
		t.Fatalf("series %v %v", vals, err)
	}
}
