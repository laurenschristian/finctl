// Package fin is a thin client for free market, rates and macro data plus portfolio review.
package fin

import (
	"net/http"
	"strings"
	"time"
)

// Client talks to the finctl target. Endpoints land here as commands are built.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}
