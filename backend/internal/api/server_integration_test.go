package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"webcrawler/internal/config"
	"webcrawler/internal/storage"
)

func TestCreateRunValidationError(t *testing.T) {
	manager := NewRunManager(storage.NewMemory(), config.CrawlerDefaults{
		MaxDepth:           3,
		MaxPages:           2000,
		TimeBudget:         5 * time.Minute,
		MaxLinksPerPage:    100,
		GlobalConcurrency:  32,
		PerHostConcurrency: 4,
		UserAgent:          "WebCrawler/1.0",
		RespectRobots:      true,
	})
	server := NewServer(manager, ServerOptions{
		AllowedOrigins:      []string{"http://localhost:3000"},
		StorageMode:         "memory",
		RunCreateRateLimit:  100,
		RunCreateRateWindow: time.Minute,
	})

	body := map[string]any{
		"seed_url":             "https://93.184.216.34",
		"max_depth":            3,
		"max_pages":            9999,
		"time_budget_seconds":  300,
		"max_links_per_page":   100,
		"global_concurrency":   16,
		"per_host_concurrency": 3,
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(raw))
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errVal, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %v", payload)
	}
	if errVal["code"] != "validation_error" {
		t.Fatalf("expected validation_error, got %v", errVal["code"])
	}
}

func TestCreateRunRateLimited(t *testing.T) {
	manager := NewRunManager(storage.NewMemory(), config.CrawlerDefaults{
		MaxDepth:           3,
		MaxPages:           2000,
		TimeBudget:         5 * time.Minute,
		MaxLinksPerPage:    100,
		GlobalConcurrency:  32,
		PerHostConcurrency: 4,
		UserAgent:          "WebCrawler/1.0",
		RespectRobots:      true,
	})
	server := NewServer(manager, ServerOptions{
		AllowedOrigins:      []string{"http://localhost:3000"},
		StorageMode:         "memory",
		RunCreateRateLimit:  1,
		RunCreateRateWindow: time.Hour,
	})

	body := map[string]any{
		"seed_url":             "https://93.184.216.34",
		"max_depth":            3,
		"max_pages":            100,
		"time_budget_seconds":  120,
		"max_links_per_page":   50,
		"global_concurrency":   8,
		"per_host_concurrency": 2,
	}
	raw, _ := json.Marshal(body)

	req1 := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(raw))
	req1.RemoteAddr = "1.2.3.4:5678"
	rr1 := httptest.NewRecorder()
	server.Router().ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/runs", bytes.NewReader(raw))
	req2.RemoteAddr = "1.2.3.4:1234"
	rr2 := httptest.NewRecorder()
	server.Router().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr2.Code)
	}
}
