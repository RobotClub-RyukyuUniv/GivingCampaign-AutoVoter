package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Session はCross Campusログイン後に発行されるセッション。X-CC-Session-ID ヘッダで送る。
type Session struct {
	ID      string    `json:"session_id"`
	SavedAt time.Time `json:"saved_at"`
}

// SessionPath はセッションの保存先（~/.config/gcvote/session.json 等）。
func SessionPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gcvote", "session.json"), nil
}

func LoadSession() (*Session, error) {
	p, err := SessionPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil || s.ID == "" {
		return nil, nil
	}
	return &s, nil
}

// SaveSession はセッションを本人だけが読めるパーミッションで保存する。
func SaveSession(id string) error {
	p, err := SessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(Session{ID: id, SavedAt: time.Now()}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

func DeleteSession() error {
	p, err := SessionPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
