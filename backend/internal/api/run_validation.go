package api

import (
	"fmt"
	"strings"

	"webcrawler/internal/config"
	"webcrawler/internal/crawler"
)

const (
	maxDepthCap           = 10
	maxPagesCap           = 2000
	timeBudgetSecondsCap  = 300
	maxLinksPerPageCap    = 100
	globalConcurrencyCap  = 32
	perHostConcurrencyCap = 4
)

func buildValidatedRunConfig(req createRunRequest, defaults config.CrawlerDefaults) (crawler.RunConfig, map[string]string) {
	errs := make(map[string]string)

	cfg := crawler.RunConfig{
		SeedURL:            strings.TrimSpace(req.SeedURL),
		MaxDepth:           intField("max_depth", req.MaxDepth, defaults.MaxDepth, 1, maxDepthCap, errs),
		MaxPages:           intField("max_pages", req.MaxPages, defaults.MaxPages, 1, maxPagesCap, errs),
		TimeBudgetSeconds:  intField("time_budget_seconds", req.TimeBudgetSeconds, int(defaults.TimeBudget.Seconds()), 1, timeBudgetSecondsCap, errs),
		MaxLinksPerPage:    intField("max_links_per_page", req.MaxLinksPerPage, defaults.MaxLinksPerPage, 1, maxLinksPerPageCap, errs),
		GlobalConcurrency:  intField("global_concurrency", req.GlobalConcurrency, defaults.GlobalConcurrency, 1, globalConcurrencyCap, errs),
		PerHostConcurrency: intField("per_host_concurrency", req.PerHostConcurrency, defaults.PerHostConcurrency, 1, perHostConcurrencyCap, errs),
		UserAgent:          strings.TrimSpace(req.UserAgent),
		RespectRobots:      defaults.RespectRobots,
	}
	if req.RespectRobots != nil {
		cfg.RespectRobots = *req.RespectRobots
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = defaults.UserAgent
	}
	if len(cfg.UserAgent) > 256 {
		errs["user_agent"] = "must be at most 256 characters"
	}
	if cfg.PerHostConcurrency > cfg.GlobalConcurrency {
		errs["per_host_concurrency"] = "must be <= global_concurrency"
	}
	return cfg, errs
}

func intField(name string, input, def, minValue, maxValue int, errs map[string]string) int {
	effectiveDefault := clampInt(def, minValue, maxValue)
	if input == 0 {
		return effectiveDefault
	}
	if input < minValue {
		errs[name] = fmt.Sprintf("must be >= %d", minValue)
		return effectiveDefault
	}
	if input > maxValue {
		errs[name] = fmt.Sprintf("must be <= %d", maxValue)
		return effectiveDefault
	}
	return input
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
