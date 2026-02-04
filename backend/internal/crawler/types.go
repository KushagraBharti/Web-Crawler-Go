package crawler

import "time"

type RunConfig struct {
	SeedURL            string        `json:"seed_url"`
	MaxDepth           int           `json:"max_depth"`
	MaxPages           int           `json:"max_pages"`
	TimeBudget         time.Duration `json:"time_budget"`
	TimeBudgetSeconds  int           `json:"time_budget_seconds"`
	MaxLinksPerPage    int           `json:"max_links_per_page"`
	GlobalConcurrency  int           `json:"global_concurrency"`
	PerHostConcurrency int           `json:"per_host_concurrency"`
	UserAgent          string        `json:"user_agent"`
	RespectRobots      bool          `json:"respect_robots"`
	RequestTimeout     time.Duration `json:"request_timeout"`
	HeaderTimeout      time.Duration `json:"header_timeout"`
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout"`
	IdleConnTimeout    time.Duration `json:"idle_conn_timeout"`
	MaxBodyBytes       int64         `json:"max_body_bytes"`
	RobotsTTL          time.Duration `json:"robots_ttl"`
	RetryMax           int           `json:"retry_max"`
	RetryBaseDelay     time.Duration `json:"retry_base_delay"`
	CircuitTripCount   int           `json:"circuit_trip_count"`
	CircuitResetTime   time.Duration `json:"circuit_reset_time"`
}

func (c RunConfig) Normalize() RunConfig {
	if c.TimeBudget == 0 && c.TimeBudgetSeconds > 0 {
		c.TimeBudget = time.Duration(c.TimeBudgetSeconds) * time.Second
	}
	return c
}

type Task struct {
	URL        string
	Canonical  string
	Host       string
	Depth      int
	NotBefore  time.Time
	Retries    int
	SourceHost string
	DiscoveredAt time.Time
	Permit     *Permit
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

type FetchResult struct {
	Task         *Task
	StatusCode   int
	ContentType  string
	Body         []byte
	FetchMS      int64
	SizeBytes    int64
	ReusedConn   bool
	ErrClass     string
	ErrMessage   string
	RedirectURL  string
	RedirectHost string
}
