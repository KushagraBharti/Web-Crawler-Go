package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Store interface {
	Migrate(ctx context.Context) error
	CreateRun(ctx context.Context, cfg RunConfig) (uuid.UUID, error)
	UpdateRunStatus(ctx context.Context, id uuid.UUID, status string, startedAt, stoppedAt *time.Time, stopReason *string) error
	GetRun(ctx context.Context, id uuid.UUID) (RunRow, error)
	GetRunSummary(ctx context.Context, id uuid.UUID) (RunSummary, error)
	ListPages(ctx context.Context, id uuid.UUID, limit int) ([]PageRow, error)
	InsertPage(ctx context.Context, rec PageRecord) error
	InsertError(ctx context.Context, runID uuid.UUID, host, url, class, message string) error
	UpsertEdge(ctx context.Context, runID uuid.UUID, src, dst string, count int) error
	UpsertHostStat(ctx context.Context, runID uuid.UUID, host string, bucket time.Time, req, errCount, p50, p95 int, bytes int64, reuse float64) error
}

type SQLStore struct {
	db *sql.DB
}

func NewSQL(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) DB() *sql.DB {
	return s.db
}

func (s *SQLStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS runs (
			id uuid PRIMARY KEY,
			seed_url text NOT NULL,
			status text NOT NULL,
			created_at timestamptz NOT NULL,
			started_at timestamptz,
			stopped_at timestamptz,
			stop_reason text,
			max_depth int,
			max_pages int,
			time_budget_seconds int,
			max_links_per_page int,
			global_concurrency int,
			per_host_concurrency int,
			user_agent text,
			respect_robots boolean
		);`,
		`CREATE INDEX IF NOT EXISTS runs_status_idx ON runs(status);`,
		`ALTER TABLE runs ADD COLUMN IF NOT EXISTS stop_reason text;`,
		`CREATE TABLE IF NOT EXISTS pages (
			id bigserial PRIMARY KEY,
			run_id uuid REFERENCES runs(id),
			url text NOT NULL,
			canonical_url text NOT NULL,
			host text NOT NULL,
			depth int NOT NULL,
			status_code int,
			content_type text,
			fetch_ms int,
			size_bytes bigint,
			error_class text,
			error_message text,
			discovered_at timestamptz NOT NULL,
			fetched_at timestamptz
		);`,
		`CREATE INDEX IF NOT EXISTS pages_run_id_idx ON pages(run_id);`,
		`CREATE INDEX IF NOT EXISTS pages_host_idx ON pages(run_id, host);`,
		`CREATE INDEX IF NOT EXISTS pages_canonical_idx ON pages(run_id, canonical_url);`,
		`CREATE TABLE IF NOT EXISTS hosts (
			run_id uuid REFERENCES runs(id),
			host text NOT NULL,
			robots_state text,
			circuit_state text,
			inflight int,
			last_error_at timestamptz,
			last_429_at timestamptz,
			PRIMARY KEY (run_id, host)
		);`,
		`CREATE TABLE IF NOT EXISTS host_stats (
			run_id uuid REFERENCES runs(id),
			host text NOT NULL,
			bucket_start timestamptz NOT NULL,
			req_count int,
			err_count int,
			p50_ms int,
			p95_ms int,
			bytes bigint,
			reuse_rate float,
			PRIMARY KEY (run_id, host, bucket_start)
		);`,
		`CREATE TABLE IF NOT EXISTS edges (
			run_id uuid REFERENCES runs(id),
			src_host text NOT NULL,
			dst_host text NOT NULL,
			count int NOT NULL,
			PRIMARY KEY (run_id, src_host, dst_host)
		);`,
		`CREATE TABLE IF NOT EXISTS errors (
			id bigserial PRIMARY KEY,
			run_id uuid REFERENCES runs(id),
			host text,
			url text,
			class text NOT NULL,
			message text,
			at timestamptz NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS errors_run_id_idx ON errors(run_id);`,
		`CREATE INDEX IF NOT EXISTS errors_class_idx ON errors(run_id, class);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

type RunConfig struct {
	SeedURL            string
	MaxDepth           int
	MaxPages           int
	TimeBudgetSeconds  int
	MaxLinksPerPage    int
	GlobalConcurrency  int
	PerHostConcurrency int
	UserAgent          string
	RespectRobots      bool
}

func (s *SQLStore) CreateRun(ctx context.Context, cfg RunConfig) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO runs (id, seed_url, status, created_at, max_depth, max_pages, time_budget_seconds, max_links_per_page, global_concurrency, per_host_concurrency, user_agent, respect_robots)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		id, cfg.SeedURL, "created", time.Now(), cfg.MaxDepth, cfg.MaxPages, cfg.TimeBudgetSeconds, cfg.MaxLinksPerPage, cfg.GlobalConcurrency, cfg.PerHostConcurrency, cfg.UserAgent, cfg.RespectRobots,
	)
	return id, err
}

func (s *SQLStore) UpdateRunStatus(ctx context.Context, id uuid.UUID, status string, startedAt, stoppedAt *time.Time, stopReason *string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE runs SET status=$1, started_at=COALESCE($2, started_at), stopped_at=COALESCE($3, stopped_at), stop_reason=COALESCE($4, stop_reason) WHERE id=$5`, status, startedAt, stoppedAt, nullableStringPtr(stopReason), id)
	return err
}

type RunRow struct {
	ID                 uuid.UUID
	SeedURL            string
	Status             string
	CreatedAt          time.Time
	StartedAt          sql.NullTime
	StoppedAt          sql.NullTime
	StopReason         sql.NullString
	MaxDepth           int
	MaxPages           int
	TimeBudgetSeconds  int
	MaxLinksPerPage    int
	GlobalConcurrency  int
	PerHostConcurrency int
	UserAgent          string
	RespectRobots      bool
}

func (s *SQLStore) GetRun(ctx context.Context, id uuid.UUID) (RunRow, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, seed_url, status, created_at, started_at, stopped_at, stop_reason, max_depth, max_pages, time_budget_seconds, max_links_per_page, global_concurrency, per_host_concurrency, user_agent, respect_robots FROM runs WHERE id=$1`, id)
	var rr RunRow
	err := row.Scan(&rr.ID, &rr.SeedURL, &rr.Status, &rr.CreatedAt, &rr.StartedAt, &rr.StoppedAt, &rr.StopReason, &rr.MaxDepth, &rr.MaxPages, &rr.TimeBudgetSeconds, &rr.MaxLinksPerPage, &rr.GlobalConcurrency, &rr.PerHostConcurrency, &rr.UserAgent, &rr.RespectRobots)
	return rr, err
}

type RunSummary struct {
	PagesFetched  int64
	PagesFailed   int64
	UniqueHosts   int64
	TotalBytes    int64
	LastFetchedAt *time.Time
}

type PageRow struct {
	URL          string
	Host         string
	Depth        int
	StatusCode   int
	ContentType  string
	FetchMS      int64
	SizeBytes    int64
	ErrorClass   string
	ErrorMessage string
	FetchedAt    *time.Time
}

func (s *SQLStore) GetRunSummary(ctx context.Context, id uuid.UUID) (RunSummary, error) {
	row := s.db.QueryRowContext(ctx, `SELECT
		COALESCE(COUNT(*) FILTER (WHERE error_class IS NULL), 0) AS pages_fetched,
		COALESCE(COUNT(*) FILTER (WHERE error_class IS NOT NULL), 0) AS pages_failed,
		COALESCE(COUNT(DISTINCT host), 0) AS unique_hosts,
		COALESCE(SUM(size_bytes), 0) AS total_bytes,
		MAX(fetched_at) AS last_fetched_at
		FROM pages WHERE run_id=$1`, id)
	var summary RunSummary
	var lastFetched sql.NullTime
	if err := row.Scan(&summary.PagesFetched, &summary.PagesFailed, &summary.UniqueHosts, &summary.TotalBytes, &lastFetched); err != nil {
		return RunSummary{}, err
	}
	if lastFetched.Valid {
		summary.LastFetchedAt = &lastFetched.Time
	}
	return summary, nil
}

func (s *SQLStore) ListPages(ctx context.Context, id uuid.UUID, limit int) ([]PageRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT url, host, depth, status_code, content_type, fetch_ms, size_bytes, error_class, error_message, fetched_at
		FROM pages WHERE run_id=$1
		ORDER BY fetched_at DESC NULLS LAST, discovered_at DESC
		LIMIT $2`, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PageRow
	for rows.Next() {
		var row PageRow
		var status sql.NullInt32
		var ct sql.NullString
		var fetchMS sql.NullInt32
		var size sql.NullInt64
		var errClass sql.NullString
		var errMsg sql.NullString
		var fetched sql.NullTime
		if err := rows.Scan(&row.URL, &row.Host, &row.Depth, &status, &ct, &fetchMS, &size, &errClass, &errMsg, &fetched); err != nil {
			return nil, err
		}
		if status.Valid {
			row.StatusCode = int(status.Int32)
		}
		if ct.Valid {
			row.ContentType = ct.String
		}
		if fetchMS.Valid {
			row.FetchMS = int64(fetchMS.Int32)
		}
		if size.Valid {
			row.SizeBytes = size.Int64
		}
		if errClass.Valid {
			row.ErrorClass = errClass.String
		}
		if errMsg.Valid {
			row.ErrorMessage = errMsg.String
		}
		if fetched.Valid {
			row.FetchedAt = &fetched.Time
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type PageRecord struct {
	RunID        uuid.UUID
	URL          string
	CanonicalURL string
	Host         string
	Depth        int
	StatusCode   int
	ContentType  string
	FetchMS      int64
	SizeBytes    int64
	ErrClass     string
	ErrMessage   string
	DiscoveredAt time.Time
	FetchedAt    *time.Time
}

func (s *SQLStore) InsertPage(ctx context.Context, rec PageRecord) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO pages (run_id, url, canonical_url, host, depth, status_code, content_type, fetch_ms, size_bytes, error_class, error_message, discovered_at, fetched_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		rec.RunID, rec.URL, rec.CanonicalURL, rec.Host, rec.Depth, nullableInt(rec.StatusCode), nullableString(rec.ContentType), nullableInt(int(rec.FetchMS)), nullableInt64(rec.SizeBytes), nullableString(rec.ErrClass), nullableString(rec.ErrMessage), rec.DiscoveredAt, rec.FetchedAt,
	)
	return err
}

func (s *SQLStore) InsertError(ctx context.Context, runID uuid.UUID, host, url, class, message string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO errors (run_id, host, url, class, message, at) VALUES ($1,$2,$3,$4,$5,$6)`, runID, nullableString(host), nullableString(url), class, nullableString(message), time.Now())
	return err
}

func (s *SQLStore) UpsertEdge(ctx context.Context, runID uuid.UUID, src, dst string, count int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO edges (run_id, src_host, dst_host, count) VALUES ($1,$2,$3,$4)
	ON CONFLICT (run_id, src_host, dst_host) DO UPDATE SET count = edges.count + EXCLUDED.count`, runID, src, dst, count)
	return err
}

func (s *SQLStore) UpsertHostStat(ctx context.Context, runID uuid.UUID, host string, bucket time.Time, req, errCount, p50, p95 int, bytes int64, reuse float64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO host_stats (run_id, host, bucket_start, req_count, err_count, p50_ms, p95_ms, bytes, reuse_rate)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	ON CONFLICT (run_id, host, bucket_start) DO UPDATE SET req_count=$4, err_count=$5, p50_ms=$6, p95_ms=$7, bytes=$8, reuse_rate=$9`,
		runID, host, bucket, req, errCount, p50, p95, bytes, reuse)
	return err
}

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullableStringPtr(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullableInt(n int) sql.NullInt32 {
	if n == 0 {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(n), Valid: true}
}

func nullableInt64(n int64) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: n, Valid: true}
}
