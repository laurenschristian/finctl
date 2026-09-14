package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/laurenschristian/finctl/internal/cache"
)

func TestGetCachesAndRateLimits(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		if r.Header.Get("User-Agent") != "test-ua" {
			t.Errorf("UA %q", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	store, _ := cache.Open(filepath.Join(t.TempDir(), "c.db"))
	defer func() { _ = store.Close() }()
	c := New(store, "test-ua")

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := c.Get(ctx, "x", srv.URL, time.Minute)
		if err != nil || string(b) != "ok" {
			t.Fatalf("get %v %q", err, b)
		}
	}
	if atomic.LoadInt64(&hits) != 1 {
		t.Fatalf("expected 1 network hit, got %d", hits)
	}
}

func TestPostJSONAndError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte("posted"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := New(nil, "ua") // nil cache is allowed
	b, err := c.PostJSON(context.Background(), "p", srv.URL, []byte(`{"a":1}`), 0)
	if err != nil || string(b) != "posted" {
		t.Fatalf("post %v %q", err, b)
	}
	if _, err := c.Get(context.Background(), "p", srv.URL, 0); err == nil {
		t.Fatal("want HTTP 500 error")
	}
}
