package artifacts

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"webcrawler/internal/crawler"
)

func TestPersistWritesExpectedFiles(t *testing.T) {
	root := t.TempDir()
	store := New(root)
	now := time.Now().UTC()
	snapshot := crawler.Snapshot{
		RunID:     "run-1",
		Config:    crawler.RunConfig{Mode: crawler.SourceModeURL, Input: "https://example.com", SeedURL: "https://example.com"},
		Seed:      crawler.SearchSeed{Query: "https://example.com", PrimaryURL: "https://example.com", Results: []string{"https://example.com"}},
		Status:    "stopped",
		CreatedAt: now,
		Summary:   crawler.Summary{Status: "stopped", PagesFetched: 1, PagesQueued: 1},
		Pages: []crawler.Page{{
			ID:           "page-1",
			URL:          "https://example.com",
			CanonicalURL: "https://example.com",
			Title:        "Example",
			FetchedAt:    now,
			DiscoveredAt: now,
		}},
		TreeNodes:   []crawler.TreeNode{{ID: "page-1", URL: "https://example.com", Title: "Example"}},
		TreeEdges:   []crawler.TreeEdge{},
		Diagnostics: crawler.Diagnostics{ArtifactDir: filepath.Join(root, "run-1"), ArtifactFiles: store.Paths("run-1")},
	}
	if err := store.Persist(snapshot); err != nil {
		t.Fatalf("persist failed: %v", err)
	}
	for name, path := range store.Paths("run-1") {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s file at %s: %v", name, path, err)
		}
	}
}
