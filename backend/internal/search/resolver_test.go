package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResolveBraveResults(t *testing.T) {
	resolver := New(
		"https://api.search.brave.com/res/v1/web/search",
		"test-key",
		&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("X-Subscription-Token"); got != "test-key" {
				t.Fatalf("unexpected api key header: %q", got)
			}
			return responseFor(req, http.StatusOK, `{
				"web": {
					"results": [
						{"url":"https://en.wikipedia.org/wiki/Alan_Turing"},
						{"url":"https://www.britannica.com/biography/Alan-Turing"},
						{"url":"https://en.wikipedia.org/wiki/Alan_Turing"}
					]
				}
			}`), nil
		})},
		"test-agent",
	)

	result, err := resolver.Resolve(context.Background(), "Alan Turing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.URLs) != 2 {
		t.Fatalf("expected 2 unique urls, got %#v", result.URLs)
	}
	if result.URLs[0] != "https://en.wikipedia.org/wiki/Alan_Turing" {
		t.Fatalf("unexpected first result: %q", result.URLs[0])
	}
	if len(result.Attempts) != 1 || result.Attempts[0].ResultCount != 2 {
		t.Fatalf("unexpected attempts: %#v", result.Attempts)
	}
}

func TestResolveRequiresAPIKey(t *testing.T) {
	resolver := New("https://api.search.brave.com/res/v1/web/search", "", &http.Client{}, "test-agent")
	_, err := resolver.Resolve(context.Background(), "Alan Turing")
	if err == nil || !strings.Contains(err.Error(), "BRAVE_SEARCH_API_KEY") {
		t.Fatalf("expected missing api key error, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func responseFor(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}
}
