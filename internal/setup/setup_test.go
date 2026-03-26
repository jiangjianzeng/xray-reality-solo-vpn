package setup

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"xray-reality-solo-vpn/internal/config"
)

func TestTicketLifecycle(t *testing.T) {
	manager := NewManager(&config.Config{
		SetupTicketFile: filepath.Join(t.TempDir(), "setup-ticket.json"),
		SetupTicketTTL:  15 * time.Minute,
	})

	ticket, err := manager.Create()
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	loaded, ok, err := manager.Validate(context.Background(), ticket.Token)
	if err != nil {
		t.Fatalf("validate ticket: %v", err)
	}
	if loaded == nil || !ok {
		t.Fatal("expected ticket to validate")
	}

	if err := manager.Consume(); err != nil {
		t.Fatalf("consume ticket: %v", err)
	}
	loaded, ok, err = manager.Validate(context.Background(), ticket.Token)
	if err != nil {
		t.Fatalf("validate after consume: %v", err)
	}
	if loaded != nil || ok {
		t.Fatal("expected ticket to be gone")
	}
}
