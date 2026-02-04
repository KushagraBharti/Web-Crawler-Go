package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"webcrawler/internal/crawler"
	"webcrawler/internal/storage"
	"webcrawler/internal/util"
)

type Server struct {
	router     chi.Router
	runManager *RunManager
	allowedOrigin string
	storageMode string
}

func NewServer(runManager *RunManager, allowedOrigin string, storageMode string) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	s := &Server{router: r, runManager: runManager, allowedOrigin: allowedOrigin, storageMode: storageMode}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(s.cors)

	s.router.Post("/runs", s.handleCreateRun)
	s.router.Post("/runs/{id}/start", s.handleStartRun)
	s.router.Post("/runs/{id}/stop", s.handleStopRun)
	s.router.Get("/runs/{id}", s.handleGetRun)
	s.router.Get("/runs/{id}/pages", s.handleListPages)
	s.router.Get("/runs/{id}/events", s.handleEvents)

	s.router.Handle("/metrics", promhttp.Handler())
	// pprof via DefaultServeMux
	s.router.Mount("/debug/pprof", http.DefaultServeMux)
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
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
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.SeedURL == "" {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "seed_url required"})
		return
	}
	if _, _, err := crawler.Canonicalize(req.SeedURL); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seed_url"})
		return
	}
	cfg := crawler.RunConfig{
		SeedURL:            req.SeedURL,
		MaxDepth:           req.MaxDepth,
		MaxPages:           req.MaxPages,
		TimeBudgetSeconds:  req.TimeBudgetSeconds,
		MaxLinksPerPage:    req.MaxLinksPerPage,
		GlobalConcurrency:  req.GlobalConcurrency,
		PerHostConcurrency: req.PerHostConcurrency,
		UserAgent:          req.UserAgent,
	}
	if req.RespectRobots != nil {
		cfg.RespectRobots = *req.RespectRobots
	} else {
		cfg.RespectRobots = s.runManager.defaults.RespectRobots
	}

	id, err := s.runManager.CreateRun(r.Context(), cfg)
	if err != nil {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]any{"id": id.String(), "status": "created", "created_at": time.Now().UTC()})
}

func (s *Server) handleStartRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := s.runManager.StartRun(r.Context(), id); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

func (s *Server) handleStopRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := s.runManager.StopRun(r.Context(), id); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	state, err := s.runManager.GetRun(r.Context(), id)
	if err != nil {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
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
		"id": state.ID.String(),
		"status": state.Status,
		"created_at": state.CreatedAt,
		"started_at": state.StartedAt,
		"stopped_at": state.StoppedAt,
		"storage_mode": s.storageMode,
		"stop_reason": stopReason,
		"limits": map[string]any{
			"max_depth": state.Config.MaxDepth,
			"max_pages": state.Config.MaxPages,
			"time_budget_seconds": int(state.Config.TimeBudget.Seconds()),
		},
		"summary": map[string]any{
			"pages_fetched":  summary.PagesFetched,
			"pages_failed":   summary.PagesFailed,
			"unique_hosts":   summary.UniqueHosts,
			"total_bytes":    summary.TotalBytes,
			"last_fetched_at": summary.LastFetchedAt,
		},
		"stats": stats,
	}
	util.WriteJSON(w, http.StatusOK, payload)
}

func (s *Server) handleListPages(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
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
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]any{"items": pages})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	telemetry, ok := s.runManager.TelemetryFor(id)
	if !ok {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "run not active"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
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
			w.Write([]byte("event: frame\n"))
			w.Write([]byte("data: "))
			if err := enc.Encode(frame); err != nil {
				return
			}
			w.Write([]byte("\n"))
			flusher.Flush()
		}
	}
}
