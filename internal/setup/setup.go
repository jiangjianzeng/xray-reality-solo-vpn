package setup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"xray-reality-solo-vpn/internal/config"
	"xray-reality-solo-vpn/internal/security"
)

type Ticket struct {
	Token     string  `json:"token"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt string  `json:"expires_at"`
	UsedAt    *string `json:"used_at,omitempty"`
}

type Manager struct {
	path string
	ttl  time.Duration
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		path: cfg.SetupTicketFile,
		ttl:  cfg.SetupTicketTTL,
	}
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) Create() (*Ticket, error) {
	token, err := security.RandomToken(32)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ticket := &Ticket{
		Token:     token,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(m.ttl).Format(time.RFC3339),
	}
	if err := m.write(ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (m *Manager) Load() (*Ticket, error) {
	bytes, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ticket Ticket
	if err := json.Unmarshal(bytes, &ticket); err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (m *Manager) Validate(_ context.Context, token string) (*Ticket, bool, error) {
	ticket, err := m.Load()
	if err != nil || ticket == nil {
		return ticket, false, err
	}
	if ticket.UsedAt != nil {
		return ticket, false, nil
	}
	expiresAt, err := time.Parse(time.RFC3339, ticket.ExpiresAt)
	if err != nil {
		return ticket, false, err
	}
	if time.Now().UTC().After(expiresAt) {
		return ticket, false, nil
	}
	return ticket, ticket.Token == token, nil
}

func (m *Manager) Consume() error {
	return os.Remove(m.path)
}

func (m *Manager) write(ticket *Ticket) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(ticket, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	return os.WriteFile(m.path, bytes, 0o600)
}
