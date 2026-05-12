package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aattwwss/yabatasg/internal/store"
)

func TestAuthRegister(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)

	body := strings.NewReader(`{"config":"[{\"name\":\"Work\"}]"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", body)
	rec := httptest.NewRecorder()
	a.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp registerResp
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Phrase == "" {
		t.Error("expected non-empty phrase")
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}

	// Verify it has 4 hyphen-separated words.
	parts := strings.Split(resp.Phrase, "-")
	if len(parts) != 4 {
		t.Errorf("expected 4-word phrase, got %q", resp.Phrase)
	}
}

func TestAuthRegisterEmptyConfig(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", nil)
	rec := httptest.NewRecorder()
	a.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

func TestAuthLink(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)

	// Create a user first.
	rec := httptest.NewRecorder()
	a.Register(rec, httptest.NewRequest("POST", "/register", nil))
	var reg registerResp
	json.NewDecoder(rec.Body).Decode(&reg)

	// Link with the phrase.
	linkBody := strings.NewReader(`{"phrase":"` + reg.Phrase + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/link", linkBody)
	rec = httptest.NewRecorder()
	a.Link(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp linkResp
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Token != reg.Token {
		t.Errorf("expected same token %q, got %q", reg.Token, resp.Token)
	}
}

func TestAuthLinkCaseInsensitive(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)
	rec := httptest.NewRecorder()
	a.Register(rec, httptest.NewRequest("POST", "/register", nil))
	var reg registerResp
	json.NewDecoder(rec.Body).Decode(&reg)

	upperPhrase := strings.ToUpper(reg.Phrase)
	linkBody := strings.NewReader(`{"phrase":"` + upperPhrase + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/link", linkBody)
	rec = httptest.NewRecorder()
	a.Link(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthLinkNotFound(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)
	body := strings.NewReader(`{"phrase":"no-such-phrase-here"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/link", body)
	rec := httptest.NewRecorder()
	a.Link(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAuthLinkEmptyPhrase(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)
	body := strings.NewReader(`{"phrase":""}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/link", body)
	rec := httptest.NewRecorder()
	a.Link(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAuthMe(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)
	rec := httptest.NewRecorder()
	a.Register(rec, httptest.NewRequest("POST", "/register", nil))
	var reg registerResp
	json.NewDecoder(rec.Body).Decode(&reg)

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+reg.Token)
	rec = httptest.NewRecorder()
	a.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp meResp
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Phrase != reg.Phrase {
		t.Errorf("expected phrase %q, got %q", reg.Phrase, resp.Phrase)
	}
}

func TestAuthMeNoHeader(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	a.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMeInvalidToken(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a := NewAuth(s)
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()
	a.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
