package crawler

import "time"

type SourceMode string

const (
	SourceModeURL     SourceMode = "url"
	SourceModeKeyword SourceMode = "keyword"
)

type RunConfig struct {
	Mode                SourceMode    `json:"mode"`
	Input               string        `json:"input"`
	SeedURL             string        `json:"seed_url"`
	MaxDepth            int           `json:"max_depth"`
	MaxPages            int           `json:"max_pages"`
	TimeBudget          time.Duration `json:"time_budget"`
	TimeBudgetSeconds   int           `json:"time_budget_seconds"`
	MaxLinksPerPage     int           `json:"max_links_per_page"`
	GlobalConcurrency   int           `json:"global_concurrency"`
	PerHostConcurrency  int           `json:"per_host_concurrency"`
	UserAgent           string        `json:"user_agent"`
	RespectRobots       bool          `json:"respect_robots"`
	RequestTimeout      time.Duration `json:"request_timeout"`
	HeaderTimeout       time.Duration `json:"header_timeout"`
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout"`
	MaxBodyBytes        int64         `json:"max_body_bytes"`
	RobotsTTL           time.Duration `json:"robots_ttl"`
	RetryMax            int           `json:"retry_max"`
	RetryBaseDelay      time.Duration `json:"retry_base_delay"`
	CircuitTripCount    int           `json:"circuit_trip_count"`
	CircuitResetTime    time.Duration `json:"circuit_reset_time"`
}

func (c RunConfig) Normalize() RunConfig {
	if c.TimeBudget == 0 && c.TimeBudgetSeconds > 0 {
		c.TimeBudget = time.Duration(c.TimeBudgetSeconds) * time.Second
	}
	if c.MaxDepth < 0 {
		c.MaxDepth = 0
	}
	if c.MaxPages < 0 {
		c.MaxPages = 0
	}
	if c.MaxLinksPerPage < 0 {
		c.MaxLinksPerPage = 0
	}
	if c.GlobalConcurrency < 0 {
		c.GlobalConcurrency = 0
	}
	if c.PerHostConcurrency < 0 {
		c.PerHostConcurrency = 0
	}
	if c.Mode == "" {
		c.Mode = SourceModeURL
	}
	return c
}

type Task struct {
	PageID       string
	ParentPageID string
	URL          string
	Canonical    string
	Host         string
	Depth        int
	NotBefore    time.Time
	Retries      int
	DiscoveredAt time.Time
	Permit       *Permit
}

type Permit struct {
	Global *Semaphore
	Host   *Semaphore
}

func (p *Permit) Release() {
	if p == nil {
		return
	}
	if p.Global != nil {
		p.Global.Release()
	}
	if p.Host != nil {
		p.Host.Release()
	}
}

type SearchSeed struct {
	Query      string   `json:"query"`
	PrimaryURL string   `json:"primary_url"`
	Results    []string `json:"results"`
}

type Page struct {
	ID            string    `json:"id"`
	ParentPageID  string    `json:"parent_page_id,omitempty"`
	SourceMode    string    `json:"source_mode"`
	SourceInput   string    `json:"source_input"`
	URL           string    `json:"url"`
	CanonicalURL  string    `json:"canonical_url"`
	Host          string    `json:"host"`
	Title         string    `json:"title"`
	Text          string    `json:"text"`
	Excerpt       string    `json:"excerpt"`
	OutgoingLinks []string  `json:"outgoing_links"`
	Depth         int       `json:"depth"`
	StatusCode    int       `json:"status_code"`
	ContentType   string    `json:"content_type"`
	FetchMS       int64     `json:"fetch_ms"`
	SizeBytes     int64     `json:"size_bytes"`
	ErrorClass    string    `json:"error_class,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	DiscoveredAt  time.Time `json:"discovered_at"`
	FetchedAt     time.Time `json:"fetched_at"`
}

type TreeNode struct {
	ID           string `json:"id"`
	ParentPageID string `json:"parent_page_id,omitempty"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	Depth        int    `json:"depth"`
}

type TreeEdge struct {
	ParentPageID string `json:"parent_page_id"`
	ChildPageID  string `json:"child_page_id"`
}

type Summary struct {
	Status        string     `json:"status"`
	PagesFetched  int        `json:"pages_fetched"`
	PagesFailed   int        `json:"pages_failed"`
	PagesQueued   int        `json:"pages_queued"`
	LastFetchedAt *time.Time `json:"last_fetched_at,omitempty"`
}

type Diagnostics struct {
	SearchSeed    *SearchSeed       `json:"search_seed,omitempty"`
	SearchLog     []DiagnosticEntry `json:"search_log"`
	SkippedURLs   []DiagnosticEntry `json:"skipped_urls"`
	RetryEvents   []DiagnosticEntry `json:"retry_events"`
	Errors        []DiagnosticEntry `json:"errors"`
	FetchLog      []DiagnosticEntry `json:"fetch_log"`
	ArtifactDir   string            `json:"artifact_dir"`
	ArtifactFiles map[string]string `json:"artifact_files"`
}

type DiagnosticEntry struct {
	At       time.Time `json:"at"`
	URL      string    `json:"url,omitempty"`
	PageID   string    `json:"page_id,omitempty"`
	ParentID string    `json:"parent_id,omitempty"`
	Reason   string    `json:"reason"`
	Detail   string    `json:"detail,omitempty"`
}

type EventFrame struct {
	Ts        time.Time  `json:"ts"`
	Status    string     `json:"status"`
	Summary   Summary    `json:"summary"`
	NewPages  []Page     `json:"new_pages,omitempty"`
	TreeNodes []TreeNode `json:"tree_nodes,omitempty"`
	TreeEdges []TreeEdge `json:"tree_edges,omitempty"`
	Latest    *Page      `json:"latest,omitempty"`
}

type Snapshot struct {
	RunID       string            `json:"run_id"`
	RootPageID  string            `json:"root_page_id,omitempty"`
	Config      RunConfig         `json:"config"`
	Seed        SearchSeed        `json:"seed"`
	Status      string            `json:"status"`
	StopReason  string            `json:"stop_reason,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	StoppedAt   *time.Time        `json:"stopped_at,omitempty"`
	Summary     Summary           `json:"summary"`
	Pages       []Page            `json:"pages"`
	TreeNodes   []TreeNode        `json:"tree_nodes"`
	TreeEdges   []TreeEdge        `json:"tree_edges"`
	Diagnostics Diagnostics       `json:"diagnostics"`
	Paths       map[string]string `json:"paths"`
}
