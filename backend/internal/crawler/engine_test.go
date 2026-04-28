package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestReadBodyLimited(t *testing.T) {
	data, size, errClass := readBodyLimited(strings.NewReader("hello"), 4)
	if errClass != ErrSizeLimit {
		t.Fatalf("expected size limit, got %s", errClass)
	}
	if size <= 4 {
		t.Fatalf("expected size > 4, got %d", size)
	}
	if data != nil {
		t.Fatal("expected nil data on size limit")
	}
}

func TestParseRetryAfter(t *testing.T) {
	if d := parseRetryAfter(""); d != 0 {
		t.Fatal("expected zero")
	}
	if d := parseRetryAfter("5"); d != 5*time.Second {
		t.Fatalf("expected 5s, got %v", d)
	}
}

func TestExtractContent(t *testing.T) {
	body := []byte(`<html><head><title>Example Page</title><script>alert(1)</script></head><body><main><p>Hello <strong>world</strong>.</p><a href="/next">Next</a></main></body></html>`)
	base := mustParseURL("https://example.com/start")
	got := ExtractContent(base, body, 10)
	if got.Title != "Example Page" {
		t.Fatalf("unexpected title: %q", got.Title)
	}
	if !strings.Contains(got.Text, "Hello world") {
		t.Fatalf("expected extracted text, got %q", got.Text)
	}
	if len(got.Links) != 1 || got.Links[0] != "https://example.com/next" {
		t.Fatalf("unexpected links: %#v", got.Links)
	}
}

func TestEngineBuildsTreeFromDiscoveredPages(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a":
			_, _ = w.Write([]byte(`<html><head><title>Node A</title></head><body><a href="/b">B</a><a href="/c">C</a></body></html>`))
		case "/b":
			_, _ = w.Write([]byte(`<html><head><title>Node B</title></head><body><a href="/d">D</a></body></html>`))
		case "/c":
			_, _ = w.Write([]byte(`<html><head><title>Node C</title></head><body><p>Leaf C</p></body></html>`))
		case "/d":
			_, _ = w.Write([]byte(`<html><head><title>Node D</title></head><body><p>Leaf D</p></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	var mu sync.Mutex
	pages := make(map[string]Page)
	nodes := make(map[string]TreeNode)
	edges := []TreeEdge{}
	done := make(chan string, 1)

	engine := NewEngine("run-test", RunConfig{
		Mode:                SourceModeURL,
		Input:               serverURL + "/a",
		SeedURL:             serverURL + "/a",
		MaxDepth:            3,
		MaxPages:            4,
		MaxLinksPerPage:     10,
		GlobalConcurrency:   4,
		PerHostConcurrency:  2,
		UserAgent:           "test",
		RequestTimeout:      5 * time.Second,
		HeaderTimeout:       5 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		IdleConnTimeout:     5 * time.Second,
		MaxBodyBytes:        1 << 20,
	}, EngineHooks{
		OnPage: func(page Page) {
			mu.Lock()
			pages[page.URL] = page
			mu.Unlock()
		},
		OnTreeNode: func(node TreeNode) {
			mu.Lock()
			nodes[node.ID] = node
			mu.Unlock()
		},
		OnTreeEdge: func(edge TreeEdge) {
			mu.Lock()
			edges = append(edges, edge)
			mu.Unlock()
		},
		OnComplete: func(reason string) {
			select {
			case done <- reason:
			default:
			}
		},
	})

	rootID := NewPageID()
	engine.Start(rootID, nil)

	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("crawl did not complete")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(pages) != 4 {
		t.Fatalf("expected 4 pages, got %d", len(pages))
	}
	nodeA := findPage(pages, "/a")
	nodeB := findPage(pages, "/b")
	nodeC := findPage(pages, "/c")
	nodeD := findPage(pages, "/d")
	if nodeA == nil || nodeB == nil || nodeC == nil || nodeD == nil {
		t.Fatalf("expected pages for /a,/b,/c,/d, got %#v", pages)
	}
	if nodeB.ParentPageID != nodeA.ID || nodeC.ParentPageID != nodeA.ID {
		t.Fatalf("expected B and C to descend from A: %+v %+v", nodeB, nodeC)
	}
	if nodeD.ParentPageID != nodeB.ID {
		t.Fatalf("expected D to descend from B: %+v", nodeD)
	}
	for _, edge := range edges {
		if edge.ParentPageID == nodeC.ID && edge.ChildPageID == nodeD.ID {
			t.Fatalf("unexpected C -> D edge present")
		}
	}
	if !slices.Contains(nodeA.OutgoingLinks, serverURL+"/b") && !slices.Contains(nodeA.OutgoingLinks, serverURL+"/c") {
		t.Fatalf("expected outgoing links from A, got %#v", nodeA.OutgoingLinks)
	}
}

func findPage(pages map[string]Page, suffix string) *Page {
	for url, page := range pages {
		if strings.HasSuffix(url, suffix) {
			p := page
			return &p
		}
	}
	return nil
}

func BenchmarkSyntheticCrawl501Pages(b *testing.B) {
	const childPages = 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/root" {
			_, _ = w.Write([]byte(`<html><head><title>Root</title></head><body>`))
			for i := 0; i < childPages; i++ {
				_, _ = fmt.Fprintf(w, `<a href="/page/%d">Page %d</a>`, i, i)
			}
			_, _ = w.Write([]byte(`</body></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/page/") {
			_, _ = fmt.Fprintf(w, `<html><head><title>%s</title></head><body><p>Readable benchmark content for %s.</p></body></html>`, r.URL.Path, r.URL.Path)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	for i := 0; i < b.N; i++ {
		done := make(chan string, 1)
		engine := NewEngine("bench", RunConfig{
			Mode:                SourceModeURL,
			Input:               server.URL + "/root",
			SeedURL:             server.URL + "/root",
			MaxDepth:            1,
			MaxPages:            childPages + 1,
			MaxLinksPerPage:     childPages,
			GlobalConcurrency:   64,
			PerHostConcurrency:  64,
			UserAgent:           "bench",
			RequestTimeout:      10 * time.Second,
			HeaderTimeout:       10 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     10 * time.Second,
			MaxBodyBytes:        1 << 20,
		}, EngineHooks{
			OnComplete: func(reason string) {
				select {
				case done <- reason:
				default:
				}
			},
		})
		engine.Start(NewPageID(), nil)
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			b.Fatal("benchmark crawl did not complete")
		}
	}
}
