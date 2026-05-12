package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aattwwss/yabatasg/internal/store"
)

type Config struct {
	store *store.Store
}

func NewConfig(s *store.Store) *Config {
	return &Config{store: s}
}

func (c *Config) Get(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization"})
		return
	}

	config, err := c.store.GetConfig(token)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		return
	}
	if err != nil {
		slog.Error("get config failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}

	// Return empty array instead of null for empty configs.
	if config == "" || config == "[]" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(config))
}

func (c *Config) Put(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization"})
		return
	}

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Validate it's an array.
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Config must be a JSON array"})
		return
	}
	if arr == nil {
		raw = []byte("[]")
	}

	config := string(raw)
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err == nil {
		config = compact.String()
	}

	if err := c.store.SetConfig(token, config); err == sql.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		return
	} else if err != nil {
		slog.Error("put config failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (c *Config) Delete(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization"})
		return
	}

	// Clear config but keep the user — other devices may share the same phrase.
	if err := c.store.SetConfig(token, "[]"); err == sql.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		return
	} else if err != nil {
		slog.Error("clear config failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
