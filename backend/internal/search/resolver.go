package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Resolver struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	userAgent  string
}

type Attempt struct {
	Source        string
	Status        string
	Snippet       string
	Error         string
	ResultCount   int
	ResponseBytes int
}

type ResultSet struct {
	URLs     []string
	Status   string
	Snippet  string
	Source   string
	Attempts []Attempt
}

type braveWebResponse struct {
	Web struct {
		Results []struct {
			URL string `json:"url"`
		} `json:"results"`
	} `json:"web"`
}

func New(baseURL, apiKey string, client *http.Client, userAgent string) *Resolver {
	return &Resolver{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: client,
		userAgent:  userAgent,
	}
}

func (r *Resolver) Resolve(ctx context.Context, query string) (ResultSet, error) {
	if strings.TrimSpace(query) == "" {
		return ResultSet{}, fmt.Errorf("empty query")
	}
	if r.apiKey == "" {
		return ResultSet{
			Attempts: []Attempt{{
				Source: r.baseURL,
				Error:  "missing BRAVE_SEARCH_API_KEY",
			}},
		}, fmt.Errorf("missing BRAVE_SEARCH_API_KEY")
	}
	if strings.TrimSpace(r.baseURL) == "" {
		return ResultSet{}, fmt.Errorf("empty search base url")
	}

	u, err := url.Parse(r.baseURL)
	if err != nil {
		return ResultSet{}, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("count", "10")
	q.Set("safesearch", "moderate")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ResultSet{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", r.apiKey)
	if r.userAgent != "" {
		req.Header.Set("User-Agent", r.userAgent)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return ResultSet{
			Source: u.String(),
			Attempts: []Attempt{{
				Source: u.String(),
				Error:  err.Error(),
			}},
		}, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return ResultSet{
			Source: u.String(),
			Attempts: []Attempt{{
				Source: u.String(),
				Status: resp.Status,
				Error:  readErr.Error(),
			}},
		}, readErr
	}
	snippet := compactSnippet(string(body))
	attempt := Attempt{
		Source:        u.String(),
		Status:        resp.Status,
		Snippet:       snippet,
		ResponseBytes: len(body),
	}

	if resp.StatusCode >= 400 {
		attempt.Error = fmt.Sprintf("search request failed: %s", resp.Status)
		return ResultSet{
			Status:   resp.Status,
			Snippet:  snippet,
			Source:   u.String(),
			Attempts: []Attempt{attempt},
		}, fmt.Errorf("search request failed: %s", resp.Status)
	}

	var payload braveWebResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		attempt.Error = err.Error()
		return ResultSet{
			Status:   resp.Status,
			Snippet:  snippet,
			Source:   u.String(),
			Attempts: []Attempt{attempt},
		}, err
	}

	results := make([]string, 0, len(payload.Web.Results))
	for _, item := range payload.Web.Results {
		candidate := strings.TrimSpace(item.URL)
		if candidate == "" {
			continue
		}
		if parsed, err := url.Parse(candidate); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			results = append(results, parsed.String())
		}
	}
	results = unique(results)
	attempt.ResultCount = len(results)

	return ResultSet{
		URLs:     results,
		Status:   resp.Status,
		Snippet:  snippet,
		Source:   u.String(),
		Attempts: []Attempt{attempt},
	}, nil
}

func unique(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func compactSnippet(raw string) string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return ""
	}
	joined := strings.Join(fields, " ")
	if len(joined) > 240 {
		return joined[:240] + "..."
	}
	return joined
}
