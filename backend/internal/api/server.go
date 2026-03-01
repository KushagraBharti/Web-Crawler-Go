package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"webcrawler/internal/crawler"
	"webcrawler/internal/storage"
	"webcrawler/internal/util"
)

type ServerOptions struct {
	AllowedOrigins       []string
	AllowedPreviewSuffix string
	StorageMode          string
	ReadyCheck           func(ctx context.Context) error
	RunCreateRateLimit   int
	RunCreateRateWindow  time.Duration
}

type Server struct {
	router               chi.Router
	runManager           *RunManager
	allowedOrigins       []string
	allowedPreviewSuffix string
	storageMode          string
	readyCheck           func(ctx context.Context) error
	runCreateLimiter     *IPRateLimiter
	targetPolicy         *crawler.TargetPolicy
}

func NewServer(runManager *RunManager, opts ServerOptions) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	readyCheck := opts.ReadyCheck
	if readyCheck == nil {
		readyCheck = func(context.Context) error { return nil }
	}

	s := &Server{
		router:               r,
		runManager:           runManager,
		allowedOrigins:       opts.AllowedOrigins,
		allowedPreviewSuffix: strings.TrimSpace(opts.AllowedPreviewSuffix),
		storageMode:          opts.StorageMode,
		readyCheck:           readyCheck,
		runCreateLimiter:     NewIPRateLimiter(opts.RunCreateRateLimit, opts.RunCreateRateWindow),
		targetPolicy:         crawler.NewTargetPolicy(10 * time.Minute),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(s.cors)

	s.router.With(s.rateLimitRunCreate).Post("/runs", s.handleCreateRun)
	s.router.Post("/runs/{id}/start", s.handleStartRun)
	s.router.Post("/runs/{id}/stop", s.handleStopRun)
	s.router.Get("/runs/{id}", s.handleGetRun)
	s.router.Get("/runs/{id}/pages", s.handleListPages)
	s.router.Get("/runs/{id}/events", s.handleEvents)
	s.router.Get("/healthz", s.handleHealth)
	s.router.Get("/readyz", s.handleReady)

	s.router.Handle("/metrics", promhttp.Handler())
	// pprof via DefaultServeMux
	s.router.Mount("/debug/pprof", http.DefaultServeMux)
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := s.isOriginAllowed(origin)
		if origin != "" {
			w.Header().Set("Vary", "Origin")
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			if origin != "" && !allowed {
				writeAPIError(w, http.StatusForbidden, "cors_forbidden", "origin is not allowed", nil)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if origin != "" && !allowed {
			writeAPIError(w, http.StatusForbidden, "cors_forbidden", "origin is not allowed", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) isOriginAllowed(origin string) bool {
	if strings.TrimSpace(origin) == "" {
		return true
	}
	for _, allowed := range s.allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	if s.allowedPreviewSuffix == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.HasSuffix(host, strings.ToLower(s.allowedPreviewSuffix))
}

func (s *Server) rateLimitRunCreate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := clientIPFromRequest(r)
		allowed, retryAfter := s.runCreateLimiter.Allow(clientIP)
		if !allowed {
			if retryAfter > 0 {
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			}
			writeAPIError(w, http.StatusTooManyRequests, "rate_limited", "too many run creation requests", map[string]any{
				"retry_after_seconds": int(retryAfter.Seconds()),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

type createRunRequest struct {
	SeedURL            string `json:"seed_url"`
	MaxDepth           int    `json:"max_depth"`
	MaxPages           int    `json:"max_pages"`
	TimeBudgetSeconds  int    `json:"time_budget_seconds"`
	MaxLinksPerPage    int    `json:"max_links_per_page"`
	GlobalConcurrency  int    `json:"global_concurrency"`
	PerHostConcurrency int    `json:"per_host_concurrency"`
	UserAgent          string `json:"user_agent"`
	RespectRobots      *bool  `json:"respect_robots"`
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req createRunRequest
	if err := util.DecodeJSON(r, &req); err != nil {
		writeValidationError(w, "invalid request body", map[string]any{"body": err.Error()})
		return
	}
	if strings.TrimSpace(req.SeedURL) == "" {
		writeValidationError(w, "seed_url is required", map[string]any{"seed_url": "seed_url is required"})
		return
	}

	canonicalSeed, parsedSeed, err := crawler.Canonicalize(req.SeedURL)
	if err != nil {
		writeValidationError(w, "invalid seed_url", map[string]any{"seed_url": "must be a valid http or https URL"})
		return
	}
	if err := s.targetPolicy.ValidateURL(r.Context(), parsedSeed); err != nil {
		writeTargetPolicyError(w, err)
		return
	}

	cfg, fieldErrors := buildValidatedRunConfig(req, s.runManager.defaults)
	if len(fieldErrors) > 0 {
		writeValidationError(w, "invalid run configuration", fieldErrors)
		return
	}
	cfg.SeedURL = canonicalSeed

	id, err := s.runManager.CreateRun(r.Context(), cfg)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "create_run_failed", "failed to create run", nil)
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]any{"id": id.String(), "status": "created", "created_at": time.Now().UTC()})
}

func (s *Server) handleStartRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeValidationError(w, "invalid run id", map[string]any{"id": "must be a valid UUID"})
		return
	}
	if err := s.runManager.StartRun(r.Context(), id); err != nil {
		writeAPIError(w, http.StatusBadRequest, "start_run_failed", err.Error(), nil)
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

func (s *Server) handleStopRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeValidationError(w, "invalid run id", map[string]any{"id": "must be a valid UUID"})
		return
	}
	if err := s.runManager.StopRun(r.Context(), id); err != nil {
		writeAPIError(w, http.StatusBadRequest, "stop_run_failed", err.Error(), nil)
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeValidationError(w, "invalid run id", map[string]any{"id": "must be a valid UUID"})
		return
	}
	state, err := s.runManager.GetRun(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "run_not_found", "run not found", nil)
		return
	}

	stats := map[string]any{}
	if state.Engine != nil {
		stats["pages_fetched"] = state.Engine.PagesFetched()
	}
	stopReason := state.StopReason
	if stopReason == "" && state.Status == "running" {
		stopReason = "running"
	}
	summary, summaryErr := s.runManager.Summary(r.Context(), id)
	if summaryErr != nil {
		summary = storage.RunSummary{}
	}

	payload := map[string]any{
		"id":           state.ID.String(),
		"status":       state.Status,
		"created_at":   state.CreatedAt,
		"started_at":   state.StartedAt,
		"stopped_at":   state.StoppedAt,
		"storage_mode": s.storageMode,
		"stop_reason":  stopReason,
		"limits": map[string]any{
			"max_depth":           state.Config.MaxDepth,
			"max_pages":           state.Config.MaxPages,
			"time_budget_seconds": int(state.Config.TimeBudget.Seconds()),
		},
		"summary": map[string]any{
			"pages_fetched":   summary.PagesFetched,
			"pages_failed":    summary.PagesFailed,
			"unique_hosts":    summary.UniqueHosts,
			"total_bytes":     summary.TotalBytes,
			"last_fetched_at": summary.LastFetchedAt,
		},
		"stats": stats,
	}
	util.WriteJSON(w, http.StatusOK, payload)
}

func (s *Server) handleListPages(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeValidationError(w, "invalid run id", map[string]any{"id": "must be a valid UUID"})
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	pages, err := s.runManager.ListPages(r.Context(), id, limit)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "list_pages_failed", "failed to list pages", nil)
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]any{"items": pages})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeValidationError(w, "invalid run id", map[string]any{"id": "must be a valid UUID"})
		return
	}
	telemetry, ok := s.runManager.TelemetryFor(id)
	if !ok {
		state, stateErr := s.runManager.GetRun(r.Context(), id)
		if stateErr != nil {
			writeAPIError(w, http.StatusNotFound, "run_not_found", "run not found", nil)
			return
		}
		writeAPIError(w, http.StatusConflict, "run_not_active", "run is not active", map[string]any{
			"status":      state.Status,
			"stop_reason": state.StopReason,
		})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "stream_unsupported", "streaming unsupported", nil)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := telemetry.Subscribe()
	defer unsubscribe()

	enc := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case frame, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write([]byte("event: frame\n"))
			_, _ = w.Write([]byte("data: "))
			if err := enc.Encode(frame); err != nil {
				return
			}
			_, _ = w.Write([]byte("\n"))
			flusher.Flush()
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	util.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.readyCheck(ctx); err != nil {
		writeAPIError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready", map[string]any{"reason": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ready",
		"time":   time.Now().UTC(),
	})
}

func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	if ip := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); ip != nil {
		return ip.String()
	}
	return "unknown"
}
