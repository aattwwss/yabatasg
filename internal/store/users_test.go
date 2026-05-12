package store

import (
	"database/sql"
	"testing"
)

func TestRegisterUser(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	user, err := s.RegisterUser("")
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}
	if user.ID == "" {
		t.Error("expected non-empty ID")
	}
	if user.Phrase == "" {
		t.Error("expected non-empty phrase")
	}
	if user.Token == "" {
		t.Error("expected non-empty token")
	}
	if user.Config != "[]" {
		t.Errorf("expected default config '[]', got %q", user.Config)
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}

	// Phrases should be unique.
	user2, err := s.RegisterUser("")
	if err != nil {
		t.Fatalf("second RegisterUser failed: %v", err)
	}
	if user.Phrase == user2.Phrase {
		t.Error("expected different phrases for different users")
	}
	if user.Token == user2.Token {
		t.Error("expected different tokens for different users")
	}
}

func TestRegisterUserWithConfig(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	cfg := `[{"name":"Work","shortcuts":[{"service":"188","stopNumber":"43219","name":"Home"}]}]`
	user, err := s.RegisterUser(cfg)
	if err != nil {
		t.Fatalf("RegisterUser with config failed: %v", err)
	}
	if user.Config != cfg {
		t.Errorf("expected config %q, got %q", cfg, user.Config)
	}
}

func TestUserByPhrase(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	user, _ := s.RegisterUser("")

	found, err := s.UserByPhrase(user.Phrase)
	if err != nil {
		t.Fatalf("UserByPhrase failed: %v", err)
	}
	if found.Token != user.Token {
		t.Errorf("expected token %q, got %q", user.Token, found.Token)
	}

	_, err = s.UserByPhrase("nonexistent-phrase-zzz")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUserByToken(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	user, _ := s.RegisterUser("")

	found, err := s.UserByToken(user.Token)
	if err != nil {
		t.Fatalf("UserByToken failed: %v", err)
	}
	if found.Phrase != user.Phrase {
		t.Errorf("expected phrase %q, got %q", user.Phrase, found.Phrase)
	}

	_, err = s.UserByToken("nonexistent-token")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetSetConfig(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	user, _ := s.RegisterUser("")

	cfg, err := s.GetConfig(user.Token)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if cfg != "[]" {
		t.Errorf("expected default '[]', got %q", cfg)
	}

	newCfg := `[{"name":"Test"}]`
	if err := s.SetConfig(user.Token, newCfg); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	cfg, err = s.GetConfig(user.Token)
	if err != nil {
		t.Fatalf("GetConfig after set failed: %v", err)
	}
	if cfg != newCfg {
		t.Errorf("expected %q, got %q", newCfg, cfg)
	}
}

func TestSetConfigInvalidToken(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	err = s.SetConfig("no-such-token", "[]")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	user, _ := s.RegisterUser("")

	if err := s.DeleteUser(user.Token); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	_, err = s.UserByToken(user.Token)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestGetConfigInvalidToken(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	_, err = s.GetConfig("no-such-token")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
