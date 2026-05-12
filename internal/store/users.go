package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/aattwwss/yabatasg/internal/auth"
)

type User struct {
	ID        string
	Phrase    string
	Token     string
	Config    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Store) RegisterUser(initialConfig string) (*User, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}
	phrase, err := auth.NewPhrase()
	if err != nil {
		return nil, err
	}
	token, err := newID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if initialConfig == "" {
		initialConfig = "[]"
	}

	_, err = s.db.Exec(
		`INSERT INTO users (id, phrase, token, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, phrase, token, initialConfig, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        id,
		Phrase:    phrase,
		Token:     token,
		Config:    initialConfig,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *Store) UserByPhrase(phrase string) (*User, error) {
	u := &User{}
	var ca, ua string
	err := s.db.QueryRow(
		`SELECT id, phrase, token, config, created_at, updated_at FROM users WHERE phrase = ?`,
		phrase,
	).Scan(&u.ID, &u.Phrase, &u.Token, &u.Config, &ca, &ua)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return u, nil
}

func (s *Store) UserByToken(token string) (*User, error) {
	u := &User{}
	var ca, ua string
	err := s.db.QueryRow(
		`SELECT id, phrase, token, config, created_at, updated_at FROM users WHERE token = ?`,
		token,
	).Scan(&u.ID, &u.Phrase, &u.Token, &u.Config, &ca, &ua)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return u, nil
}

func (s *Store) GetConfig(token string) (string, error) {
	var config string
	err := s.db.QueryRow(`SELECT config FROM users WHERE token = ?`, token).Scan(&config)
	if err != nil {
		return "", err
	}
	return config, nil
}

func (s *Store) SetConfig(token, config string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(`UPDATE users SET config = ?, updated_at = ? WHERE token = ?`, config, now, token)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteUser(token string) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE token = ?`, token)
	return err
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
