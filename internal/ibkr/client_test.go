package ibkr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPositionsRedactsAccountIDs(t *testing.T) {
	const realA, realB = "U16472226", "U27034473"
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/api/iserver/auth/status", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"authenticated":true,"connected":true}`))
	})
	mux.HandleFunc("/v1/api/portfolio/accounts", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"accountId":"` + realB + `"},{"accountId":"` + realA + `"}]`))
	})
	mux.HandleFunc("/v1/api/portfolio/"+realA+"/positions/0", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"ticker":"NVDA","position":100,"mktPrice":200,"mktValue":20000,"avgCost":150,"unrealizedPnl":5000}]`))
	})
	mux.HandleFunc("/v1/api/portfolio/"+realB+"/positions/0", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"ticker":"AAPL","position":50,"mktPrice":200,"mktValue":10000,"avgCost":180,"unrealizedPnl":1000}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL)
	ok, err := c.Authenticated(context.Background())
	if err != nil || !ok {
		t.Fatalf("auth %v %v", ok, err)
	}
	pos, err := c.Positions(context.Background())
	if err != nil || len(pos) != 2 {
		t.Fatalf("positions %v %+v", err, pos)
	}
	// The real account ids must never appear in the output.
	blob, _ := json.Marshal(pos)
	if strings.Contains(string(blob), realA) || strings.Contains(string(blob), realB) {
		t.Fatalf("real account id leaked: %s", blob)
	}
	// Alias is stable by sorted real id: U164... < U270..., so account-1 = realA.
	for _, p := range pos {
		if p.Symbol == "NVDA" && p.Account != "account-1" {
			t.Fatalf("NVDA alias = %q, want account-1", p.Account)
		}
		if p.Symbol == "AAPL" && p.Account != "account-2" {
			t.Fatalf("AAPL alias = %q, want account-2", p.Account)
		}
	}
	// Weights: NVDA 20000 / 30000 = 0.6667.
	if pos[0].Symbol != "NVDA" || pos[0].Weight < 0.66 || pos[0].Weight > 0.67 {
		t.Fatalf("weight %+v", pos[0])
	}
}

func TestNotAuthenticated401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	ok, err := New(srv.URL).Authenticated(context.Background())
	if err != nil || ok {
		t.Fatalf("want not-authenticated with no error, got ok=%v err=%v", ok, err)
	}
}
