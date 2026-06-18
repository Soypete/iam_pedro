package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type PalaceWrapper struct {
	palaceBase string
	logger     *log.Logger
}

type MineRequest struct {
	Messages []string `json:"messages"`
	Palace   string   `json:"palace"`
	Wing     string   `json:"wing"`
	Room     string   `json:"room"`
	Mode     string   `json:"mode"`
	Agent    string   `json:"agent"`
	Limit    int      `json:"limit"`
	DryRun   bool     `json:"dry_run"`
}

type SearchRequest struct {
	Query   string `json:"query"`
	Palace  string `json:"palace"`
	Wing    string `json:"wing"`
	Room    string `json:"room"`
	Results int    `json:"results"`
}

type RouteRegisterRequest struct {
	EntityPath string `json:"entity_path"`
	Location   string `json:"location"`
	Palace     string `json:"palace"`
}

type RouteResolveRequest struct {
	EntityPath string `json:"entity_path"`
	Palace     string `json:"palace"`
}

func main() {
	addr := getEnv("MEMPALACE_WRAPPER_ADDR", ":8082")
	palaceBase := getEnv("MEMPALACE_DATA_DIR", "/data/palaces")

	wrapper := &PalaceWrapper{
		palaceBase: palaceBase,
		logger:     log.New(os.Stdout, "[mempalace-wrapper] ", log.LstdFlags),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mine", wrapper.handleMine)
	mux.HandleFunc("/search", wrapper.handleSearch)
	mux.HandleFunc("/route/register", wrapper.handleRouteRegister)
	mux.HandleFunc("/route/resolve", wrapper.handleRouteResolve)
	mux.HandleFunc("/healthz", wrapper.handleHealthz)

	wrapper.logger.Printf("Starting mempalace-wrapper on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (p *PalaceWrapper) handleMine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		p.respondError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	var req MineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Palace == "" {
		req.Palace = p.palaceBase
	}

	if len(req.Messages) == 0 {
		p.respondError(w, http.StatusBadRequest, "messages required")
		return
	}

	tempDir, err := os.MkdirTemp("", "mempalace-*")
	if err != nil {
		p.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create temp dir: %v", err))
		return
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	for i, msg := range req.Messages {
		filename := filepath.Join(tempDir, fmt.Sprintf("chat_%d.txt", i))
		if err := os.WriteFile(filename, []byte(msg), 0644); err != nil {
			p.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write temp file: %v", err))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	args := []string{"mine", tempDir, "--palace", req.Palace}
	if req.Wing != "" {
		args = append(args, "--wing", req.Wing)
	}
	if req.Room != "" {
		args = append(args, "--room", req.Room)
	}
	if req.Mode != "" {
		args = append(args, "--mode", req.Mode)
	}
	if req.Agent != "" {
		args = append(args, "--agent", req.Agent)
	}
	if req.Limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.DryRun {
		args = append(args, "--dry-run")
	}

	p.logger.Printf("Running: mempalace %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "mempalace", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.respondError(w, http.StatusInternalServerError, fmt.Sprintf("mempalace error: %v, output: %s", err, string(output)))
		return
	}

	p.respondJSON(w, map[string]interface{}{
		"success": true,
		"output":  string(output),
	})
}

func (p *PalaceWrapper) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		p.respondError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Palace == "" {
		req.Palace = p.palaceBase
	}
	if req.Results == 0 {
		req.Results = 5
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	args := []string{"search", req.Query, "--palace", req.Palace, "--results", fmt.Sprintf("%d", req.Results)}
	if req.Wing != "" {
		args = append(args, "--wing", req.Wing)
	}
	if req.Room != "" {
		args = append(args, "--room", req.Room)
	}

	p.logger.Printf("Running: mempalace %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "mempalace", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.respondError(w, http.StatusInternalServerError, fmt.Sprintf("mempalace error: %v, output: %s", err, string(output)))
		return
	}

	p.respondJSON(w, map[string]interface{}{
		"success": true,
		"output":  string(output),
	})
}

func (p *PalaceWrapper) handleRouteRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		p.respondError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	var req RouteRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Palace == "" {
		req.Palace = p.palaceBase
	}
	if req.EntityPath == "" || req.Location == "" {
		p.respondError(w, http.StatusBadRequest, "entity_path and location required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	args := []string{"route", "register", "--palace", req.Palace, "--entity", req.EntityPath, "--location", req.Location}

	p.logger.Printf("Running: mempalace %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "mempalace", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.respondError(w, http.StatusInternalServerError, fmt.Sprintf("mempalace error: %v, output: %s", err, string(output)))
		return
	}

	p.respondJSON(w, map[string]interface{}{
		"success": true,
		"output":  string(output),
	})
}

func (p *PalaceWrapper) handleRouteResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		p.respondError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	entityPath := r.URL.Query().Get("entity_path")
	palace := r.URL.Query().Get("palace")

	if entityPath == "" {
		p.respondError(w, http.StatusBadRequest, "entity_path required")
		return
	}

	if palace == "" {
		palace = p.palaceBase
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	args := []string{"route", "resolve", "--palace", palace, "--entity", entityPath}

	p.logger.Printf("Running: mempalace %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "mempalace", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.respondError(w, http.StatusInternalServerError, fmt.Sprintf("mempalace error: %v, output: %s", err, string(output)))
		return
	}

	p.respondJSON(w, map[string]interface{}{
		"success": true,
		"result":  string(output),
	})
}

func (p *PalaceWrapper) handleHealthz(w http.ResponseWriter, r *http.Request) {
	p.respondJSON(w, map[string]interface{}{
		"status": "ok",
	})
}

func (p *PalaceWrapper) respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		p.logger.Printf("error encoding response: %v", err)
	}
}

func (p *PalaceWrapper) respondError(w http.ResponseWriter, code int, msg string) {
	p.logger.Printf("error: %s", msg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

var _ = io.Discard
