package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aattwwss/yabatasg/internal/store"
)

type Auth struct {
	store *store.Store
}

func NewAuth(s *store.Store) *Auth {
	return &Auth{store: s}
}

type registerReq struct {
	Config string `json:"config"`
}

type registerResp struct {
	Phrase string `json:"phrase"`
	Token  string `json:"token"`
}

type linkReq struct {
	Phrase string `json:"phrase"`
}

type linkResp struct {
	Token string `json:"token"`
}

type meResp struct {
	Phrase    string `json:"phrase"`
	CreatedAt string `json:"createdAt"`
}

func (a *Auth) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	user, err := a.store.RegisterUser(req.Config)
	if err != nil {
		slog.Error("register user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create account"})
		return
	}

	writeJSON(w, http.StatusCreated, registerResp{
		Phrase: user.Phrase,
		Token:  user.Token,
	})
}

func (a *Auth) Link(w http.ResponseWriter, r *http.Request) {
	var req linkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Phrase == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Phrase is required"})
		return
	}

	req.Phrase = strings.ToLower(strings.TrimSpace(req.Phrase))

	user, err := a.store.UserByPhrase(req.Phrase)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "No account found with that phrase"})
		return
	}
	if err != nil {
		slog.Error("link lookup failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}

	writeJSON(w, http.StatusOK, linkResp{Token: user.Token})
}

func (a *Auth) Me(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization"})
		return
	}

	user, err := a.store.UserByToken(token)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		return
	}
	if err != nil {
		slog.Error("auth check failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}

	writeJSON(w, http.StatusOK, meResp{
		Phrase:    user.Phrase,
		CreatedAt: user.CreatedAt.UTC().Format("2006-01-02"),
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	t, ok := strings.CutPrefix(h, "Bearer ")
	if !ok {
		return ""
	}
	return t
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
