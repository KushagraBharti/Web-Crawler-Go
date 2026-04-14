package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"

	"webcrawler/internal/crawler"
)

type Store struct {
	root string
}

func New(root string) *Store {
	return &Store{root: root}
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) RunDir(runID string) string {
	return filepath.Join(s.root, runID)
}

func (s *Store) Paths(runID string) map[string]string {
	runDir := s.RunDir(runID)
	return map[string]string{
		"run":         filepath.Join(runDir, "run.json"),
		"pages":       filepath.Join(runDir, "pages.json"),
		"tree":        filepath.Join(runDir, "tree.json"),
		"diagnostics": filepath.Join(runDir, "diagnostics.json"),
	}
}

func (s *Store) Persist(snapshot crawler.Snapshot) error {
	runDir := s.RunDir(snapshot.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	paths := s.Paths(snapshot.RunID)

	runPayload := map[string]any{
		"run_id":      snapshot.RunID,
		"config":      snapshot.Config,
		"seed":        snapshot.Seed,
		"status":      snapshot.Status,
		"stop_reason": snapshot.StopReason,
		"created_at":  snapshot.CreatedAt,
		"started_at":  snapshot.StartedAt,
		"stopped_at":  snapshot.StoppedAt,
		"summary":     snapshot.Summary,
		"paths":       paths,
	}
	treePayload := map[string]any{
		"nodes": snapshot.TreeNodes,
		"edges": snapshot.TreeEdges,
	}

	if err := writeJSON(paths["run"], runPayload); err != nil {
		return err
	}
	if err := writeJSON(paths["pages"], snapshot.Pages); err != nil {
		return err
	}
	if err := writeJSON(paths["tree"], treePayload); err != nil {
		return err
	}
	if err := writeJSON(paths["diagnostics"], snapshot.Diagnostics); err != nil {
		return err
	}
	return nil
}

func writeJSON(path string, payload any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
