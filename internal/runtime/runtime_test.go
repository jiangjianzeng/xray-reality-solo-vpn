package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"xray-reality-solo-vpn/internal/config"
	"xray-reality-solo-vpn/internal/store"
)

func TestBuildMihomoConfig(t *testing.T) {
	manager := NewManager(&config.Config{
		LineServerAddress: "1.2.3.4",
		LinePublicPort:    443,
		XrayRealityNames:  []string{"www.cloudflare.com"},
		XrayPublicKey:     "public-key",
	}, nil)

	client := store.Client{
		Name:    "Alice's Mac",
		UUID:    "uuid-1",
		ShortID: "0123456789abcdef",
	}

	output := manager.BuildMihomoConfig(client)
	if !strings.Contains(output, "Alice''s Mac") {
		t.Fatalf("expected escaped client name, got %s", output)
	}
	if !strings.Contains(output, "public-key") {
		t.Fatalf("expected public key in output, got %s", output)
	}
}

func TestReadServiceStatusUsesSystemctl(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "systemctl.log")
	systemctlPath := filepath.Join(tempDir, "systemctl")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" >\"" + logPath + "\"\necho active\n"
	if err := os.WriteFile(systemctlPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake systemctl: %v", err)
	}

	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	manager := NewManager(&config.Config{
		XrayServiceName: "xray-custom",
	}, nil)

	status := manager.ReadServiceStatus(context.Background())
	if status.State != "active" {
		t.Fatalf("expected active state, got %#v", status)
	}
	if status.Message != "active" {
		t.Fatalf("expected active message, got %#v", status)
	}

	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake systemctl log: %v", err)
	}
	if string(logged) != "is-active\nxray-custom\n" {
		t.Fatalf("unexpected systemctl args: %q", string(logged))
	}
}

func TestRefreshTrafficStatsParsesNumericValuesAndUpdatesClient(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	systemctlPath := filepath.Join(tempDir, "systemctl")
	if err := os.WriteFile(systemctlPath, []byte("#!/bin/sh\necho active\n"), 0o755); err != nil {
		t.Fatalf("write fake systemctl: %v", err)
	}

	xrayPath := filepath.Join(tempDir, "xray")
	xrayOutput := `{"stat":[{"name":"user>>>alice-mac@example.com>>>traffic>>>uplink","value":1234},{"name":"user>>>alice-mac@example.com>>>traffic>>>downlink","value":5678}]}`
	if err := os.WriteFile(xrayPath, []byte("#!/bin/sh\nprintf '%s' '"+xrayOutput+"'\n"), 0o755); err != nil {
		t.Fatalf("write fake xray: %v", err)
	}

	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &config.Config{
		DataDir:         dataDir,
		LineDomain:      "example.com",
		XrayServiceName: "xray",
		XrayExecutable:  xrayPath,
		XrayAPIListenPort: 10085,
	}

	s, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	client, err := s.CreateClient(context.Background(), "Alice Mac")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	manager := NewManager(cfg, s)

	result := manager.RefreshTrafficStats(context.Background())
	if result.State != "running" {
		t.Fatalf("expected running result, got %#v", result)
	}

	updated, err := s.GetClientByID(context.Background(), client.ID)
	if err != nil {
		t.Fatalf("get client: %v", err)
	}
	if updated == nil {
		t.Fatal("expected client to exist")
	}
	if updated.RXBytes != 5678 || updated.TXBytes != 1234 {
		t.Fatalf("unexpected traffic totals: rx=%d tx=%d", updated.RXBytes, updated.TXBytes)
	}
	if updated.LastSeenAt == nil {
		t.Fatal("expected last seen to be updated")
	}
	if updated.RXBps <= 0 || updated.TXBps <= 0 {
		t.Fatalf("expected non-zero rates on first refresh, got rxBps=%d txBps=%d", updated.RXBps, updated.TXBps)
	}

	parsedLastSeen, err := time.Parse(time.RFC3339, *updated.LastSeenAt)
	if err != nil {
		t.Fatalf("parse last seen: %v", err)
	}
	if time.Since(parsedLastSeen) > time.Minute {
		t.Fatalf("expected recent last seen, got %s", parsedLastSeen)
	}
}
