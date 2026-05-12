package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aattwwss/yabatasg/internal/store"
)

func register(t *testing.T, s *store.Store) string {
	t.Helper()
	a := NewAuth(s)
	rec := httptest.NewRecorder()
	a.Register(rec, httptest.NewRequest("POST", "/register", nil))
	var resp registerResp
	json.NewDecoder(rec.Body).Decode(&resp)
	return resp.Token
}

func TestConfigGet(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	token := register(t, s)
	c := NewConfig(s)

	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Default config should be "[]".
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Errorf("expected '[]', got %q", body)
	}
}

func TestConfigGetNoAuth(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	c := NewConfig(s)
	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	rec := httptest.NewRecorder()
	c.Get(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestConfigPutAndGet(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	token := register(t, s)
	c := NewConfig(s)

	cfg := `[{"name":"Work","shortcuts":[{"service":"188","stopNumber":"43219","name":"Home"}]}]`
	putReq := httptest.NewRequest("PUT", "/api/v1/config", strings.NewReader(cfg))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	c.Put(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT failed: %d %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest("GET", "/api/v1/config", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	c.Get(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET failed: %d", getRec.Code)
	}

	// Verify it parses as valid JSON array.
	var arr []map[string]any
	if err := json.NewDecoder(getRec.Body).Decode(&arr); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 group, got %d", len(arr))
	}
	if arr[0]["name"] != "Work" {
		t.Errorf("expected group name 'Work', got %q", arr[0]["name"])
	}
}

func TestConfigPutInvalidJSON(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	token := register(t, s)
	c := NewConfig(s)

	req := httptest.NewRequest("PUT", "/api/v1/config", strings.NewReader(`{"not":"an array"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c.Put(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestConfigPutNullBecomesEmpty(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	token := register(t, s)
	c := NewConfig(s)

	req := httptest.NewRequest("PUT", "/api/v1/config", strings.NewReader(`null`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c.Put(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestConfigPutNoAuth(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	c := NewConfig(s)
	req := httptest.NewRequest("PUT", "/api/v1/config", strings.NewReader(`[]`))
	rec := httptest.NewRecorder()
	c.Put(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestConfigDelete(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	token := register(t, s)
	c := NewConfig(s)

	// Set some config first.
	cfg := `[{"name":"Work","shortcuts":[]}]`
	putReq := httptest.NewRequest("PUT", "/api/v1/config", strings.NewReader(cfg))
	putReq.Header.Set("Authorization", "Bearer "+token)
	c.Put(httptest.NewRecorder(), putReq)

	// Delete (clear config).
	req := httptest.NewRequest("DELETE", "/api/v1/config", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Token should still work — only config is cleared, user stays for other devices.
	getReq := httptest.NewRequest("GET", "/api/v1/config", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	c.Get(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Errorf("expected 200 after delete, got %d", getRec.Code)
	}
	body := strings.TrimSpace(getRec.Body.String())
	if body != "[]" {
		t.Errorf("expected '[]', got %q", body)
	}
}

func TestConfigDeleteNoAuth(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	c := NewConfig(s)
	req := httptest.NewRequest("DELETE", "/api/v1/config", nil)
	rec := httptest.NewRecorder()
	c.Delete(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"valid", "Bearer abc123", "abc123"},
		{"empty", "", ""},
		{"no bearer", "Basic abc123", ""},
		{"lowercase", "bearer abc123", ""},
		{"missing value", "Bearer ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := bearerToken(req)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
