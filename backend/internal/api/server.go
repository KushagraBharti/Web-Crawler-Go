package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"webcrawler/internal/crawler"
	"webcrawler/internal/util"
)

type Server struct {
	router        chi.Router
	runManager    *RunManager
	allowedOrigin string
}

func NewServer(runManager *RunManager, allowedOrigin string) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	s := &Server{router: r, runManager: runManager, allowedOrigin: allowedOrigin}
	s.routes()
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) routes() {
	s.router.Use(s.cors)

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		util.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.router.Get("/runs", s.handleListRuns)
	s.router.Post("/runs", s.handleCreateRun)
	s.router.Post("/runs/{id}/start", s.handleStartRun)
	s.router.Post("/runs/{id}/stop", s.handleStopRun)
	s.router.Delete("/runs/{id}", s.handleDeleteRun)
	s.router.Get("/runs/{id}", s.handleGetRun)
	s.router.Get("/runs/{id}/pages", s.handleListPages)
	s.router.Get("/runs/{id}/pages/{pageId}", s.handleGetPage)
	s.router.Get("/runs/{id}/tree", s.handleGetTree)
	s.router.Get("/runs/{id}/diagnostics", s.handleGetDiagnostics)
	s.router.Get("/runs/{id}/events", s.handleEvents)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	util.WriteJSON(w, http.StatusOK, map[string]any{"items": s.runManager.ListRuns()})
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req CreateRunInput
	if err := util.DecodeJSON(r, &req); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Input) == "" {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "input required"})
		return
	}
	if req.Mode == "" {
		req.Mode = crawler.SourceModeURL
	}
	run, err := s.runManager.CreateRun(r.Context(), req)
	if err != nil {
		log.Printf("POST /runs failed mode=%s input=%q err=%v", req.Mode, req.Input, err)
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]any{
		"id":         run.ID,
		"status":     run.Status,
		"created_at": run.CreatedAt,
		"seed":       run.Seed,
		"config":     run.Config,
	})
}

func (s *Server) handleStartRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req StartRunInput
	if r.Body != nil && r.Body != http.NoBody {
		if err := util.DecodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
			util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	if err := s.runManager.StartRun(r.Context(), id, req); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

func (s *Server) handleStopRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.runManager.StopRun(id); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

func (s *Server) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.runManager.DeleteRun(id); err != nil {
		util.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := s.runManager.GetRun(id)
	if err != nil {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, run)
}

func (s *Server) handleListPages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pages, err := s.runManager.ListPages(id)
	if err != nil {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]any{"items": pages})
}

func (s *Server) handleGetPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pageID := chi.URLParam(r, "pageId")
	page, err := s.runManager.GetPage(id, pageID)
	if err != nil {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, page)
}

func (s *Server) handleGetTree(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tree, err := s.runManager.GetTree(id)
	if err != nil {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, tree)
}

func (s *Server) handleGetDiagnostics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	diagnostics, err := s.runManager.GetDiagnostics(id)
	if err != nil {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	util.WriteJSON(w, http.StatusOK, diagnostics)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, unsubscribe, err := s.runManager.Subscribe(id)
	if err != nil {
		util.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	defer unsubscribe()

	flusher, ok := w.(http.Flusher)
	if !ok {
		util.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

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
