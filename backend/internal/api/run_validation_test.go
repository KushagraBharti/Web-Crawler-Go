package api

import (
	"testing"
	"time"

	"webcrawler/internal/config"
)

func TestBuildValidatedRunConfig(t *testing.T) {
	defaults := config.CrawlerDefaults{
		MaxDepth:           3,
		MaxPages:           2000,
		TimeBudget:         5 * time.Minute,
		MaxLinksPerPage:    100,
		GlobalConcurrency:  32,
		PerHostConcurrency: 4,
		UserAgent:          "WebCrawler/1.0",
		RespectRobots:      true,
	}

	cfg, errs := buildValidatedRunConfig(createRunRequest{
		SeedURL:            "https://example.com",
		MaxPages:           99999,
		TimeBudgetSeconds:  600,
		GlobalConcurrency:  128,
		PerHostConcurrency: 6,
	}, defaults)
	if len(errs) == 0 {
		t.Fatal("expected validation errors")
	}
	if cfg.MaxPages != maxPagesCap {
		t.Fatalf("expected clamped max_pages, got %d", cfg.MaxPages)
	}
	if cfg.TimeBudgetSeconds != timeBudgetSecondsCap {
		t.Fatalf("expected clamped time budget, got %d", cfg.TimeBudgetSeconds)
	}
	if cfg.GlobalConcurrency != globalConcurrencyCap {
		t.Fatalf("expected clamped global concurrency, got %d", cfg.GlobalConcurrency)
	}
	if cfg.PerHostConcurrency != perHostConcurrencyCap {
		t.Fatalf("expected clamped per-host concurrency, got %d", cfg.PerHostConcurrency)
	}
}
